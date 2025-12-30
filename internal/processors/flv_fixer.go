package processors

import (
	"context"

	"github.com/eric2788/bilirec/pkg/flv"
	"github.com/eric2788/bilirec/pkg/pipeline"
	"github.com/sirupsen/logrus"
)

type FlvFixerProcessor struct {
	fixer *flv.RealtimeFixer
}

func NewFlvFixerProcessor() *pipeline.ProcessorInfo[[]byte] {
	ffp := &FlvFixerProcessor{
		fixer: flv.NewRealtimeFixer(),
	}
	return pipeline.NewProcessorInfo(
		"flv-fixer",
		ffp,
	)
}

func (p *FlvFixerProcessor) Open(ctx context.Context, log *logrus.Entry) error {
	return nil
}

func (p *FlvFixerProcessor) Process(ctx context.Context, log *logrus.Entry, data []byte) ([]byte, error) {
	return p.fixer.Fix(data)
}

func (p *FlvFixerProcessor) Close() error {
	p.fixer.Close()
	return nil
}
