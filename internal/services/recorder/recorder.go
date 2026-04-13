package recorder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/processors"
	"github.com/eric2788/bilirec/internal/services/convert"
	"github.com/eric2788/bilirec/internal/services/stream"
	"github.com/eric2788/bilirec/pkg/ds"
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

// var idlePtr *RecordStatus = utils.Ptr(Idle)
var recordingPtr *RecordStatus = utils.Ptr(Recording)
var recoveringPtr *RecordStatus = utils.Ptr(Recovering)

var ErrMaxConcurrentRecordingsReached = errors.New("maximum concurrent recordings reached")
var ErrRecordingStarted = errors.New("recording already started")
var ErrStreamNotLive = errors.New("the room is not live streaming")
var ErrEmptyStreamURLs = errors.New("no stream urls available")
var ErrStreamURLsUnreachable = errors.New("all stream urls are unreachable")
var ErrRoomLocked = errors.New("the room is locked")
var ErrRoomEncrypted = errors.New("the room is encrypted")
var ErrInsufficientDiskSpace = errors.New("insufficient disk space")

type Recorder struct {
	status       atomic.Pointer[RecordStatus]
	bytesRead    atomic.Uint64
	segmentBytes atomic.Uint64
	startTime    time.Time
	outputPath   string
	mu           sync.RWMutex

	cancel context.CancelFunc
}

func (r *Recorder) GetOutputPath() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.outputPath
}

func (r *Recorder) SetOutputPath(path string) {
	r.mu.Lock()
	r.outputPath = path
	r.mu.Unlock()
}

type Service struct {
	st            *stream.Service
	cv            *convert.Service
	bilic         *bilibili.Client
	recording     *xsync.Map[int, *Recorder]
	writtingFiles ds.Set[string]
	pipes         *xsync.Map[int, *pipeline.Pipe[[]byte]]

	cfg *config.Config
	ctx context.Context
}

func NewService(
	lc fx.Lifecycle,
	st *stream.Service,
	cv *convert.Service,
	bilic *bilibili.Client,
	cfg *config.Config,
) *Service {

	ctx, cancel := context.WithCancel(context.Background())

	s := &Service{
		st:            st,
		cv:            cv,
		bilic:         bilic,
		recording:     xsync.NewMap[int, *Recorder](),
		writtingFiles: ds.NewSyncedSet[string](),
		pipes:         xsync.NewMap[int, *pipeline.Pipe[[]byte]](),
		cfg:           cfg,
		ctx:           ctx,
	}

	cv.SetActiveRecordingsGetter(s.recording.Size)

	go s.backgroundMaintenance(ctx)
	go initOutputDir(cfg)

	lc.Append(fx.StopHook(cancel))
	return s
}

func (r *Service) Start(roomId int) error {

	l := logger.WithField("room", roomId)

	if status := r.GetStatus(roomId); status == Recording {
		return ErrRecordingStarted
	}

	if r.recording.Size() >= r.cfg.MaxConcurrentRecordings {
		if existing, ok := r.recording.Load(roomId); !ok { // if not recovering existing recording
			return ErrMaxConcurrentRecordingsReached
		} else if status := existing.status.Load(); status != recoveringPtr { // not recovering
			return ErrMaxConcurrentRecordingsReached
		}
	}

	// Check disk space - require at least configured minimum free space
	diskSpace, err := utils.GetDiskSpace(r.cfg.OutputDir)
	if err != nil {
		l.Warnf("cannot check disk space: %v", err)
	} else if diskSpace.Free < uint64(r.cfg.MinDiskSpaceBytes) {
		return ErrInsufficientDiskSpace
	}

	roomInfo, err := r.bilic.GetLiveRoomInfo(roomId)
	if err != nil {
		return err
	} else if roomInfo.IsEncrypted {
		return ErrRoomEncrypted
	} else if roomInfo.LockStatus != 0 {
		return ErrRoomLocked
	} else if roomInfo.LiveStatus != 1 {
		return ErrStreamNotLive
	}

	urls, err := r.bilic.GetStreamURLsV2(roomId)
	if err != nil {
		return err
	} else if len(urls) == 0 {
		return ErrEmptyStreamURLs
	}

	now := time.Now()

	outputPath, err := r.prepareFilePath(roomInfo, now)
	if err != nil {
		return fmt.Errorf("cannot prepare file path: %v", err)
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

		// initialize Recorder info
		info := &Recorder{
			cancel:     cancel,
			startTime:  now,
			outputPath: outputPath,
		}
		info.status.Store(recordingPtr)
		r.writtingFiles.Add(filepath.Base(outputPath))

		return r.prepare(roomId, ch, ctx, info)
	}
	cancel()
	l.Warn("no more url left")
	return ErrStreamURLsUnreachable
}

