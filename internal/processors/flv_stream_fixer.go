package processors

import (
	"context"

	"github.com/eric2788/bilirec/pkg/flv"
	"github.com/eric2788/bilirec/pkg/pipeline"
	"github.com/sirupsen/logrus"
)

type FlvStreamFixerProcessor struct {
	fixer *flv.RealtimeFixer
}

func NewFlvStreamFixer() *pipeline.ProcessorInfo[[]byte] {
	ffp := &FlvStreamFixerProcessor{
		fixer: flv.NewRealtimeFixer(),
	}
	return pipeline.NewProcessorInfo(
		"flv-fixer",
		ffp,
	)
}

func (p *FlvStreamFixerProcessor) Open(ctx context.Context, log *logrus.Entry) error {
	return nil
}

func (p *FlvStreamFixerProcessor) Process(ctx context.Context, log *logrus.Entry, data []byte) ([]byte, error) {
	return p.fixer.Fix(data)
}

func (p *FlvStreamFixerProcessor) Close() error {
	p.fixer.Close()
	return nil
}
