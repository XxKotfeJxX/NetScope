package diagnostics

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventRunStarted     EventType = "run.started"
	EventCheckStarted   EventType = "check.started"
	EventCheckCompleted EventType = "check.completed"
	EventCheckFailed    EventType = "check.failed"
	EventRunCompleted   EventType = "run.completed"
	EventRunCancelled   EventType = "run.cancelled"
)

type RunEvent struct {
	Type      EventType    `json:"-"`
	RunID     uuid.UUID    `json:"runId"`
	Check     CheckType    `json:"check,omitempty"`
	Status    string       `json:"status,omitempty"`
	Duration  int64        `json:"durationMs,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
	Result    *CheckResult `json:"result,omitempty"`
}

type EventPublisher interface {
	Publish(uuid.UUID, RunEvent)
	Subscribe(uuid.UUID) (<-chan RunEvent, func())
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID]map[uint64]chan RunEvent
	nextID      atomic.Uint64
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[uuid.UUID]map[uint64]chan RunEvent)}
}

func (h *Hub) Publish(runID uuid.UUID, event RunEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, subscriber := range h.subscribers[runID] {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (h *Hub) Subscribe(runID uuid.UUID) (<-chan RunEvent, func()) {
	h.mu.Lock()
	id := h.nextID.Add(1)
	channel := make(chan RunEvent, 16)
	if h.subscribers[runID] == nil {
		h.subscribers[runID] = make(map[uint64]chan RunEvent)
	}
	h.subscribers[runID][id] = channel
	h.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subscribers[runID], id)
			if len(h.subscribers[runID]) == 0 {
				delete(h.subscribers, runID)
			}
			close(channel)
			h.mu.Unlock()
		})
	}
	return channel, cancel
}
