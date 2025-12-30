package processors

import (
	"bufio"
	"context"
	"os"
	"time"

	"github.com/eric2788/bilirec/pkg/pipeline"
	"github.com/sirupsen/logrus"
)

type BufferedWriterProcessor struct {
	file   *os.File
	path   string
	writer *bufio.Writer
	logger *logrus.Entry
}

func NewWriter(path string) *pipeline.ProcessorInfo[[]byte] {
	return pipeline.NewProcessorInfo(
		"buffered-writer",
		&BufferedWriterProcessor{
			path: path,
		},
		pipeline.WithTimeout[[]byte](30*time.Second),
	)
}

func (w *BufferedWriterProcessor) Open(ctx context.Context, log *logrus.Entry) error {
	file, err := os.Create(w.path)
	if err != nil {
		return err
	}
	w.file = file
	w.writer = bufio.NewWriterSize(file, 4*1024*1024) // 4 MB buffer
	w.logger = log.WithField("file", file.Name())
	go w.flushPeriodically(ctx)
	return nil
}

func (w *BufferedWriterProcessor) Process(ctx context.Context, log *logrus.Entry, data []byte) ([]byte, error) {
	_, err := w.writer.Write(data)
	return data, err
}

func (w *BufferedWriterProcessor) Close() error {
	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.file.Close()
}

func (w *BufferedWriterProcessor) flushPeriodically(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := w.writer.Flush(); err != nil {
				w.logger.Warnf("error flushing writer: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
