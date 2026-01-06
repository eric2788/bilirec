package convert

import (
	"context"

	"go.etcd.io/bbolt"
)

type GetActiveRecordings func() int

type ConvertManager interface {
	StartWorker(ctx context.Context, db *bbolt.DB) error
	Enqueue(inputPath, outputPath, format string, deleteSource bool) (*TaskQueue, error)
	Cancel(taskID string) error
	ListInProgress() ([]*TaskQueue, error)
}

type TaskQueue struct {
	TaskID       string `json:"task_id"`
	InputPath    string `json:"input_path"`
	OutputPath   string `json:"output_path"`
	InputFormat  string `json:"input_format"`
	OutputFormat string `json:"output_format"`
	DeleteSource bool   `json:"delete_source"`
}
