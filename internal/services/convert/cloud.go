package convert

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/pkg/cloudconvert"
	"github.com/eric2788/bilirec/pkg/ds"
	"github.com/eric2788/bilirec/pkg/pool"
	"github.com/eric2788/bilirec/utils"
	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
	"golang.org/x/sync/semaphore"
)

const cloudConvertBucket = "Queue_CloudConvert"

type cloudConvertManager struct {
	db         *bbolt.DB
	logger     *logrus.Entry
	client     *cloudconvert.Client
	serializer *pool.Serializer

	downloading  ds.Set[string]
	downloadPool *pool.BytesPool
	concurrent   *semaphore.Weighted
}

func newCloudConvertManager(client *cloudconvert.Client) ConvertManager {
	return &cloudConvertManager{
		logger:       logger.WithField("manager", "cloudconvert"),
		client:       client,
		serializer:   pool.NewSerializer(),
		downloading:  ds.NewSyncedSet[string](),
		downloadPool: pool.NewBytesPool(config.ReadOnly.DownloadBufferSize()),
		concurrent:   semaphore.NewWeighted(2),
	}
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

func (c *cloudConvertManager) Enqueue(inputPath, outputPath, format string, deleteSource bool) (*TaskQueue, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	task, err := c.client.CreateUploadTask()
	if err != nil {
		return nil, err
	} else if err := c.client.UploadFileToTask(file, &task.Data); err != nil {
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
	// do export here and get that task ID
	exporter, err := c.client.CreateExportURL(&cloudconvert.ExportURLRequest{
		Input: res.Data.ID,
	})
	if err != nil {
		return nil, err
	}

	queue := &TaskQueue{
		TaskID:       exporter.Data.ID,
		InputPath:    inputPath,
		OutputPath:   outputPath,
		InputFormat:  originalFormat,
		OutputFormat: format,
		DeleteSource: deleteSource,
	}

	err = c.mutate(func(bucket *bbolt.Bucket) error {
		data, err := c.serializer.Serialize(queue)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(queue.TaskID), data)
	})

	return queue, err
}

func (c *cloudConvertManager) Cancel(taskID string) error {
	if exist, err := c.client.CancelTask(taskID); err != nil {
		return err
	} else if !exist {
		return ErrTaskNotFound
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
						err = c.onFinished(ctx, queue, &info.Data)
					case cloudconvert.TaskStatusError:
						err = c.onFailed(queue, &info.Data)
					}

					if err != nil {
						c.logger.Errorf("handling task id=%v status=%v failed: %v", queue.TaskID, info.Data.Status, err)
					}
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (c *cloudConvertManager) onFinished(ctx context.Context, queue *TaskQueue, data *cloudconvert.TaskData) error {
	// download file
	var download *cloudconvert.TaskResultFile
	if len(data.Result.Files) == 0 {
		return fmt.Errorf("no result files for task %s", queue.TaskID)
	} else if len(data.Result.Files) == 1 {
		download = &data.Result.Files[0]
	} else {
		c.logger.Warnf("multiple result files for task %s, will use smart detect", queue.TaskID)
		// check output format and compare output path from TAskQueue
		for _, file := range data.Result.Files {
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
			download = &data.Result.Files[0]
		}
	}

	c.downloading.Add(queue.TaskID)
	defer c.downloading.Remove(queue.TaskID)

	if err := c.downloadExportedFile(ctx, download.URL, queue.OutputPath); err != nil {
		c.logger.Errorf("failed to download exported file for task %s: %v", queue.TaskID, err)
		return err
	}

	c.logger.Infof("successfully downloaded exported file for task %s to %s", queue.TaskID, queue.OutputPath)

	err := utils.WithRetry(3, c.logger, "delete bucket", func() error {
		return c.mutate(func(bucket *bbolt.Bucket) error {
			return bucket.Delete([]byte(queue.TaskID))
		})
	})
	if err != nil {
		return err
	} else if !queue.DeleteSource || queue.InputPath == queue.OutputPath {
		return nil
	}

	return utils.WithRetry(3, c.logger, "delete source file", func() error {
		if !utils.IsFileExists(queue.InputPath) {
			c.logger.Debugf("source file %s does not exist, skipping delete", queue.InputPath)
			return nil
		}
		return os.Remove(queue.InputPath)
	})
}

func (c *cloudConvertManager) onFailed(queue *TaskQueue, info *cloudconvert.TaskData) error {
	// print log and queue again
	c.logger.Errorf("task %s failed with message: %s", queue.TaskID, *info.Message)
	c.logger.Infof("re-enqueueing task %s", queue.TaskID)

	// enqueue again
	newInfo, err := c.Enqueue(queue.InputPath, queue.OutputPath, queue.OutputFormat, queue.DeleteSource)
	if err != nil {
		c.logger.Errorf("failed to re-enqueue task %s: %v", queue.TaskID, err)
		return err
	}

	c.logger.Infof("re-enqueued task %s as new task %s", queue.TaskID, newInfo.TaskID)

	err = utils.WithRetry(3, c.logger, "delete bucket", func() error {
		return c.mutate(func(bucket *bbolt.Bucket) error {
			return bucket.Delete([]byte(queue.TaskID))
		})
	})

	if err != nil {
		// cancel re-enqueued task if we failed to delete old one
		c.logger.Warnf("cancelling re-enqueued task %s due to failure in deleting old task %s", newInfo.TaskID, queue.TaskID)
		if cancelErr := c.Cancel(newInfo.TaskID); cancelErr != nil {
			c.logger.Errorf("failed to cancel re-enqueued task %s: %v", newInfo.TaskID, cancelErr)
		}
		return err
	}

	return nil
}

func (c *cloudConvertManager) downloadExportedFile(ctx context.Context, url, outPath string) error {

	c.concurrent.Acquire(ctx, 1)
	defer c.concurrent.Release(1)

	if utils.IsFileExists(outPath) {
		c.logger.Warnf("file %s already exists, skipping download", outPath)
		return nil
	}

	// Open stream from CloudConvert client
	rc, err := c.client.DownloadAsFileStream(url)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	writer := pool.NewFileStreamWriter(ctx, c.downloadPool)
	return writer.WriteToFile(rc, outPath, config.ReadOnly.DownloadWriterBufferSize())
}

func (c *cloudConvertManager) mutate(fn func(bucket *bbolt.Bucket) error) error {
	return c.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(cloudConvertBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", cloudConvertBucket)
		}
		return fn(bucket)
	})
}

func (c *cloudConvertManager) read(fn func(bucket *bbolt.Bucket) error) error {
	return c.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(cloudConvertBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", cloudConvertBucket)
		}
		return fn(bucket)
	})
}
