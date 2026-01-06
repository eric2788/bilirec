package convert

import (
	"context"

	"github.com/eric2788/bilirec/pkg/cloudconvert"
	"go.etcd.io/bbolt"
)

type cloudConvertManager struct {
	db     *bbolt.DB
	client *cloudconvert.Client
}

func (c *cloudConvertManager) StartWorker(ctx context.Context, db *bbolt.DB) error {
	panic("not implemented") // TODO: Implement
}

func (c *cloudConvertManager) Enqueue(inputPath string, outputPath string, format string) (*TaskQueue, error) {
	panic("not implemented") // TODO: Implement
}

func (c *cloudConvertManager) Cancel(taskID string) error {
	panic("not implemented") // TODO: Implement
}

func (c *cloudConvertManager) ListInProgress() ([]*TaskQueue, error) {
	panic("not implemented") // TODO: Implement
}
