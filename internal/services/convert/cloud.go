package convert

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/eric2788/bilirec/pkg/cloudconvert"
	"github.com/eric2788/bilirec/pkg/pool"
	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
)

const cloudConvertBucket = "Queue_CloudConvert"

var ErrCloudConvertNotConfigured = errors.New("cloudconvert client is not initialized")

type cloudConvertManager struct {
	db         *bbolt.DB
	logger     *logrus.Entry
	client     *cloudconvert.Client
	serializer *pool.Serializer
}

func (c *cloudConvertManager) StartWorker(ctx context.Context, db *bbolt.DB) error {
	if c.client == nil {
		return ErrCloudConvertNotConfigured
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(cloudConvertBucket))
		return err
	}); err != nil {
		return err
	}
	c.db = db
	go c.checkTaskStatusPeriodically(ctx)
	return nil
}

func (c *cloudConvertManager) Enqueue(inputPath string, outputPath string, format string) (*TaskQueue, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	task, err := c.client.CreateUploadTask()
	if err != nil {
		return nil, err
	}
	if err := c.client.UploadFileToTask(file, &task.Data); err != nil {
		return nil, err
	}
	res, err := c.client.VideoConvert(&cloudconvert.VideoConvertPayload{
		Input:        task.Data.ID,
		InputFormat:  filepath.Ext(inputPath)[1:],
		OutputFormat: format,
		VideoCodec:   "copy",
		AudioCodec:   "copy",
		Filename:     filepath.Base(outputPath),
	})
	if err != nil {
		return nil, err
	}
	return &TaskQueue{
		TaskID:     res.Data.ID,
		InputPath:  inputPath,
		OutputPath: outputPath,
		Format:     format,
	}, nil
}

func (c *cloudConvertManager) Cancel(taskID string) error {
	panic("not implemented") // TODO: Implement
}

func (c *cloudConvertManager) ListInProgress() ([]*TaskQueue, error) {
	panic("not implemented") // TODO: Implement
}

func (c *cloudConvertManager) checkTaskStatusPeriodically(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.logger.Debugf("checking task queue...")
			c.bucketUpdate(func(b *bbolt.Bucket) error {
				cu := b.Cursor()
				for k, v := cu.First(); k != nil; k, v = cu.Next() {

					var queue TaskQueue
					if err := c.serializer.Deserialize(v, &queue); err != nil {
						c.logger.Errorf("failed to deserialize task queue id=%v: %v", string(k), err)
						continue
					} else if queue.TaskID != string(k) {
						c.logger.Warnf("task queue id mismatch: key=%v, taskID=%v", string(k), queue.TaskID)
					}

					c.logger.Debugf("checking task queue for id=%v", string(k))
					info, err := c.client.GetTask(queue.TaskID)
					if err != nil {
						c.logger.Errorf("failed to get task info for id=%v: %v", queue.TaskID, err)
						continue
					}

					c.logger.Infof("task id=%v status=%v", queue.TaskID, info.Data.Status)

					if info.Data.Status == cloudconvert.TaskStatusFinished {
						// do finish work
					} else if info.Data.Status == cloudconvert.TaskStatusError {
						// do enqueue again
					}

				}
				return nil
			})
		case <-ctx.Done():
			return
		}
	}
}

func (c *cloudConvertManager) bucketUpdate(fn func(bucket *bbolt.Bucket) error) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(cloudConvertBucket))
		return fn(bucket)
	})
}

func (c *cloudConvertManager) bucketView(fn func(bucket *bbolt.Bucket) error) error {
	return c.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(cloudConvertBucket))
		return fn(bucket)
	})
}
