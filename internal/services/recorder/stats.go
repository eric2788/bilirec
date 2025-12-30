package recorder

import "time"

type Stats struct {
	BytesWritten   uint64       `json:"bytes_written"`
	Status         RecordStatus `json:"status"`
	StartTime      int64        `json:"start_time"`
	RecoveredCount int64        `json:"recovered_count"`
	ElapsedSeconds int64        `json:"elapsed_seconds"`
}

func (r *Service) GetStatus(roomId int64) RecordStatus {
	info, ok := r.recording.Load(roomId)
	if !ok {
		return Idle
	} else if status := info.status.Load(); status == nil {
		return Idle
	} else {
		return *status
	}
}

func (r *Service) ListRecording() []int64 {
	rooms := make([]int64, 0)
	r.recording.Range(func(key int64, value *Recorder) bool {
		rooms = append(rooms, key)
		return true
	})
	return rooms
}

func (r *Service) GetStats(roomId int64) (*Stats, bool) {
	info, ok := r.recording.Load(roomId)
	if !ok {
		return nil, false
	}
	status := r.GetStatus(roomId)
	return &Stats{
		BytesWritten:   info.bytesRead.Load(),
		Status:         status,
		StartTime:      info.startTime.Unix(),
		RecoveredCount: info.recoveredCount.Value(),
		ElapsedSeconds: int64(time.Since(info.startTime).Seconds()),
	}, true
}
