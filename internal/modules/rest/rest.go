// @title BiliRec API
// @version 1.0
// @description BiliRec REST API
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer <token>" in the Authorization header
package rest

import (
	"context"
	"time"

	_ "github.com/eric2788/bilirec/docs"
	"github.com/eric2788/bilirec/internal/modules/config"

	"github.com/sirupsen/logrus"
	"go.uber.org/fx"

	jwt "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
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

	if cfg.Username != "" && cfg.PasswordHash != "" {
		logger.Info("JWT authentication enabled for REST API")
		app.Post("/login",
			limiter.New(limiter.Config{Max: 10, Expiration: 1 * time.Minute}),
			loginHandler(cfg),
		)
		app.Use(jwt.New(jwt.Config{
			SigningKey: jwt.SigningKey{Key: []byte(cfg.JwtSecret)},
		}))
	}

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
