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
	"errors"
	"time"

	_ "github.com/eric2788/bilirec/docs"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/utils"

	"github.com/sirupsen/logrus"
	"go.uber.org/fx"

	jwt "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/contrib/v3/swagger"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/basicauth"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	logging "github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/pprof"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

var logger = logrus.WithField("module", "rest")

func provider(ls fx.Lifecycle, cfg *config.Config) *fiber.App {
	app := fiber.New()

	app.Use(recover.New())

	if cfg.Debug {
		hexStr := utils.RandomHexStringMust(32)
		logger.Infof("you can use hex token (%s) or username/password to login /debug/pprof", hexStr)
		app.Use(pprof.New(), basicauth.New(basicauth.Config{
			Authorizer: func(s1, s2 string, c fiber.Ctx) bool {
				if c.Get("Authorization") == hexStr {
					return true
				}
				return compareUsernameAndPassword(cfg, s1, s2)
			},
		}))
	}

	app.Use(logging.New(logging.Config{
		Format: "| ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
		Stream: logger.Writer(),
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
			ErrorHandler: func(c fiber.Ctx, err error) error {
				if errors.Is(err, extractors.ErrNotFound) {
					return c.Status(fiber.StatusUnauthorized).SendString(jwt.ErrMissingToken.Error())
				}
				if e, ok := err.(*fiber.Error); ok {
					return c.Status(e.Code).SendString(e.Message)
				}
				return c.Status(fiber.StatusUnauthorized).SendString("Invalid or expired JWT")
			},
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
