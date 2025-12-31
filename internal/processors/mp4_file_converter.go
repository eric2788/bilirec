package processors

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/eric2788/bilirec/pkg/pipeline"
	"github.com/sirupsen/logrus"
)

type Mp4FileConverterProcessor struct {
	deleteSource bool
	oldPath      string
}

func NewMp4FileConverter(deleteSource bool) *pipeline.ProcessorInfo[string] {
	return pipeline.NewProcessorInfo(
		"mp4-file-converter",
		&Mp4FileConverterProcessor{deleteSource: deleteSource},
		pipeline.WithTimeout[string](1*time.Minute),
	)
}

func (p *Mp4FileConverterProcessor) Open(ctx context.Context, log *logrus.Entry) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-h")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (p *Mp4FileConverterProcessor) Process(ctx context.Context, log *logrus.Entry, path string) (string, error) {
	p.oldPath = path
	newFileName := path[0:strings.LastIndex(path, ".")] + ".mp4"
	convertCmd := exec.CommandContext(ctx,
		"ffmpeg",
		"-hide_banner",
		"-i",
		path,
		"-c",
		"copy",
		newFileName,
	)
	convertCmd.Stderr = log.Writer()
	convertCmd.Stdout = log.Writer()
	if err := convertCmd.Run(); err != nil {
		return path, err
	}
	return newFileName, nil
}

func (p *Mp4FileConverterProcessor) Close() error {
	if p.deleteSource && p.oldPath != "" {
		return os.Remove(p.oldPath)
	}
	return nil
}
