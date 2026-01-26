package room

import (
	"context"
	"os"
	"time"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/pkg/db"
	"github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

const defaultTTL = 5 * time.Minute

var logger = logrus.WithField("service", "room")

type Service struct {
	bilic *bilibili.Client

	ctx    context.Context
	bucket *db.Bucket
	cache  *ttlcache.Cache[string, *bilibili.LiveRoomInfoDetail]
}

func NewService(lc fx.Lifecycle, cfg *config.Config, bilic *bilibili.Client) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &Service{
		bilic: bilic,
		ctx:   ctx,
		cache: ttlcache.New(
			ttlcache.WithTTL[string, *bilibili.LiveRoomInfoDetail](defaultTTL),
			ttlcache.WithCapacity[string, *bilibili.LiveRoomInfoDetail](100),
		),
	}

	lc.Append(fx.StartStopHook(
		func() error {
			if err := os.MkdirAll(cfg.DatabaseDir, 0755); err != nil {
				return err
			}
			if client, err := db.Open(cfg.DatabaseDir + string(os.PathSeparator) + "subscribes.db"); err != nil {
				return err
			} else if bucket, err := client.Bucket(roomSubscribeBucket); err != nil {
				return err
			} else {
				svc.bucket = bucket
			}
			go svc.cache.Start()
			return nil
		},
		func() error {
			cancel()
			svc.cache.Stop()
			svc.cache.DeleteAll()
			return svc.bucket.Close()
		},
	))

	return svc
}
