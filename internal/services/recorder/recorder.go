package recorder

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/services/stream"
	"github.com/eric2788/bilirec/pkg/pipeline"
	"github.com/eric2788/bilirec/utils"
	"github.com/puzpuzpuz/xsync/v4"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

var logger = logrus.WithField("service", "recorder")

type RecordStatus string

const Recording RecordStatus = "recording"
const Recovering RecordStatus = "recovering"
const Idle RecordStatus = "idle"

//var idlePtr *RecordStatus = utils.Ptr(Idle)
var recordingPtr *RecordStatus = utils.Ptr(Recording)
var recoveringPtr *RecordStatus = utils.Ptr(Recovering)

var ErrMaxConcurrentRecordingsReached = fmt.Errorf("maximum concurrent recordings reached")
var ErrRecordingStarted = fmt.Errorf("recording already started")
var ErrStreamNotLive = fmt.Errorf("the room is not live streaming")
var ErrEmptyStreamURLs = fmt.Errorf("no stream urls available")
var ErrStreamURLsUnreachable = fmt.Errorf("all stream urls are unreachable")
var ErrMaxRecordingHoursReached = fmt.Errorf("maximum recording hours reached")

type Recorder struct {
	status         atomic.Pointer[RecordStatus]
	bytesRead      atomic.Uint64
	recoveredCount *xsync.Counter
	startTime      time.Time

	cancel context.CancelFunc
}

type Service struct {
	st        *stream.Service
	bilic     *bilibili.Client
	recording *xsync.Map[int64, *Recorder]
	pipes     *xsync.Map[int64, *pipeline.Pipe[[]byte]]

	cfg *config.Config
	ctx context.Context
}

func NewService(
	lc fx.Lifecycle,
	st *stream.Service,
	bilic *bilibili.Client,
	cfg *config.Config,
) *Service {

	ctx, cancel := context.WithCancel(context.Background())

	s := &Service{
		st:        st,
		bilic:     bilic,
		recording: xsync.NewMap[int64, *Recorder](),
		pipes:     xsync.NewMap[int64, *pipeline.Pipe[[]byte]](),
		cfg:       cfg,
		ctx:       ctx,
	}

	lc.Append(fx.StopHook(cancel))
	return s
}

func (r *Service) Start(roomId int64) error {

	l := logger.WithField("room", roomId)

	if status := r.GetStatus(roomId); status == Recording {
		return ErrRecordingStarted
	}

	if r.recording.Size() >= r.cfg.MaxConcurrentRecordings {
		return ErrMaxConcurrentRecordingsReached
	}

	isLive, err := r.bilic.IsStreamLiving(roomId)
	if err != nil {
		return fmt.Errorf("cannot check stream living status: %v", err)
	} else if !isLive {
		return ErrStreamNotLive
	}

	urls, err := r.bilic.GetStreamURLsV2(roomId)
	if err != nil {
		return err
	} else if len(urls) == 0 {
		return ErrEmptyStreamURLs
	}

	ctx, cancel := context.WithCancel(r.ctx)

	// retry mechanism
	for _, url := range urls {
		resp, err := r.bilic.FetchLiveStreamUrl(url)
		if err != nil {
			l.Errorf("cannot fetch url: %v, will try next url", err)
			continue
		}
		ch, err := r.st.ReadStream(resp, ctx)
		if err != nil {
			l.Errorf("cannot capture url stream: %v, will try next url", err)
			continue
		}

		return r.prepare(roomId, ch, ctx, cancel)
	}
	cancel()
	l.Warn("no more url left")
	return ErrStreamURLsUnreachable
}

func (r *Service) Stop(roomId int64) bool {

	info, hasRecording := r.recording.LoadAndDelete(roomId)
	pipe, hasPipe := r.pipes.LoadAndDelete(roomId)

	if hasRecording {
		info.cancel()
	} else {
		logger.Warnf("recording for room %d not found", roomId)
	}

	if hasPipe && !hasRecording {
		logger.Warnf("found orphaned pipe from room %d, closing it...", roomId)
		pipe.Close()
	}

	return hasRecording
}

