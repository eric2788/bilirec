package notify

import (
	"runtime"
	"testing"
	"time"
)

func TestService_SubscribePublishUnsubscribe(t *testing.T) {
	svc := NewService()

	id, ch, unsubscribe := svc.Subscribe(4)
	if id <= 0 {
		t.Fatalf("expected positive subscriber id, got %d", id)
	}

	evt := Event{
		Type:      "live_detected",
		RoomID:    123,
		Message:   "hello",
		Timestamp: time.Now().Unix(),
	}

	svc.Publish(evt)

	select {
	case got := <-ch:
		if got.Type != evt.Type || got.RoomID != evt.RoomID || got.Message != evt.Message {
			t.Fatalf("unexpected event payload: %+v", got)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("did not receive published event in time")
	}

	unsubscribe()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be closed after unsubscribe")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("channel was not closed after unsubscribe")
	}

	// Publishing after unsubscribe should be safe and should not block.
	svc.Publish(evt)
}

func TestService_Subscribe_DefaultBuffer(t *testing.T) {
	svc := NewService()

	_, ch, unsubscribe := svc.Subscribe(0)
	defer unsubscribe()

	if cap(ch) != 1 {
		t.Fatalf("expected default buffer size 1, got %d", cap(ch))
	}
}

func TestService_Publish_MultipleSubscribers(t *testing.T) {
	svc := NewService()

	_, ch1, unsub1 := svc.Subscribe(2)
	defer unsub1()

	_, ch2, unsub2 := svc.Subscribe(2)
	defer unsub2()

	evt := Event{Type: "live_detected", RoomID: 456, Message: "multi", Timestamp: time.Now().Unix()}
	svc.Publish(evt)

	recv := func(name string, ch <-chan Event) {
		t.Helper()
		select {
		case got := <-ch:
			if got.RoomID != evt.RoomID || got.Type != evt.Type {
				t.Fatalf("%s got unexpected event: %+v", name, got)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("%s did not receive event", name)
		}
	}

	recv("subscriber-1", ch1)
	recv("subscriber-2", ch2)
}

func TestService_MemoryRisk_SubscribeUnsubscribeCycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory risk test in short mode")
	}

	svc := NewService()

	var before, after runtime.MemStats

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	runtime.ReadMemStats(&before)

	const iterations = 50000
	const bufferSize = 32

	for i := 0; i < iterations; i++ {
		_, _, unsubscribe := svc.Subscribe(bufferSize)
		unsubscribe()
	}

	if got := len(svc.subscribers); got != 0 {
		t.Fatalf("expected no active subscribers after test loop, got %d", got)
	}

	runtime.GC()
	runtime.GC()
	time.Sleep(300 * time.Millisecond)
	runtime.ReadMemStats(&after)

	retainedMB := float64(int64(after.Alloc)-int64(before.Alloc)) / (1024 * 1024)
	t.Logf("retained memory after %d cycles: %.2f MB", iterations, retainedMB)

	const maxRetainedMB = 8.0
	if retainedMB > maxRetainedMB {
		t.Fatalf("possible memory leak risk: retained %.2f MB > %.2f MB", retainedMB, maxRetainedMB)
	}
}

func BenchmarkService_SubscribeUnsubscribe(b *testing.B) {
	svc := NewService()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, unsubscribe := svc.Subscribe(32)
		unsubscribe()
	}
}

func BenchmarkService_Publish_WithSubscribers(b *testing.B) {
	svc := NewService()

	_, ch1, unsub1 := svc.Subscribe(64)
	defer unsub1()
	_, ch2, unsub2 := svc.Subscribe(64)
	defer unsub2()

	// Drain channels so benchmarks measure publish cost instead of queue saturation behavior.
	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ch1:
			}
		}
	}()
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ch2:
			}
		}
	}()
	defer close(stop)

	evt := Event{Type: "live_detected", RoomID: 42, Message: "bench", Timestamp: time.Now().Unix()}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		svc.Publish(evt)
	}
}
