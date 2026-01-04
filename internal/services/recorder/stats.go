package recorder

import (
	"os"
	"path/filepath"
	"strconv"
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
	return r.writtingFiles.Contains(filepath.Base(path))
}

// only works if the first relative path segment is the room ID
func (r *Service) IsRecordingUnder(relPath string) bool {
	roomId, err := strconv.Atoi(strings.SplitN(filepath.Clean(relPath), string(os.PathSeparator), 2)[0])
	if err != nil {
		return false
	}
	_, ok := r.recording.Load(roomId)
	return ok
}
