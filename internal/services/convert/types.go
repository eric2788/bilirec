package convert

import (
	"context"

	"go.etcd.io/bbolt"
)

type ConvertManager interface {
	StartWorker(ctx context.Context, db *bbolt.DB) error
	Enqueue(inputPath, outputPath, format string) (*TaskQueue, error)
	Cancel(taskID string) error
	ListInProgress() ([]*TaskQueue, error)
}

type TaskQueue struct {
	TaskID     string `json:"task_id"`
	InputPath  string `json:"input_path"`
	OutputPath string `json:"output_path"`
	Format     string `json:"format"`
}
