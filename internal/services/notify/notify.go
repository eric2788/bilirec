package notify

import (
	"sync"
	"time"
)

type Event struct {
	Type      string `json:"type"`
	RoomID    int    `json:"room_id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type Service struct {
	mu          sync.RWMutex
	subscribers map[int]chan Event
	nextID      int
}

func NewService() *Service {
	return &Service{
		subscribers: make(map[int]chan Event),
	}
}

func (s *Service) Subscribe(buffer int) (int, <-chan Event, func()) {
	if buffer <= 0 {
		buffer = 1
	}

	s.mu.Lock()
	s.nextID++
	id := s.nextID
	ch := make(chan Event, buffer)
	s.subscribers[id] = ch
	s.mu.Unlock()

	unsubscribe := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if existing, ok := s.subscribers[id]; ok {
			delete(s.subscribers, id)
			close(existing)
		}
	}

	return id, ch, unsubscribe
}

func (s *Service) Publish(event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
			// Drop when subscriber is too slow.
		}
	}
}

func (s *Service) PublishLive(roomID int, autoRecordStarted bool) {
	message := "直播間已開播"
	eventType := "live_detected"
	if autoRecordStarted {
		message = "直播間已開播並已啟動自動錄製"
		eventType = "live_auto_record_started"
	}

	s.Publish(Event{
		Type:      eventType,
		RoomID:    roomID,
		Message:   message,
		Timestamp: time.Now().Unix(),
	})
}
