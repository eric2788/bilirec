package bilibili_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/go-resty/resty/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestGetStreamUrls(t *testing.T) {
	var client *bilibili.Client
	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Populate(&client),
		fx.StartTimeout(25*time.Second),
	)

	app.RequireStart()
	defer app.RequireStop()

	urls, err := client.GetStreamURLs(8222458)
	if err != nil {
		t.Fatal(err)
	}
	for _, url := range urls {
		t.Logf("Stream URL: %s", url)
	}
}

func TestGetStreamUrlsV2(t *testing.T) {
	var client *bilibili.Client
	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Populate(&client),
		fx.StartTimeout(25*time.Second),
	)
	app.RequireStart()
	defer app.RequireStop()

	urls, err := client.GetStreamURLsV2(8222458)
	if err != nil {
		t.Fatal(err)
	}
	for _, url := range urls {
		t.Logf("Stream URL: %s", url)
	}
}

func TestHeaders(t *testing.T) {
	var client *bilibili.Client
	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Populate(&client),
		fx.StartTimeout(25*time.Second),
	)
	app.RequireStart()
	defer app.RequireStop()

	a := &strings.Builder{}
	if _, err := client.Do(func(req *resty.Request) (*resty.Response, error) {
		err := req.Header.Write(a)
		return nil, err
	}); err != nil {
		t.Fatal(err)
	}
	t.Log("client header: ", a)

	b := &strings.Builder{}
	if _, err := client.DoLive(func(req *resty.Request) (*resty.Response, error) {
		err := req.Header.Write(b)
		return nil, err
	}); err != nil {
		t.Fatal(err)
	}
	t.Log("live client header: ", b)

}

func init() {
	if os.Getenv("CI") != "" {
		os.Setenv("ANONYMOUS_LOGIN", "true")
	}
}
