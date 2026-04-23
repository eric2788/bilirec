package subscribe_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/services/room"
	"github.com/eric2788/bilirec/internal/services/subscribe"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

var testRoomIDs = []int{
	1740570024, 1746703849, 1750976883, 1817612638, 1901320728,
	1908320745, 1929633038, 22150848, 1880711533, 2712573,
	1792593743, 1758144136, 1861306691, 30646111, 25191163,
	1817656604, 1935933756, 1814196999, 1731715643, 1935932996,
	1780363604, 1791981879, 1793683231, 1938419607, 1964697471,
	32324812, 1975553420, 1741665248, 1814408095, 1892451653,
	30825092, 1869215690, 1819589148, 1746177378, 23293581,
	1861780230, 1869214619, 1931554655, 1829747272, 1935930495,
	31020316, 1935938617, 27780132,
}

func getTestRoomID(index int) int {
	return testRoomIDs[index%len(testRoomIDs)]
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	if os.Getenv("CI") != "" {
		os.Setenv("ANONYMOUS_LOGIN", "true")
	}
}

func newSubscribeService(t *testing.T) *subscribe.Service {
	t.Helper()

	var svc *subscribe.Service
	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Provide(room.NewService),
		fx.Provide(subscribe.NewService),
		fx.Populate(&svc),
		fx.StartTimeout(5*time.Second),
	)
	app.RequireStart()
	t.Cleanup(func() {
		app.RequireStop()
	})
	return svc
}

func TestPubsub_Subscribe(t *testing.T) {
	svc := newSubscribeService(t)

	existing, _ := svc.ListSubscribedRooms()
	for _, rid := range existing {
		_ = svc.Unsubscribe(rid)
	}

	testRoomID := getTestRoomID(0)
	err := svc.Subscribe(testRoomID)
	if err != nil {
		t.Fatalf("first subscribe failed: %v", err)
	}

	err = svc.Subscribe(testRoomID)
	if err != subscribe.ErrRoomAlreadySubscribed {
		t.Fatalf("expected ErrRoomAlreadySubscribed, got: %v", err)
	}

	if err := svc.Unsubscribe(testRoomID); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}

	err = svc.Subscribe(testRoomID)
	if err != nil {
		t.Fatalf("resubscribe after unsubscribe failed: %v", err)
	}
}

func TestPubsub_Unsubscribe(t *testing.T) {
	svc := newSubscribeService(t)
	testRoomID := getTestRoomID(1)

	err := svc.Unsubscribe(testRoomID)
	if err != subscribe.ErrRoomNotSubscribed {
		t.Fatalf("expected ErrRoomNotSubscribed, got: %v", err)
	}

	if err := svc.Subscribe(testRoomID); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	err = svc.Unsubscribe(testRoomID)
	if err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}

	err = svc.Unsubscribe(testRoomID)
	if err != subscribe.ErrRoomNotSubscribed {
		t.Fatalf("expected ErrRoomNotSubscribed on second unsubscribe, got: %v", err)
	}
}

func TestPubsub_IsSubscribed(t *testing.T) {
	svc := newSubscribeService(t)
	testRoomID := getTestRoomID(2)

	isSubscribed, err := svc.IsSubscribed(testRoomID)
	if err != nil {
		t.Fatalf("IsSubscribed failed: %v", err)
	}
	if isSubscribed {
		t.Fatal("expected not subscribed before subscription")
	}

	if err := svc.Subscribe(testRoomID); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	isSubscribed, err = svc.IsSubscribed(testRoomID)
	if err != nil {
		t.Fatalf("IsSubscribed failed: %v", err)
	}
	if !isSubscribed {
		t.Fatal("expected subscribed after subscription")
	}

	if err := svc.Unsubscribe(testRoomID); err != nil {
		t.Fatalf("unsubscribe failed: %v", err)
	}

	isSubscribed, err = svc.IsSubscribed(testRoomID)
	if err != nil {
		t.Fatalf("IsSubscribed failed: %v", err)
	}
	if isSubscribed {
		t.Fatal("expected not subscribed after unsubscription")
	}
}

