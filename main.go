package main

import (
	"os"
	"time"

	"github.com/eric2788/bilirec/internal/controllers/convert"
	"github.com/eric2788/bilirec/internal/controllers/file"
	"github.com/eric2788/bilirec/internal/controllers/record"
	"github.com/eric2788/bilirec/internal/controllers/room"
	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/modules/rest"
	c "github.com/eric2788/bilirec/internal/services/convert"
	f "github.com/eric2788/bilirec/internal/services/file"
	"github.com/eric2788/bilirec/internal/services/recorder"
	"github.com/eric2788/bilirec/internal/services/stream"
	"github.com/eric2788/bilirec/utils"
	"go.uber.org/fx"
)

func main() {

	app := fx.New(
		config.Module,
		bilibili.Module,
		rest.Module,

		fx.Provide(c.NewService),
		fx.Provide(stream.NewService),
		fx.Provide(recorder.NewService),
		fx.Provide(f.NewService),

		fx.Invoke(room.NewController),
		fx.Invoke(record.NewController),
		fx.Invoke(file.NewController),
		fx.Invoke(convert.NewController),

		fx.StartTimeout(utils.Ternary(os.Getenv("ANONYMOUS_LOGIN") == "true", 15*time.Second, 1*time.Minute)),
	)

	app.Run()
}
