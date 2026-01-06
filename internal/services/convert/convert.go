package convert

import (
	"context"
	"fmt"
	"os"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/pkg/cloudconvert"
	"github.com/eric2788/bilirec/utils"
	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
	"go.uber.org/fx"
)

var logger = logrus.WithField("service", "convert")

var ErrTaskNotFound = fmt.Errorf("convert task not found")

type Service struct {
	cloudthreshold int64
	managers       map[string]ConvertManager
	ctx            context.Context
	db             *bbolt.DB
}

func NewService(ls fx.Lifecycle, cfg *config.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &Service{
		cloudthreshold: cfg.CloudConvertThreshold,
		managers:       make(map[string]ConvertManager),
		ctx:            ctx,
	}

	if cfg.CloudConvertApiKey != "" {
		svc.managers["cloudconvert"] = newCloudConvertManager(
			cloudconvert.NewClient(ctx, cfg.CloudConvertApiKey),
		)
	} else {
		logger.Info("cloud convert api key not provided, cloud convert disabled")
	}

	ls.Append(fx.StartStopHook(
		func() error {
			if err := os.MkdirAll(cfg.DatabaseDir, 0755); err != nil {
				return err
			}
			// use bbolt for offline storage
			db, err := bbolt.Open(cfg.DatabaseDir+string(os.PathSeparator)+"queues.db", 0600, nil)
			if err != nil {
				return err
			}
			for _, manager := range svc.managers {
				if err := manager.StartWorker(ctx, db); err != nil {
					return fmt.Errorf("failed to start convert manager: %v", err)
				}
			}
			svc.db = db
			return nil
		},
		func() error {
			cancel()
			return svc.db.Close()
		},
	))
	return svc
}

func (s *Service) Enqueue(path, format string, deleteSource bool) (*TaskQueue, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	outputFormat := utils.ChangePathFormat(path, format)
	var manager ConvertManager
	if s.shoulduseCloudConvert(fileInfo.Size()) {
		manager = s.managers["cloudconvert"]
	} else {
		manager = s.managers["ffmpeg"]
	}
	return manager.Enqueue(path, outputFormat, format, deleteSource)
}

func (s *Service) Cancel(taskID string) error {
	for _, manager := range s.managers {
		if err := manager.Cancel(taskID); err == nil {
			return nil
		} else if err != ErrTaskNotFound {
			return err
		}
	}
	return fmt.Errorf("task %s not found in any convert manager", taskID)
}

func (s *Service) ListInProgress() ([]*TaskQueue, error) {
	var allQueues []*TaskQueue
	for _, manager := range s.managers {
		queues, err := manager.ListInProgress()
		if err != nil {
			return nil, err
		}
		allQueues = append(allQueues, queues...)
	}
	return allQueues, nil
}

func (s *Service) SetActiveRecordingsGetter(getter GetActiveRecordings) {
	if _, ok := s.managers["ffmpeg"]; ok {
		return
	}
	s.managers["ffmpeg"] = newFFmpegConvertManager(getter)
}

func (s *Service) shoulduseCloudConvert(fileSize int64) bool {
	_, cloudEnabled := s.managers["cloudconvert"]
	return cloudEnabled && s.cloudthreshold >= 0 && fileSize >= s.cloudthreshold
}
