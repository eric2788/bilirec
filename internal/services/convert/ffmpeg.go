package convert

import (
	"context"
	"fmt"
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

func (f *ffmpegConvertManager) StartWorker(ctx context.Context, db *bbolt.DB) error {
	if err := exec.CommandContext(ctx, "ffmpeg", "-h").Run(); err != nil {
		return err
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

func (f *ffmpegConvertManager) Enqueue(inputPath string, outputPath string, format string) (*TaskQueue, error) {
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
	ticker := time.NewTicker(3 * time.Minute)
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
			if err := f.mutate(func(bucket *bbolt.Bucket) error {
				k, v := bucket.Cursor().First()
				if k == nil {
					f.logger.Debug("no ffmpeg tasks in queue")
					return nil
				}

				id := string(k)
				var queue TaskQueue
				if err := f.serializer.Deserialize(v, &queue); err != nil {
					return fmt.Errorf("deserialize task %s: %w", id, err)
				}

				taskLog := f.logger.WithField("task_id", id)
				taskLog.Infof("processing ffmpeg task input=%s output=%s", queue.InputPath, queue.OutputPath)

				if err := f.processTask(ctx, &queue, taskLog); err != nil {
					return fmt.Errorf("process task %s: %w", id, err)
				}

				if err := bucket.Delete(k); err != nil {
					return fmt.Errorf("delete task %s: %w", id, err)
				}

				taskLog.Info("completed and removed from queue")
				return nil
			}); err != nil {
				f.logger.WithError(err).Error("processing ffmpeg queue task failed")
			}
		case <-ctx.Done():
			return
		}
	}
}

func (f *ffmpegConvertManager) processTask(ctx context.Context, queue *TaskQueue, taskLog *logrus.Entry) error {

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

	return cmd.Run()
}

func (f *ffmpegConvertManager) mutate(fn func(bucket *bbolt.Bucket) error) error {
	return f.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ffmpegBucket))
		return fn(bucket)
	})
}

func (f *ffmpegConvertManager) read(fn func(bucket *bbolt.Bucket) error) error {
	return f.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(ffmpegBucket))
		return fn(bucket)
	})
}
