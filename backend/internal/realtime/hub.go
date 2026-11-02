package realtime

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Resource  string      `json:"resource"`
	ResourceID string     `json:"resourceId"`
	Data      interface{} `json:"data"`
	OccurredAt time.Time  `json:"occurredAt"`
}

type Hub struct {
	mu          sync.Mutex
	subscribers map[string]chan Event
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string]chan Event)}
}

func (hub *Hub) Subscribe(buffer int) (string, <-chan Event, func()) {
	if buffer < 1 {
		buffer = 1
	}
	id := uuid.NewString()
	channel := make(chan Event, buffer)
	hub.mu.Lock()
	hub.subscribers[id] = channel
	hub.mu.Unlock()
	unsubscribe := func() {
		hub.mu.Lock()
		if existing, ok := hub.subscribers[id]; ok {
			delete(hub.subscribers, id)
			close(existing)
		}
		hub.mu.Unlock()
	}
	return id, channel, unsubscribe
}

func (hub *Hub) Publish(event Event) error {
	if event.Type == "" || event.Resource == "" || event.ResourceID == "" {
		return errors.New("realtime event type and resource identity are required")
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	hub.mu.Lock()
	defer hub.mu.Unlock()
	for _, subscriber := range hub.subscribers {
		select {
		case subscriber <- event:
		default:
			select {
			case <-subscriber:
			default:
			}
			select {
			case subscriber <- event:
			default:
			}
		}
	}
	return nil
}