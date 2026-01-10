package convert

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eric2788/bilirec/pkg/cloudconvert"
	"github.com/eric2788/bilirec/pkg/ds"
	"github.com/eric2788/bilirec/pkg/pool"
	"github.com/eric2788/bilirec/utils"
	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
	"golang.org/x/sync/semaphore"
)

const cloudConvertBucket = "Queue_CloudConvert"

var ErrCloudConvertNotConfigured = errors.New("cloudconvert client is not initialized")

type cloudConvertManager struct {
	db         *bbolt.DB
	logger     *logrus.Entry
	client     *cloudconvert.Client
	serializer *pool.Serializer

	downloading  ds.Set[string]
	downloadPool *pool.BytesPool
	concurrent   *semaphore.Weighted
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
	originalFormat := filepath.Ext(inputPath)[1:]
	res, err := c.client.VideoConvert(&cloudconvert.VideoConvertPayload{
		Input:        task.Data.ID,
		InputFormat:  originalFormat,
		OutputFormat: format,
		VideoCodec:   "copy",
		AudioCodec:   "copy",
		Filename:     filepath.Base(outputPath),
	})
	if err != nil {
		return nil, err
	}
	return &TaskQueue{
		TaskID:       res.Data.ID,
		InputPath:    inputPath,
		OutputPath:   outputPath,
		InputFormat:  originalFormat,
		OutputFormat: format,
	}, nil
}

func (c *cloudConvertManager) Cancel(taskID string) error {
	if err := c.client.CancelTask(taskID); err != nil {
		return err
	}
	return c.mutate(func(bucket *bbolt.Bucket) error {
		return bucket.Delete([]byte(taskID))
	})
}

func (c *cloudConvertManager) ListInProgress() ([]*TaskQueue, error) {
	var queues []*TaskQueue
	err := c.read(func(bucket *bbolt.Bucket) error {
		return bucket.ForEach(func(k, v []byte) error {
			var queue TaskQueue
			if err := c.serializer.Deserialize(v, &queue); err != nil {
				return fmt.Errorf("deserialize task %s: %w", string(k), err)
			}
			queues = append(queues, &queue)
			return nil
		})
	})
	return queues, err
}

func (c *cloudConvertManager) checkTaskStatusPeriodically(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.logger.Debugf("checking task queue...")
			if list, err := c.ListInProgress(); err != nil {
				c.logger.Errorf("failed to list in-progress tasks: %v", err)
			} else {
				for _, queue := range list {
					id := queue.TaskID

					if c.downloading.Contains(id) {
						c.logger.Debugf("task id=%v is downloading, skip status check", id)
						continue
					}

					c.logger.Debugf("checking task queue for id=%v", id)
					info, err := c.client.GetTask(queue.TaskID)
					if err != nil {
						c.logger.Errorf("failed to get task info for id=%v: %v", queue.TaskID, err)
						continue
					}

					c.logger.Infof("task id=%v status=%v", queue.TaskID, info.Data.Status)

					switch info.Data.Status {
					case cloudconvert.TaskStatusFinished:
						c.onFinished(queue, &info.Data)
					case cloudconvert.TaskStatusError:
						c.onFailed(queue, &info.Data)
					}
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (c *cloudConvertManager) onFinished(queue *TaskQueue, info *cloudconvert.TaskData) error {
	res, err := c.client.CreateExportURL(&cloudconvert.ExportURLRequest{
		Input: info.ID,
	})
	if err != nil {
		return err
	}
	// download file
	var download *cloudconvert.ExportedFile
	if len(res.Data.Result.Files) == 0 {
		return fmt.Errorf("no result files for task %s", queue.TaskID)
	} else if len(res.Data.Result.Files) == 1 {
		download = &res.Data.Result.Files[0]
	} else {
		c.logger.Warnf("multiple result files for task %s, will use smart detect", queue.TaskID)
		// check output format and compare output path from TAskQueue
		for _, file := range res.Data.Result.Files {
			format := utils.GetPathFormat(file.Filename)
			if filepath.Base(queue.OutputPath) == file.Filename {
				c.logger.Debugf("base(%s) == %s: matched filename", queue.OutputPath, file.Filename)
				download = &file
				break
			} else if format == queue.OutputFormat {
				c.logger.Debugf("format(%s) == %s: matched format", format, queue.OutputFormat)
				download = &file
				break
			}
		}
		if download == nil {
			logger.Debug("no matched filename or format, fallback to first file")
			download = &res.Data.Result.Files[0]
		}
	}

	go func() {
		c.downloading.Add(queue.TaskID)
		defer c.downloading.Remove(queue.TaskID)

		if err := c.downloadExportedFile(context.Background(), download.URL, queue.OutputPath); err != nil {
			c.logger.Errorf("failed to download exported file for task %s: %v", queue.TaskID, err)
			return
		}
		c.logger.Infof("successfully downloaded exported file for task %s to %s", queue.TaskID, queue.OutputPath)
		// delete here
		if err := c.mutate(func(bucket *bbolt.Bucket) error {
			return bucket.Delete([]byte(queue.TaskID))
		}); err != nil {
			c.logger.Errorf("failed to delete completed task %s from queue: %v", queue.TaskID, err)
		}
	}()

	return nil
}

func (c *cloudConvertManager) onFailed(queue *TaskQueue, info *cloudconvert.TaskData) {
	// print log and queue again
	c.logger.Errorf("task %s failed with message: %s", queue.TaskID, *info.Message)
	c.logger.Infof("re-enqueueing task %s", queue.TaskID)

	if info, err := c.Enqueue(queue.InputPath, queue.OutputPath, queue.OutputFormat); err != nil {
		c.logger.Errorf("failed to re-enqueue task %s: %v", queue.TaskID, err)
	} else {
		c.logger.Infof("successfully re-enqueued task %s as new task %s", queue.TaskID, info.TaskID)
	}

	if err := c.mutate(func(bucket *bbolt.Bucket) error {
		return bucket.Delete([]byte(queue.TaskID))
	}); err != nil {
		c.logger.Errorf("failed to delete failed task %s from queue: %v", queue.TaskID, err)
	}
}

func (c *cloudConvertManager) downloadExportedFile(ctx context.Context, url, outPath string) error {

	c.concurrent.Acquire(ctx, 1)
	defer c.concurrent.Release(1)

	// Open stream from CloudConvert client
	rc, err := c.client.DownloadAsFileStream(url)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	return utils.StreamToFile(ctx, rc, outPath, c.downloadPool)
}

func (c *cloudConvertManager) mutate(fn func(bucket *bbolt.Bucket) error) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(cloudConvertBucket))
		return fn(bucket)
	})
}

func (c *cloudConvertManager) read(fn func(bucket *bbolt.Bucket) error) error {
	return c.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(cloudConvertBucket))
		return fn(bucket)
	})
}
