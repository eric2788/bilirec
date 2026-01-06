package recorder_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/internal/services/convert"
	"github.com/eric2788/bilirec/internal/services/recorder"
	"github.com/eric2788/bilirec/internal/services/stream"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestRecorder_MemoryLeak_SingleSession(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping recorder memory test in short mode")
	}

	const testRoom = 1880711533 // Replace with a known live room for testing

	var recorderService *recorder.Service

	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Provide(stream.NewService),
		fx.Provide(convert.NewService),
		fx.Provide(recorder.NewService),
		fx.Populate(&recorderService),
	)

	app.RequireStart()
	defer app.RequireStop()

	var m1, m2, m3, m4 runtime.MemStats

	// Phase 1: Baseline
	runtime.GC()
	time.Sleep(200 * time.Millisecond)
	runtime.ReadMemStats(&m1)

	t.Log("📝 Starting recording session...")

	// Phase 2: Start recording
	err := recorderService.Start(testRoom)
	if err != nil {
		if err == recorder.ErrStreamNotLive {
			t.Skip("Stream not live, skipping test")
		}
		t.Fatalf("Failed to start recording:  %v", err)
	}

	// Let it record for a while
	recordDuration := 30 * time.Second
	t.Logf("⏱️  Recording for %v.. .", recordDuration)
	time.Sleep(recordDuration)

	// Phase 3: Memory during recording
	runtime.ReadMemStats(&m2)

	// Phase 4: Stop recording
	t.Log("🛑 Stopping recording...")
	if !recorderService.Stop(testRoom) {
		t.Error("Failed to stop recording")
	}

	// Wait for cleanup
	time.Sleep(3 * time.Second)

	// Phase 5: Memory after stop (before GC)
	runtime.ReadMemStats(&m3)

	// Phase 6: Force GC and measure final memory
	t.Log("🧹 Running garbage collection...")
	runtime.GC()
	runtime.GC()
	time.Sleep(1 * time.Second)
	runtime.ReadMemStats(&m4)

	// Analyze memory
	baseline := float64(m1.Alloc) / (1024 * 1024)
	duringRecord := float64(m2.Alloc) / (1024 * 1024)
	afterStop := float64(m3.Alloc) / (1024 * 1024)
	afterGC := float64(m4.Alloc) / (1024 * 1024)

	t.Logf("📊 Memory Analysis:")
	t.Logf("  Baseline:        %.2f MB", baseline)
	t.Logf("  During record:   %.2f MB (growth: +%.2f MB)", duringRecord, duringRecord-baseline)
	t.Logf("  After stop:      %.2f MB (retained: +%.2f MB)", afterStop, afterStop-baseline)
	t.Logf("  After GC:        %.2f MB (retained: +%.2f MB)", afterGC, afterGC-baseline)
	t.Logf("  Cleanup:         %.2f MB reclaimed", afterStop-afterGC)

	// Memory leak detection
	const (
		maxRetainedAfterStop = 30.0 // MB (some buffering expected)
		maxRetainedAfterGC   = 15.0 // MB
	)

	if afterStop-baseline > maxRetainedAfterStop {
		t.Errorf("⚠️  High memory after stop: %.2f MB retained (threshold: %.2f MB)",
			afterStop-baseline, maxRetainedAfterStop)
	}

	if afterGC-baseline > maxRetainedAfterGC {
		t.Errorf("⚠️  Possible memory leak: %.2f MB retained after GC (threshold: %.2f MB)",
			afterGC-baseline, maxRetainedAfterGC)
	} else {
		t.Logf("✅ Memory properly cleaned up")
	}

	// Check cleanup efficiency
	cleanupEfficiency := (afterStop - afterGC) / (duringRecord - baseline) * 100
	t.Logf("📈 Cleanup efficiency: %.1f%%", cleanupEfficiency)

	// because we have freeMemory and GC in recorder stop, so the efficiency will be zero
	// if cleanupEfficiency < 70.0 {
	// 	t.Errorf("⚠️  Low cleanup efficiency: %.1f%% (expected > 70%%)", cleanupEfficiency)
	// }
}

