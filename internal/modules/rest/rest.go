package rest

import (
	"context"

	_ "github.com/eric2788/bilirec/docs"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"

	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	logging "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var logger = logrus.WithField("module", "rest")

func provider(ls fx.Lifecycle, cfg *config.Config) *fiber.App {
	app := fiber.New()

	app.Use(recover.New())
	app.Use(logging.New())
	app.Use(swagger.New(swagger.Config{
		BasePath: "/",
		FilePath: "./docs/swagger.json",
		Path:     "/",
		Title:    "BiliRec API Documentation",
	}))

	ls.Append(
		fx.StartStopHook(
			func(ctx context.Context) error {
				addr := ":" + cfg.Port
				logger.Infof("starting http server on %s", addr)
				go func() {
					if err := app.Listen(addr); err != nil {
						logger.Errorf("http server error: %v", err)
					}
				}()
				return nil
			},
			func(ctx context.Context) error {
				logger.Info("stopping http server")
				return app.ShutdownWithContext(ctx)
			},
		),
	)

	return app
}

var Module = fx.Module("rest", fx.Provide(provider))
