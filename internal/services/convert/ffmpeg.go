package convert

import (
	"context"

	"go.etcd.io/bbolt"
)

type ffmpegConvertManager struct {
	db *bbolt.DB
}

func (f *ffmpegConvertManager) StartWorker(ctx context.Context, db *bbolt.DB) error {
	panic("not implemented") // TODO: Implement
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