func TestRecorder_MemoryLeak_MultipleStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multiple session test in short mode")
	}

	const testRoom = 1842862714

	var recorderService *recorder.Service

	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Provide(stream.NewService),
		fx.Provide(convert.NewService),
		fx.Provide(recorder.NewService),
		fx.Populate(&recorderService),
	)

	app.RequireStart()
	defer app.RequireStop()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	const cycles = 5
	t.Logf("🔄 Testing %d record/stop cycles...", cycles)

	memSamples := make([]float64, cycles+1)
	memSamples[0] = float64(m1.Alloc) / (1024 * 1024)

	for cycle := 0; cycle < cycles; cycle++ {
		t.Logf("Cycle %d/%d", cycle+1, cycles)

		// Start recording
		err := recorderService.Start(testRoom)
		if err != nil {
			if err == recorder.ErrStreamNotLive {
				t.Skip("Stream not live")
			}
			t.Fatalf("Cycle %d: Failed to start:  %v", cycle, err)
		}

		// Record briefly
		time.Sleep(10 * time.Second)

		// Stop recording
		if !recorderService.Stop(testRoom) {
			t.Errorf("Cycle %d: Failed to stop", cycle)
		}

		// Wait for cleanup
		time.Sleep(2 * time.Second)

		// Force GC and measure
		runtime.GC()
		time.Sleep(500 * time.Millisecond)
		runtime.ReadMemStats(&m2)
		memSamples[cycle+1] = float64(m2.Alloc) / (1024 * 1024)

		t.Logf("  Memory after cycle %d: %.2f MB", cycle+1, memSamples[cycle+1])
	}

	// Analyze trend
	baseline := memSamples[0]
	final := memSamples[cycles]
	totalGrowth := final - baseline
	avgGrowthPerCycle := totalGrowth / float64(cycles)

	t.Logf("📊 Multi-Cycle Analysis:")
	t.Logf("  Baseline:             %.2f MB", baseline)
	t.Logf("  After %d cycles:      %.2f MB", cycles, final)
	t.Logf("  Total growth:         %.2f MB", totalGrowth)
	t.Logf("  Avg growth per cycle: %.2f MB", avgGrowthPerCycle)

	// Check for linear growth (indicates leak)
	if avgGrowthPerCycle > 5.0 {
		t.Errorf("⚠️  Memory growing linearly: %.2f MB per cycle (possible leak)", avgGrowthPerCycle)
	} else {
		t.Logf("✅ Memory growth acceptable")
	}

	// Total growth should be bounded
	if totalGrowth > 25.0 {
		t.Errorf("⚠️  Excessive memory growth: %.2f MB after %d cycles", totalGrowth, cycles)
	}
}

func TestRecorder_MemoryLeak_ProcessedDataCleanup(t *testing.T) {
	// This test specifically checks the processedData handling
	// mentioned in your question

	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	const testRoom = 1842862714

	var recorderService *recorder.Service

	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Provide(stream.NewService),
		fx.Provide(convert.NewService),
		fx.Provide(recorder.NewService),
		fx.Populate(&recorderService),
	)

	app.RequireStart()
	defer app.RequireStop()

	var m1, m2 runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Start recording
	err := recorderService.Start(testRoom)
	if err != nil {
		if err == recorder.ErrStreamNotLive {
			t.Skip("Stream not live")
		}
		t.Fatalf("Failed to start:  %v", err)
	}

	// Record for a while to process lots of data
	t.Log("⏱️  Recording to test processedData cleanup...")
	time.Sleep(20 * time.Second)

	// Stop and immediately measure
	recorderService.Stop(testRoom)
	time.Sleep(1 * time.Second)

	// Force aggressive GC
	for i := 0; i < 3; i++ {
		runtime.GC()
		time.Sleep(300 * time.Millisecond)
	}

	runtime.ReadMemStats(&m2)

	baseline := float64(m1.Alloc) / (1024 * 1024)
	afterCleanup := float64(m2.Alloc) / (1024 * 1024)
	retained := afterCleanup - baseline

	t.Logf("📊 ProcessedData Cleanup Test:")
	t.Logf("  Baseline:       %.2f MB", baseline)
	t.Logf("  After cleanup:  %.2f MB", afterCleanup)
	t.Logf("  Retained:       %.2f MB", retained)

	// With proper `_ = processedData` usage, retained should be minimal
	if retained > 12.0 {
		t.Errorf("⚠️  ProcessedData may not be cleaned up properly: %.2f MB retained", retained)
	} else {
		t.Logf("✅ ProcessedData properly cleared by GC")
	}
}

