package recorder

import (
	"fmt"
	"os"
	"time"

	"github.com/eric2788/bilirec/internal/processors"
	"github.com/eric2788/bilirec/pkg/pipeline"
)

func (r *Service) newPipeline(roomId int64) (*pipeline.Pipe[[]byte], error) {

	dirPath := fmt.Sprintf("%s/%d", r.cfg.OutputDir, roomId)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%s/%d.flv", dirPath, time.Now().Unix())

	return pipeline.NewPipeline(
		// fix FLV stream
		processors.NewFlvFixerProcessor(),
		// write to file with buffered writer
		// flushes every 5 seconds then writes to disk
		processors.NewWriter(filename),
	), nil
}