func (r *Service) Stop(roomId int) bool {

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

func (r *Service) prepare(roomId int, ch <-chan []byte, ctx context.Context, info *Recorder) error {

	pipe, err := r.newPipe(info.GetOutputPath(), ctx)
	if err != nil {
		return err
	}

	r.recording.Store(roomId, info)
	r.pipes.Store(roomId, pipe)

	go r.rev(roomId, ch, ctx, info, pipe)
	go r.checkRecordingDurationPeriodically(roomId, ctx)
	return nil
}

func (r *Service) newPipe(outputPath string, ctx context.Context) (*pipeline.Pipe[[]byte], error) {
	pipe := pipeline.New(
		// fix FLV stream
		processors.NewFlvStreamFixer(),
		// write to file with buffered writer
		// flushes every 5 seconds then writes to disk
		processors.NewBufferedStreamWriter(outputPath, config.ReadOnly.LiveStreamWriterBufferSize()),
	)

	startCtx, startCancel := context.WithTimeout(ctx, 10*time.Second)
	if err := pipe.Open(startCtx); err != nil {
		startCancel()
		return nil, fmt.Errorf("cannot open pipeline: %v", err)
	}
	startCancel()
	return pipe, nil
}

func (r *Service) rev(roomId int, ch <-chan []byte, ctx context.Context, info *Recorder, pipe *pipeline.Pipe[[]byte]) {
	l := logger.WithField("room", roomId)
	defer r.recover(roomId)
	currentPipe := pipe
	defer func() {
		currentPipe.Close()
		go r.finalize(roomId, info)
	}()
	for data := range ch {

		info.bytesRead.Add(uint64(len(data)))
		info.segmentBytes.Add(uint64(len(data)))
		result, err := currentPipe.Process(r.ctx, data)
		r.st.Flush(data)
		r.st.Flush(result)

		if err != nil {
			l.Errorf("error writing data to file: %v", err)
			if err == processors.ErrNotFlvFile {
				l.Warn("received FLV validation errors, stream may be unstable")
				timer := time.NewTimer(5 * time.Second)
				select {
				case <-timer.C:
					return
				case <-r.ctx.Done():
					timer.Stop()
					return
				}
			}
			return
		}

		if r.shouldRotateSegment(info) {
			if err := r.rotateSegment(roomId, info, ctx, &currentPipe); err != nil {
				l.Errorf("failed to rotate segment by size: %v", err)
			}
		}
	}
}

func (r *Service) shouldRotateSegment(info *Recorder) bool {
	maxSize := r.cfg.MaxRecordingFileSizeBytes
	return maxSize > 0 && info.segmentBytes.Load() >= uint64(maxSize)
}

func (r *Service) nextSegmentOutputPath(currentPath string, now time.Time) string {
	dir := filepath.Dir(currentPath)
	ext := filepath.Ext(currentPath)
	baseWithoutExt := strings.TrimSuffix(filepath.Base(currentPath), ext)

	prefix := baseWithoutExt
	if idx := strings.LastIndex(baseWithoutExt, "-"); idx > 0 {
		prefix = baseWithoutExt[:idx]
	}

	timestamp := now.Format("20060102_150405")
	candidate := filepath.Join(dir, fmt.Sprintf("%s-%s%s", prefix, timestamp, ext))

	if candidate != currentPath && !r.writtingFiles.Contains(filepath.Base(candidate)) {
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}

	for suffix := 1; ; suffix++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s-%s-%d%s", prefix, timestamp, suffix, ext))
		if r.writtingFiles.Contains(filepath.Base(candidate)) {
			continue
		}
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}

func (r *Service) rotateSegment(roomId int, info *Recorder, ctx context.Context, pipe **pipeline.Pipe[[]byte]) error {
	oldPipe := *pipe
	oldPath := info.GetOutputPath()
	nextPath := r.nextSegmentOutputPath(oldPath, time.Now())

	nextPipe, err := r.newPipe(nextPath, ctx)
	if err != nil {
		return err
	}

	r.writtingFiles.Add(filepath.Base(nextPath))
	info.SetOutputPath(nextPath)
	info.segmentBytes.Store(0)

	r.pipes.Store(roomId, nextPipe)
	*pipe = nextPipe

	oldPipe.Close()
	go r.finalize(roomId, &Recorder{outputPath: oldPath})
	return nil
}

func (r *Service) checkRecordingDurationPeriodically(roomId int, ctx context.Context) {
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

// Note: Each recovery attempt creates a NEW file with a new timestamp.
// This is intentional - we want separate files for each recording segment
// rather than appending to the same file. Multiple files per session is expected.
func (r *Service) recover(roomId int) {
	l := logger.WithField("room", roomId)
	l.Infof("trying to recover stream capture...")
	info, ok := r.recording.Load(roomId)
	if !ok {
		l.Infof("recording stopped manually, skipped.")
		return
	} else if status := info.status.Load(); status == recoveringPtr {
		l.Infof("stream is recovering, skipped.")
		return
	}

	info.status.Store(recoveringPtr)
	for attempt := 1; attempt <= r.cfg.MaxRecoveryAttempts; attempt++ {
		err := r.Start(roomId)
		if err == nil {
			l.Info("start live stream recovery: success")
			return
		}
		l.Errorf("recovery attempt #%d failed: %v", attempt, err)
		switch err {
		case ErrMaxConcurrentRecordingsReached:
			l.Infof("stop recovery due to: %v", err)
			r.Stop(roomId)
			return
		case ErrStreamNotLive, ErrRoomEncrypted, ErrRoomLocked:
			l.Infof("stream is offline, will not recover.")
			r.Stop(roomId)
			return
		default:
			// Should check if recording was manually stopped
			if _, ok := r.recording.Load(roomId); !ok {
				l.Infof("recording removed during retry, will not recover.")
				return
			}

			if attempt < r.cfg.MaxRecoveryAttempts {
				l.Infof("will retry stream recovery in 15 seconds...")
				timer := time.NewTimer(15 * time.Second)
				select {
				case <-timer.C:
					continue
				case <-r.ctx.Done():
					l.Infof("service is stopping, aborting recovery")
					timer.Stop()
					return
				}
			}
		}
	}

	l.Infof("maximum recovery attempts reached (%d), stopping recording", r.cfg.MaxRecoveryAttempts)
	r.Stop(roomId)
}

func (r *Service) finalize(roomId int, info *Recorder) {
	if info == nil {
		logger.Warnf("skipping finalize for room %d: no recording info", roomId)
		return
	}

	outputPath := info.GetOutputPath()
	defer r.writtingFiles.Remove(filepath.Base(outputPath))

	fileInfo, err := os.Stat(outputPath)
	if err != nil {
		logger.Errorf("failed to stat recorded file for room %d: %v", roomId, err)
		return
	} else if fileInfo.Size() < 1024 { // less than 1KB
		logger.Warnf("recorded file for room %d is too small (%d bytes), skipping finallization and removing file", roomId, fileInfo.Size())
		if err := os.Remove(outputPath); err != nil {
			logger.Errorf("failed to remove empty file %s: %v", outputPath, err)
		}
		return
	}

	if !r.cfg.ConvertFLVToMp4 {
		logger.Debug("no need to convert flv to mp4, skipped")
		return
	}

	// process finalization via convert service
	if queue, err := r.cv.Enqueue(outputPath, "mp4", r.cfg.DeleteFlvAfterConvert); err != nil {
		logger.Errorf("failed to enqueue conversion for room %d: %v", roomId, err)
		logger.Warnf("you may need to convert mp4 manually for room: %d", roomId)
	} else {
		logger.Infof("enqueued convertion for room %d: %s", roomId, queue.TaskID)
		logger.Infof("the output path will be: %s", queue.OutputPath)
	}
}

func (r *Service) backgroundMaintenance(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	lastActiveCount := 0

	for {
		select {
		case <-ticker.C:
			activeCount := r.recording.Size()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			if activeCount == 0 && lastActiveCount > 0 {
				// Just transitioned from active to idle - cleanup
				logger.Info("No ongoing recordings, performing maintenance GC")
				runtime.GC()
				debug.FreeOSMemory()

				runtime.ReadMemStats(&m)
				logger.Infof("After cleanup: Alloc=%d MB, Sys=%d MB",
					m.Alloc/1024/1024, m.Sys/1024/1024)
			} else if activeCount == 0 {
				// Still idle - just log
				logger.Debugf("Idle: Memory: Alloc=%d MB, Sys=%d MB, NumGC=%d",
					m.Alloc/1024/1024, m.Sys/1024/1024, m.NumGC)
			} else {
				// Recordings active - just log stats
				logger.Debugf("Active recordings: %d, Memory: Alloc=%d MB, Sys=%d MB, NumGC=%d",
					activeCount, m.Alloc/1024/1024, m.Sys/1024/1024, m.NumGC)
			}

			lastActiveCount = activeCount

		case <-ctx.Done():
			return
		}
	}
}

// the time should be the time you start the record, not live start
func (r *Service) prepareFilePath(info *bilibili.LiveRoomInfoDetail, start time.Time) (string, error) {
	dirPath := fmt.Sprintf("%s/%s-%d", r.cfg.OutputDir, info.Uname, info.RoomID)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", err
	}
	safeTitle := utils.TruncateString(utils.SanitizeFilename(info.Title), 20)
	return fmt.Sprintf("%s/%s-%s.flv", dirPath, safeTitle, start.Format("20060102_150405")), nil
}

func initOutputDir(cfg *config.Config) {
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		logger.Fatalf("cannot create output directory %s: %v", cfg.OutputDir, err)
	}
}
