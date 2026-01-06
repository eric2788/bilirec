package convert

import (
	"context"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/pkg/cloudconvert"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

var logger = logrus.WithField("service", "convert")

type Service struct {
	cloud          *cloudconvert.Client
	cloudthreshold int64
}

func NewService(ls fx.Lifecycle, cfg *config.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &Service{
		cloudthreshold: cfg.CloudConvertThreshold,
	}

	if cfg.CloudConvertApiKey != "" {
		svc.cloud = cloudconvert.NewClient(ctx, cfg.CloudConvertApiKey)
	} else {
		logger.Info("cloud convert api key not provided, cloud convert disabled")
	}

	ls.Append(fx.StopHook(cancel))

	return svc
}

func (s *Service) ShouldUseCloudConvert(fileSize int64) bool {
	return s.cloud != nil && s.cloudthreshold >= 0 && fileSize >= s.cloudthreshold
}
