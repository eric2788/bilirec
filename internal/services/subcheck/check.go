package subcheck

import (
	"context"
	"maps"
	"slices"
	"strconv"
	"time"

	"github.com/eric2788/bilirec/internal/services/notify"
	"github.com/eric2788/bilirec/internal/services/recorder"
	"github.com/eric2788/bilirec/internal/services/room"
	"github.com/eric2788/bilirec/internal/services/subscribe"
	"github.com/eric2788/bilirec/pkg/ds"
	"github.com/eric2788/bilirec/pkg/fp"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

var logger = logrus.WithField("service", "subcheck")

const checkInterval = 1 * time.Minute

type Service struct {
	subSvc    *subscribe.Service
	roomSvc   *room.Service
	recSvc    *recorder.Service
	notifySvc *notify.Service
	notified  ds.Set[int]

	ctx    context.Context
	cancel context.CancelFunc
}

func NewService(lc fx.Lifecycle, subSvc *subscribe.Service, roomSvc *room.Service, recSvc *recorder.Service, notifySvc *notify.Service) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		subSvc:    subSvc,
		roomSvc:   roomSvc,
		recSvc:    recSvc,
		notifySvc: notifySvc,
		notified:  ds.NewSet[int](),
		ctx:       ctx,
		cancel:    cancel,
	}

	lc.Append(fx.StartStopHook(s.start, s.stop))
	return s
}

func (s *Service) start() error {
	go s.loop()
	return nil
}

func (s *Service) stop() error {
	s.cancel()
	return nil
}

func (s *Service) loop() {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	// Run once on startup so users do not always wait for the first tick.
	s.tryStartAllAutoRecordRooms()

	for {
		select {
		case <-ticker.C:
			s.tryStartAllAutoRecordRooms()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) tryStartAllAutoRecordRooms() {
	rooms, err := s.subSvc.ListSubscribedRoomsWithConfig()
	if err != nil {
		logger.Warnf("failed to list room subscriptions: %v", err)
		return
	}

	liveCheckRooms := fp.FilterByValue(rooms, func(cfg *subscribe.RoomConfig) bool {
		return cfg != nil && (cfg.Notify || cfg.AutoRecord)
	})

	notifyLiveState := s.getLiveRoomStates(slices.Collect(maps.Keys(liveCheckRooms)))

	s.invalidateNotified(rooms)

	for roomID, cfg := range liveCheckRooms {

		if cfg.AutoRecord {
			status := s.recSvc.GetStatus(roomID)
			if status == recorder.Recording || status == recorder.Recovering {
				continue
			}

			isLive, ok := notifyLiveState[roomID]
			if !ok {
				continue
			}
			if !isLive {
				s.clearNotified(roomID)
				continue
			}

			err := s.recSvc.Start(roomID)
			if err == nil {
				logger.Infof("started recording for room %d from auto-record", roomID)
				if cfg.Notify {
					s.publishLiveOnce(roomID, true)
				}
				continue
			}

			switch err {
			case recorder.ErrRecordingStarted:
				continue
			default:
				logger.Warnf("failed to start recording for room %d from auto-record: %v", roomID, err)
				continue
			}
		} else if cfg.Notify {
			isLive, ok := notifyLiveState[roomID]
			if !ok {
				continue
			}
			if isLive {
				s.publishLiveOnce(roomID, false)
			} else {
				s.clearNotified(roomID)
			}
			continue
		}
	}
}

func (s *Service) clearNotified(roomID int) {
	s.notified.Remove(roomID)
}

func (s *Service) publishLiveOnce(roomID int, autoRecordStarted bool) {
	if s.notified.Contains(roomID) {
		return
	}
	s.notifySvc.PublishLive(roomID, autoRecordStarted)
	s.notified.Add(roomID)
}

func (s *Service) invalidateNotified(rooms map[int]*subscribe.RoomConfig) {
	for _, roomID := range s.notified.ToSlice() {
		cfg, ok := rooms[roomID]
		if !ok || cfg == nil || !cfg.Notify {
			s.clearNotified(roomID)
		}
	}
}

func (s *Service) getLiveRoomStates(liveCheckRoomIDs []int) map[int]bool {
	notifyLiveState := make(map[int]bool)

	if len(liveCheckRoomIDs) > 0 {
		infos, err := s.roomSvc.GetMultipleRoomInfos(liveCheckRoomIDs...)
		if err != nil {
			logger.Warnf("batch fetch live status failed: %v, fallback to per-room check", err)
			for _, roomID := range liveCheckRoomIDs {
				isLive, checkErr := s.roomSvc.IsRoomLive(roomID)
				if checkErr != nil {
					logger.Warnf("failed to check live status for room %d: %v", roomID, checkErr)
					continue
				}
				notifyLiveState[roomID] = isLive
			}
		} else {
			for _, roomID := range liveCheckRoomIDs {
				if info, ok := infos[strconv.Itoa(roomID)]; ok && info != nil {
					notifyLiveState[roomID] = info.LiveStatus == 1
				}
			}
		}
	}

	return notifyLiveState
}
