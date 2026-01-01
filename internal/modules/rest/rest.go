// @title BiliRec API
// @version 1.0
// @description Bilibili Live Recording Service API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://github.com/eric2788/bilirec

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// @host 192.168.0.127:2356
// @BasePath /
// @schemes http https
//
//go:generate swag init -g rest.go -o ../../docs
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
	app.Use(logging.New(logging.Config{
		Format: "| ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
		Output: logger.Writer(),
	}))
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
