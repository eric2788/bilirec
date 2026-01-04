package recorder

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Stats struct {
	BytesWritten   uint64       `json:"bytes_written"`
	Status         RecordStatus `json:"status"`
	StartTime      int64        `json:"start_time"`
	ElapsedSeconds int64        `json:"elapsed_seconds"`
	OutputPath     string       `json:"output_path"`
}

func (r *Service) GetStatus(roomId int) RecordStatus {
	info, ok := r.recording.Load(roomId)
	if !ok {
		return Idle
	} else if status := info.status.Load(); status == nil {
		return Idle
	} else {
		return *status
	}
}

func (r *Service) ListRecording() []int {
	rooms := make([]int, 0)
	r.recording.Range(func(key int, value *Recorder) bool {
		rooms = append(rooms, key)
		return true
	})
	return rooms
}

func (r *Service) GetStats(roomId int) (*Stats, bool) {
	info, ok := r.recording.Load(roomId)
	if !ok {
		return nil, false
	}
	status := r.GetStatus(roomId)
	return &Stats{
		BytesWritten:   info.bytesRead.Load(),
		Status:         status,
		StartTime:      info.startTime.Unix(),
		ElapsedSeconds: int64(time.Since(info.startTime).Seconds()),
		OutputPath:     info.outputPath,
	}, true
}

func (r *Service) IsRecording(path string) bool {
	return r.occupiedPaths.Contains(path)
}

func (r *Service) IsRecordingUnder(relPath string) bool {
	// normalize to absolute path based on r.cfg.OutputDir
	dirAbs, _ := filepath.Abs(filepath.Join(r.cfg.OutputDir, relPath))
	for _, p := range r.occupiedPaths.ToSlice() {
		pAbs, _ := filepath.Abs(p)
		if pAbs == dirAbs || strings.HasPrefix(pAbs, dirAbs+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
