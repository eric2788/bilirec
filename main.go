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
	co "github.com/eric2788/bilirec/internal/services/convert"
	fi "github.com/eric2788/bilirec/internal/services/file"
	pa "github.com/eric2788/bilirec/internal/services/path"
	re "github.com/eric2788/bilirec/internal/services/recorder"
	ro "github.com/eric2788/bilirec/internal/services/room"
	st "github.com/eric2788/bilirec/internal/services/stream"
	"github.com/eric2788/bilirec/utils"
	"go.uber.org/fx"
)

func main() {

	app := fx.New(
		config.Module,
		bilibili.Module,
		rest.Module,

		fx.Provide(pa.NewService),
		fx.Provide(co.NewService),
		fx.Provide(st.NewService),
		fx.Provide(re.NewService),
		fx.Provide(ro.NewService),
		fx.Provide(fi.NewService),

		fx.Invoke(room.NewController),
		fx.Invoke(record.NewController),
		fx.Invoke(file.NewController),
		fx.Invoke(convert.NewController),

		fx.StartTimeout(utils.Ternary(os.Getenv("ANONYMOUS_LOGIN") == "true", 15*time.Second, 1*time.Minute)),
	)

	app.Run()
}
