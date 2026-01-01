package main_test

import (
	"os"
	"testing"
	"time"

	"github.com/eric2788/bilirec/internal/controllers/room"
	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/modules/rest"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestAppLaunch(t *testing.T) {
	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		rest.Module,

		fx.Invoke(room.NewController),
	)
	app.RequireStart()
	defer app.RequireStop()
	<-time.After(10 * time.Second)
	t.Log("REST app started successfully")
}

func init() {
	if os.Getenv("CI") != "" {
		os.Setenv("ANONYMOUS_LOGIN", "true")
	}
}
