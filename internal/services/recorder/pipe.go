package recorder

import (
	"fmt"
	"os"

	"github.com/eric2788/bilirec/internal/processors"
	"github.com/eric2788/bilirec/pkg/pipeline"
)

func (r *Service) newStreamPipeline(roomId int, info *Recorder) (*pipeline.Pipe[[]byte], error) {
	dirPath := fmt.Sprintf("%s/%d", r.cfg.OutputDir, roomId)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("%s/%d.flv", dirPath, info.startTime.Unix())

	return pipeline.New(
		// fix FLV stream
		processors.NewFlvStreamFixer(),
		// write to file with buffered writer
		// flushes every 5 seconds then writes to disk
		processors.NewBufferedStreamWriter(filename),
	), nil
}

func (r *Service) newFinalPipeline() (*pipeline.Pipe[string], error) {
	pipes := pipeline.New[string]()

	if r.cfg.ConvertFLVToMp4 {
		pipes.AddProcessors(
			processors.NewMp4FileConverter(
				processors.FileConverterWithDeleteSource(r.cfg.DeleteFlvAfterConvert),
			),
		)
	}

	return pipes, nil
}
