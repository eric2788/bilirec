package recorder

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/pkg/ds"
)

func TestShouldRotateSegment(t *testing.T) {
	s := &Service{
		cfg: &config.Config{MaxRecordingFileSizeBytes: 100},
	}
	info := &Recorder{}

	info.segmentBytes.Store(99)
	if s.shouldRotateSegment(info) {
		t.Fatal("expected no rotation when segment bytes below max")
	}

	info.segmentBytes.Store(100)
	if !s.shouldRotateSegment(info) {
		t.Fatal("expected rotation when segment bytes reaches max")
	}

	s.cfg.MaxRecordingFileSizeBytes = 0
	if s.shouldRotateSegment(info) {
		t.Fatal("expected no rotation when max size is disabled")
	}
}

func TestNextSegmentOutputPath(t *testing.T) {
	tempDir := t.TempDir()
	s := &Service{
		writtingFiles: ds.NewSyncedSet[string](),
	}

	currentPath := filepath.Join(tempDir, "title-with-dash-20260101_090000.flv")
	fixed := time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC)

	next := s.nextSegmentOutputPath(currentPath, fixed)
	expected := filepath.Join(tempDir, "title-with-dash-20260101_093000.flv")
	if next != expected {
		t.Fatalf("unexpected next segment path, got %s, want %s", next, expected)
	}

	if err := os.WriteFile(expected, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	nextWithSuffix := s.nextSegmentOutputPath(currentPath, fixed)
	expectedWithSuffix := filepath.Join(tempDir, "title-with-dash-20260101_093000-1.flv")
	if nextWithSuffix != expectedWithSuffix {
		t.Fatalf("unexpected collision-resolved path, got %s, want %s", nextWithSuffix, expectedWithSuffix)
	}
}
