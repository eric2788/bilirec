package recorder

import (
	"fmt"
	"os"
	"time"

	bili "github.com/CuteReimu/bilibili/v2"
	"github.com/eric2788/bilirec/internal/processors"
	"github.com/eric2788/bilirec/pkg/pipeline"
	"github.com/eric2788/bilirec/utils"
)

func (r *Service) newStreamPipeline(outputPath string) (*pipeline.Pipe[[]byte], error) {

	return pipeline.New(
		// fix FLV stream
		processors.NewFlvStreamFixer(),
		// write to file with buffered writer
		// flushes every 5 seconds then writes to disk
		processors.NewBufferedStreamWriter(outputPath),
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

// the time should be the time you start the record, not live start
func (r *Service) prepareFilePath(info *bili.LiveRoomInfo, start time.Time) (string, error) {
	dirPath := fmt.Sprintf("%s/%d", r.cfg.OutputDir, info.RoomId)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", err
	}
	safeTitle := utils.TruncateString(utils.SanitizeFilename(info.Title), 15)
	return fmt.Sprintf("%s/%s-%d.flv", dirPath, safeTitle, start.Unix()), nil
}
