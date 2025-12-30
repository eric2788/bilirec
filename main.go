package main

import (
	"github.com/eric2788/bilirec/internal/controllers/file"
	"github.com/eric2788/bilirec/internal/controllers/record"
	"github.com/eric2788/bilirec/internal/controllers/room"
	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/modules/rest"
	f "github.com/eric2788/bilirec/internal/services/file"
	"github.com/eric2788/bilirec/internal/services/recorder"
	"github.com/eric2788/bilirec/internal/services/stream"
	"go.uber.org/fx"
)

// @title BiliRec API
// @version 1.0
// @description Bilibili Live Recording Service API
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://github.com/eric2788/bilirec

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

func main() {

	app := fx.New(
		config.Module,
		bilibili.Module,
		rest.Module,

		fx.Provide(recorder.NewService),
		fx.Provide(stream.NewService),
		fx.Provide(f.NewService),

		fx.Invoke(room.NewController),
		fx.Invoke(record.NewController),
		fx.Invoke(file.NewController),
	)

	app.Run()
}