func (r *Service) prepare(roomId int64, ch <-chan []byte, ctx context.Context, cancel context.CancelFunc) error {

	// initialize Recorder info
	info := &Recorder{
		cancel:         cancel,
		recoveredCount: xsync.NewCounter(),
		startTime:      time.Now(),
	}
	info.status.Store(recordingPtr)

	// initialize pipeline
	pipe, err := r.newStreamPipeline(roomId, info)
	if err != nil {
		return fmt.Errorf("cannot create pipeline: %v", err)
	}

	startCtx, startCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := pipe.Open(startCtx); err != nil {
		startCancel()
		return fmt.Errorf("cannot open pipeline: %v", err)
	}
	startCancel()

	r.recording.Store(roomId, info)
	r.pipes.Store(roomId, pipe)

	go r.rev(roomId, ch, info, pipe)
	go r.checkRecordingDurationPeriodically(roomId, ctx)
	return nil
}

func (r *Service) rev(roomId int64, ch <-chan []byte, info *Recorder, pipe *pipeline.Pipe[[]byte]) {
	l := logger.WithField("room", roomId)
	defer r.recover(roomId)
	defer func() {
		pipe.Close()
		go r.finalize(roomId, info)
	}()
	for data := range ch {

		info.bytesRead.Add(uint64(len(data)))
		_, err := pipe.Process(r.ctx, data)
		r.st.Flush(data)

		if err != nil {
			l.Errorf("error writing data to file: %v", err)
			return
		}
	}
}

func (r *Service) checkRecordingDurationPeriodically(roomId int64, ctx context.Context) {
	log := logger.WithField("room", roomId)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	maxDuration := time.Duration(r.cfg.MaxRecordingHours) * time.Hour
	for {
		select {
		case <-ticker.C:
			info, ok := r.recording.Load(roomId)
			if !ok {
				return
			}
			elapsed := time.Since(info.startTime)
			if elapsed >= maxDuration {
				log.Infof("maximum recording hours reached (%v), stopping", elapsed.Round(time.Minute))
				r.Stop(roomId)
				return
			}

			if int(elapsed.Minutes())%30 == 0 {
				remaining := maxDuration - elapsed
				log.Infof("recording: %v elapsed, %v remaining, %d MB", elapsed.Round(time.Minute), remaining.Round(time.Minute), info.bytesRead.Load()/1024/1024)
			}

		case <-ctx.Done():
			return
		}
	}
}

func (r *Service) recover(roomId int64) {
	l := logger.WithField("room", roomId)
	l.Infof("trying to recover stream capture...")
	info, ok := r.recording.Load(roomId)
	if !ok {
		l.Infof("recording stopped manually, skipped.")
		return
	} else if status := info.status.Load(); status == recoveringPtr {
		l.Infof("stream is recovering, skipped.")
		return
	} else if info.recoveredCount.Value() >= int64(r.cfg.MaxRecoveryAttempts) {
		l.Infof("maximum recovery attempts reached (%d), stopping recording", r.cfg.MaxRecoveryAttempts)
		return
	}
	info.recoveredCount.Inc()
	info.status.Store(recoveringPtr)
	err := r.Start(roomId)
	if err != nil {

		retry := func() {
			timer := time.NewTimer(15 * time.Second)
			defer timer.Stop()
			select {
			case <-timer.C:
				r.recover(roomId)
			case <-r.ctx.Done():
				return
			}
		}

		l.Errorf("cannot recover stream capture: %v", err)
		switch err {
		case ErrMaxRecordingHoursReached, ErrMaxConcurrentRecordingsReached:
			l.Infof("stop recovery due to: %v", err)
		case ErrStreamNotLive:
			l.Infof("stream is offline, will not recover.")
		default:
			l.Infof("will retry stream recovery in 15 seconds...")
			retry()
		}
		return
	}
	l.Info("start live stream recovery: success")
}

func (r *Service) finalize(roomId int64, info *Recorder) {
	if info == nil {
		logger.Warnf("skipping finalize for room %d: no recording info", roomId)
		return
	}
	finalPipe, err := r.newFinalPipeline()
	if err != nil {
		logger.Errorf("cannot create final pipeline for room %d: %v", roomId, err)
		return
	}
	if err := finalPipe.Open(r.ctx); err != nil {
		logger.Errorf("cannot open final pipeline for room %d: %v", roomId, err)
		return
	}
	defer finalPipe.Close()

	logger.Infof("finalize info: %v", info)

	dirPath := fmt.Sprintf("%s/%d", r.cfg.OutputDir, roomId)
	filename := fmt.Sprintf("%s/%d.flv", dirPath, info.startTime.Unix())
	output, err := finalPipe.Process(r.ctx, filename)
	if err != nil {
		logger.Errorf("cannot process final pipeline for room %d: %v", roomId, err)
		return
	}
	logger.Infof("finalized recording for room %d: %s", roomId, output)
}
