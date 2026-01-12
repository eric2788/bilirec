package convert

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/eric2788/bilirec/pkg/pool"
	"github.com/eric2788/bilirec/utils"
	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
)

const ffmpegBucket = "Queue_FFmpeg"

type ffmpegConvertManager struct {
	db         *bbolt.DB
	logger     *logrus.Entry
	serializer *pool.Serializer
	getActives GetActiveRecordings
}

func newFFmpegConvertManager(getActives GetActiveRecordings) ConvertManager {
	return &ffmpegConvertManager{
		logger:     logger.WithField("manager", "ffmpeg"),
		serializer: pool.NewSerializer(),
		getActives: getActives,
	}
}

func (f *ffmpegConvertManager) StartWorker(ctx context.Context, db *bbolt.DB) error {
	if !utils.FFmpegAvailable() {
		return ErrCloudConvertNotConfigured
	} else if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(ffmpegBucket))
		return err
	}); err != nil {
		return err
	}
	f.db = db
	go f.runTaskPeriodically(ctx)
	return nil
}

func (f *ffmpegConvertManager) Enqueue(inputPath, outputPath, format string, deleteSource bool) (*TaskQueue, error) {
	uuid, err := utils.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	queue := &TaskQueue{
		TaskID:       uuid,
		InputPath:    inputPath,
		OutputPath:   outputPath,
		InputFormat:  utils.GetPathFormat(inputPath),
		OutputFormat: format,
		DeleteSource: deleteSource,
	}
	err = f.mutate(func(bucket *bbolt.Bucket) error {
		data, err := f.serializer.Serialize(queue)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(uuid), data)
	})
	return queue, err
}

func (f *ffmpegConvertManager) Cancel(taskID string) error {
	return f.mutate(func(bucket *bbolt.Bucket) error {
		if bucket.Get([]byte(taskID)) == nil {
			return ErrTaskNotFound
		}
		return bucket.Delete([]byte(taskID))
	})
}

func (f *ffmpegConvertManager) ListInProgress() ([]*TaskQueue, error) {
	var queues []*TaskQueue
	err := f.read(func(bucket *bbolt.Bucket) error {
		return bucket.ForEach(func(k, v []byte) error {
			var queue TaskQueue
			if err := f.serializer.Deserialize(v, &queue); err != nil {
				return fmt.Errorf("deserialize task %s: %w", string(k), err)
			}
			queues = append(queues, &queue)
			return nil
		})
	})
	return queues, err
}

func (f *ffmpegConvertManager) runTaskPeriodically(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			actives := f.getActives()
			if actives > 0 {
				f.logger.Debugf("active recordings detected (%d), skipping ffmpeg tasks", actives)
				continue
			}
			f.logger.Debug("checking ffmpeg task queue")
			var queue *TaskQueue
			if err := f.read(func(bucket *bbolt.Bucket) error {
				k, v := bucket.Cursor().First()
				if k == nil {
					return nil
				}
				queue = &TaskQueue{}
				if err := f.serializer.Deserialize(v, queue); err != nil {
					return fmt.Errorf("deserialize task %s: %w", string(k), err)
				}
				return nil
			}); err != nil {
				f.logger.Errorf("reading ffmpeg queue task failed: %v", err)
				continue
			} else if queue == nil {
				f.logger.Debug("no ffmpeg tasks in queue")
				continue
			}

			taskLog := f.logger.WithField("task_id", queue.TaskID)
			taskLog.Infof("processing ffmpeg task input=%s output=%s", queue.InputPath, queue.OutputPath)

			if err := f.processTask(ctx, queue, taskLog); err != nil {
				taskLog.Errorf("ffmpeg task failed: %v", err)
				continue
			}

			if err := utils.WithRetry(3, taskLog, "delete bucket", func() error {
				return f.mutate(func(bucket *bbolt.Bucket) error {
					return bucket.Delete([]byte(queue.TaskID))
				})
			}); err != nil {
				taskLog.Errorf("failed to remove ffmpeg task from queue: %v", err)
				continue
			}

			taskLog.Info("completed and removed from queue")
		case <-ctx.Done():
			return
		}
	}
}

func (f *ffmpegConvertManager) processTask(ctx context.Context, queue *TaskQueue, taskLog *logrus.Entry) error {

	if utils.IsFileExists(queue.OutputPath) {
		taskLog.Warnf("output file %s already exists, skipping conversion", queue.OutputPath)
		return nil
	}

	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-hide_banner",
		"-i",
		queue.InputPath,
		"-c",
		"copy",
		queue.OutputPath,
	)

	cmd.Stdout = taskLog.Writer()
	cmd.Stderr = taskLog.Writer()

	if err := cmd.Run(); err != nil {
		return err
	} else if !queue.DeleteSource || queue.InputPath == queue.OutputPath {
		return nil
	}

	return utils.WithRetry(3, taskLog, "delete source file", func() error {
		if !utils.IsFileExists(queue.InputPath) {
			taskLog.Debugf("source file %s does not exist, skipping delete", queue.InputPath)
			return nil
		}
		return os.Remove(queue.InputPath)
	})
}

func (f *ffmpegConvertManager) mutate(fn func(bucket *bbolt.Bucket) error) error {
	return f.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ffmpegBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", ffmpegBucket)
		}
		return fn(bucket)
	})
}

func (f *ffmpegConvertManager) read(fn func(bucket *bbolt.Bucket) error) error {
	return f.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ffmpegBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", ffmpegBucket)
		}
		return fn(bucket)
	})
}
