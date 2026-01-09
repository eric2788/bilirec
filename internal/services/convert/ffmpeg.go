package convert

import (
	"context"
	"os/exec"

	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
)

const ffmpegBucket = "Queue_FFmpeg"

type ffmpegConvertManager struct {
	db     *bbolt.DB
	logger *logrus.Entry
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
	panic("not implemented") // TODO: Implement
}

func (f *ffmpegConvertManager) Cancel(taskID string) error {
	panic("not implemented") // TODO: Implement
}

func (f *ffmpegConvertManager) ListInProgress() ([]*TaskQueue, error) {
	panic("not implemented") // TODO: Implement
}

func (f *ffmpegConvertManager) runTaskPeriodically(ctx context.Context) {

}

func (f *ffmpegConvertManager) processTask(ctx context.Context, queue *TaskQueue) error {
	if queue == nil {
		return nil
	}

	logger := f.logger
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}

	logger.Infof("starting ffmpeg task id=%s input=%s output=%s", queue.TaskID, queue.InputPath, queue.OutputPath)

	cmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-hide_banner",
		"-i",
		queue.InputPath,
		"-c",
		"copy",
		queue.OutputPath,
	)
	cmd.Stdout = logger.Writer()
	cmd.Stderr = logger.Writer()

	if err := cmd.Run(); err != nil {
		logger.Errorf("ffmpeg failed for task %s: %v", queue.TaskID, err)
		return err
	}

	logger.Infof("ffmpeg finished for task %s", queue.TaskID)
	return nil
}
