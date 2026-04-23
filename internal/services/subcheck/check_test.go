package subcheck

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/services/room"
	"github.com/eric2788/bilirec/internal/services/subscribe"
	"go.uber.org/fx"
)

type testLifecycle struct {
	hooks []fx.Hook
}

func (l *testLifecycle) Append(h fx.Hook) {
	l.hooks = append(l.hooks, h)
}

func (l *testLifecycle) Start(tb testing.TB) {
	tb.Helper()
	for i, hook := range l.hooks {
		if hook.OnStart != nil {
			if err := hook.OnStart(context.Background()); err != nil {
				tb.Fatalf("start hook %d failed: %v", i, err)
			}
		}
	}
}

func (l *testLifecycle) Stop(tb testing.TB) {
	tb.Helper()
	for i := len(l.hooks) - 1; i >= 0; i-- {
		hook := l.hooks[i]
		if hook.OnStop != nil {
			if err := hook.OnStop(context.Background()); err != nil {
				tb.Fatalf("stop hook %d failed: %v", i, err)
			}
		}
	}
}

func newSubcheckServiceForTest(tb testing.TB) (*Service, *testLifecycle) {
	tb.Helper()

	subLifecycle := &testLifecycle{}
	subCfg := &config.Config{DatabaseDir: tb.TempDir()}
	subSvc := subscribe.NewService(subLifecycle, subCfg, &room.Service{})
	subLifecycle.Start(tb)

	mainLifecycle := &testLifecycle{}
	svc := NewService(mainLifecycle, subSvc, nil, nil, nil)

	tb.Cleanup(func() {
		svc.stop()
		subLifecycle.Stop(tb)
	})

	return svc, mainLifecycle
}

func TestService_TryStartAllAutoRecordRooms_EmptySubscriptions(t *testing.T) {
	svc, _ := newSubcheckServiceForTest(t)

	for i := 0; i < 3; i++ {
		svc.tryStartAllAutoRecordRooms()
	}

	if got := svc.notified.Size(); got != 0 {
		t.Fatalf("expected notified set to stay empty, got %d", got)
	}
}

func TestService_StartStop_CancelsContext(t *testing.T) {
	svc, _ := newSubcheckServiceForTest(t)

	if err := svc.stop(); err != nil {
		t.Fatalf("stop returned error: %v", err)
	}

	select {
	case <-svc.ctx.Done():
		// expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected context to be canceled after stop")
	}
}

func TestService_MemoryRisk_RepeatedCheckCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory risk test in short mode")
	}

	svc, _ := newSubcheckServiceForTest(t)

	var before, after runtime.MemStats

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&before)

	const iterations = 5000
	for i := 0; i < iterations; i++ {
		svc.tryStartAllAutoRecordRooms()
	}

	if got := svc.notified.Size(); got != 0 {
		t.Fatalf("expected notified set to stay empty after %d runs, got %d", iterations, got)
	}

	runtime.GC()
	runtime.GC()
	time.Sleep(300 * time.Millisecond)
	runtime.ReadMemStats(&after)

	retainedBytes := int64(after.Alloc) - int64(before.Alloc)
	retainedMB := float64(retainedBytes) / (1024 * 1024)
	t.Logf("retained memory after %d cycles: %.2f MB", iterations, retainedMB)

	const maxRetainedMB = 8.0
	if retainedMB > maxRetainedMB {
		t.Fatalf("possible memory leak risk: retained %.2f MB > %.2f MB", retainedMB, maxRetainedMB)
	}
}

func BenchmarkService_TryStartAllAutoRecordRooms_EmptySubscriptions(b *testing.B) {
	svc, _ := newSubcheckServiceForTest(b)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		svc.tryStartAllAutoRecordRooms()
	}
}
