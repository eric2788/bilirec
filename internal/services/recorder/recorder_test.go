package recorder_test

import (
	"os"
	"runtime"
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

	const room = 1842862714

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

	var m1, m2, m3 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	t.Log("start it manually")
	err := recorderService.Start(room)
	if err != nil {
		t.Fatal(err)
		return
	}

	<-time.After(15 * time.Second)

	runtime.ReadMemStats(&m2)

	t.Log("stop it manually")
	t.Logf("stop success: %v", recorderService.Stop(room))

	<-time.After(5 * time.Second)
	runtime.ReadMemStats(&m3)

	t.Logf("memory before start: %.2f MB", float64(m1.Alloc/1024/1024))
	t.Logf("memory before stop: %.2f MB", float64(m2.Alloc/1024/1024))
	t.Logf("memory after stop: %.2f MB", float64(m3.Alloc/1024/1024))
	t.Logf("growth during recording: %.2f MB", float64((m2.Alloc-m1.Alloc)/1024/1024))
	t.Logf("growth after stop: %.2f MB", float64((m3.Alloc-m2.Alloc)/1024/1024))
}

func init() {
	if os.Getenv("CI") != "" {
		os.Setenv("ANONYMOUS_LOGIN", "true")
	}
}