func TestPubsub_ListSubscribedRooms(t *testing.T) {
	svc := newSubscribeService(t)

	existingRooms, err := svc.ListSubscribedRooms()
	if err != nil {
		t.Fatalf("ListSubscribedRooms failed: %v", err)
	}
	for _, rid := range existingRooms {
		_ = svc.Unsubscribe(rid)
	}

	rooms, err := svc.ListSubscribedRooms()
	if err != nil {
		t.Fatalf("ListSubscribedRooms failed: %v", err)
	}
	if len(rooms) != 0 {
		t.Fatalf("expected empty list after cleanup, got %d rooms", len(rooms))
	}

	testRooms := []int{getTestRoomID(3), getTestRoomID(4), getTestRoomID(5)}
	for _, roomID := range testRooms {
		if err := svc.Subscribe(roomID); err != nil {
			t.Fatalf("subscribe %d failed: %v", roomID, err)
		}
	}

	rooms, err = svc.ListSubscribedRooms()
	if err != nil {
		t.Fatalf("ListSubscribedRooms failed: %v", err)
	}
	if len(rooms) != len(testRooms) {
		t.Fatalf("expected %d rooms, got %d", len(testRooms), len(rooms))
	}

	roomMap := make(map[int]bool)
	for _, rid := range rooms {
		roomMap[rid] = true
	}
	for _, expectedRoom := range testRooms {
		if !roomMap[expectedRoom] {
			t.Fatalf("room %d not in list", expectedRoom)
		}
	}

	for _, rid := range testRooms {
		if err := svc.Unsubscribe(rid); err != nil {
			t.Fatalf("unsubscribe failed: %v", err)
		}
	}
}

func TestMemoryLeak_SubscribeUnsubscribeCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory test in short mode")
	}

	svc := newSubscribeService(t)
	var m1, m2, m3 runtime.MemStats

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&m1)

	const iterations = 200
	for i := 0; i < iterations; i++ {
		roomID := getTestRoomID(i)
		if err := svc.Subscribe(roomID); err != nil {
			t.Fatalf("subscribe failed: %v", err)
		}
		if err := svc.Unsubscribe(roomID); err != nil {
			t.Fatalf("unsubscribe failed: %v", err)
		}
	}

	runtime.ReadMemStats(&m2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	runtime.ReadMemStats(&m3)

	baselineAlloc := float64(m1.Alloc) / (1024 * 1024)
	afterGCAlloc := float64(m3.Alloc) / (1024 * 1024)
	retainedAfterGC := afterGCAlloc - baselineAlloc

	const maxRetainedMB = 5.0
	if retainedAfterGC > maxRetainedMB {
		t.Errorf("possible memory leak: %.2f MB retained after GC", retainedAfterGC)
	}
}

func TestMemoryLeak_ListCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory test in short mode")
	}

	svc := newSubscribeService(t)
	var m1, m2, m3 runtime.MemStats

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&m1)

	const iterations = 200
	for i := 0; i < iterations; i++ {
		if _, err := svc.ListSubscribedRooms(); err != nil {
			t.Fatalf("ListSubscribedRooms failed: %v", err)
		}
	}

	runtime.ReadMemStats(&m2)
	runtime.GC()
	time.Sleep(500 * time.Millisecond)
	runtime.ReadMemStats(&m3)

	baselineAlloc := float64(m1.Alloc) / (1024 * 1024)
	afterGCAlloc := float64(m3.Alloc) / (1024 * 1024)
	retainedAfterGC := afterGCAlloc - baselineAlloc

	const maxRetainedMB = 5.0
	if retainedAfterGC > maxRetainedMB {
		t.Errorf("possible memory leak: %.2f MB retained after GC", retainedAfterGC)
	}
}

func TestConcurrency_SubscribeAndListIsolated(t *testing.T) {
	svc := newSubscribeService(t)
	done := make(chan bool, 20)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			for j := 0; j < 10; j++ {
				roomIndex := (id * 4) + (j % 4)
				roomID := getTestRoomID(roomIndex)
				_ = svc.Subscribe(roomID)
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 50; j++ {
				if _, err := svc.ListSubscribedRooms(); err != nil {
					t.Errorf("ListSubscribedRooms failed: %v", err)
					return
				}
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}