func TestRecorder_MemoryLeak_ConcurrentRecordings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// Test multiple concurrent recordings
	testRooms := []int{1842862714, 1508259} // Add more live rooms if available

	var recorderService *recorder.Service

	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Provide(stream.NewService),
		fx.Provide(convert.NewService),
		fx.Provide(recorder.NewService),
		fx.Populate(&recorderService),
	)

	app.RequireStart()
	defer app.RequireStop()

	var m1, m2, m3 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	t.Logf("🔀 Testing concurrent recordings for %d rooms", len(testRooms))

	// Start all recordings
	startedRooms := []int{}
	for _, room := range testRooms {
		err := recorderService.Start(room)
		if err != nil {
			if err == recorder.ErrStreamNotLive {
				t.Logf("Room %d not live, skipping", room)
				continue
			}
			t.Logf("Failed to start room %d: %v", room, err)
			continue
		}
		startedRooms = append(startedRooms, room)
		t.Logf("Started recording room %d", room)
	}

	if len(startedRooms) == 0 {
		t.Skip("No rooms available for testing")
	}

	// Let them run concurrently
	time.Sleep(20 * time.Second)
	runtime.ReadMemStats(&m2)

	// Stop all recordings
	for _, room := range startedRooms {
		recorderService.Stop(room)
		t.Logf("Stopped recording room %d", room)
	}

	time.Sleep(3 * time.Second)
	runtime.GC()
	runtime.GC()
	time.Sleep(1 * time.Second)
	runtime.ReadMemStats(&m3)

	baseline := float64(m1.Alloc) / (1024 * 1024)
	duringRecord := float64(m2.Alloc) / (1024 * 1024)
	afterCleanup := float64(m3.Alloc) / (1024 * 1024)

	t.Logf("📊 Concurrent Recording Analysis:")
	t.Logf("  Baseline:         %.2f MB", baseline)
	t.Logf("  During recording: %.2f MB", duringRecord)
	t.Logf("  After cleanup:     %.2f MB", afterCleanup)
	t.Logf("  Retained:         %.2f MB", afterCleanup-baseline)

	// With concurrent recordings, allow slightly more retained memory
	maxRetained := 20.0 * float64(len(startedRooms))
	if afterCleanup-baseline > maxRetained {
		t.Errorf("⚠️  Possible leak in concurrent scenario: %.2f MB retained (threshold: %.2f MB)",
			afterCleanup-baseline, maxRetained)
	} else {
		t.Logf("✅ Concurrent recordings cleaned up properly")
	}
}

func TestRecorder_Goroutine_Leak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping goroutine leak test in short mode")
	}

	const testRoom = 1842862714

	var recorderService *recorder.Service

	app := fxtest.New(t,
		config.Module,
		bilibili.Module,
		fx.Provide(stream.NewService),
		fx.Provide(convert.NewService),
		fx.Provide(recorder.NewService),
		fx.Populate(&recorderService),
	)

	app.RequireStart()
	defer app.RequireStop()

	// Baseline goroutine count
	time.Sleep(1 * time.Second)
	baseline := runtime.NumGoroutine()

	t.Logf("🧵 Baseline goroutines: %d", baseline)

	const cycles = 3
	for cycle := 0; cycle < cycles; cycle++ {
		t.Logf("Cycle %d/%d", cycle+1, cycles)

		err := recorderService.Start(testRoom)
		if err != nil {
			if err == recorder.ErrStreamNotLive {
				t.Skip("Stream not live")
			}
			t.Fatalf("Failed to start: %v", err)
		}

		time.Sleep(8 * time.Second)

		duringRecord := runtime.NumGoroutine()
		t.Logf("  During recording: %d goroutines (+%d)", duringRecord, duringRecord-baseline)

		recorderService.Stop(testRoom)
		time.Sleep(2 * time.Second)

		afterStop := runtime.NumGoroutine()
		t.Logf("  After stop: %d goroutines (+%d)", afterStop, afterStop-baseline)
	}

	// Final check
	time.Sleep(2 * time.Second)
	final := runtime.NumGoroutine()

	t.Logf("📊 Goroutine Analysis:")
	t.Logf("  Baseline:  %d", baseline)
	t.Logf("  Final:     %d", final)
	t.Logf("  Growth:   +%d", final-baseline)

	// Allow some growth for runtime goroutines, but not excessive
	if final-baseline > 10 {
		t.Errorf("⚠️  Possible goroutine leak: %d goroutines not cleaned up", final-baseline)
	} else {
		t.Logf("✅ No goroutine leak detected")
	}
}

func init() {
	if os.Getenv("CI") != "" {
		os.Setenv("ANONYMOUS_LOGIN", "true")
	}
}
