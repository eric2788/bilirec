package recorder_test

import (
	"os"
	"testing"
	"time"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/services/recorder"
	"github.com/eric2788/bilirec/internal/services/stream"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestFlvRecord(t *testing.T) {

	const room = 1964693790

	var recorderService *recorder.Service

	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Provide(stream.NewService),
		fx.Provide(recorder.NewService),
		fx.Populate(&recorderService),
	)

	app.RequireStart()
	defer app.RequireStop()

	t.Log("start it manually")
	err := recorderService.Start(room)
	if err != nil {
		t.Fatal(err)
		return
	}

	<-time.After(10 * time.Second)
	t.Log("stop it manually")
	t.Logf("stop success: %v", recorderService.Stop(room))
}

func init() {
	if os.Getenv("CI") != "" {
		os.Setenv("ANONYMOUS_LOGIN", "true")
	}
}
