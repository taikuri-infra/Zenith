package deploy

import (
	"sync"
	"time"
)

// EventType describes what happened during a deployment lifecycle.
type EventType string

const (
	EventDeploymentStarted EventType = "deployment_started"
	EventBuildProgress     EventType = "build_progress"
	EventBuildComplete     EventType = "build_complete"
	EventDeployStarted     EventType = "deploy_started"
	EventDeployComplete    EventType = "deploy_complete"
	EventDeployFailed      EventType = "deploy_failed"
)

// DeployEvent represents a single deployment lifecycle event.
type DeployEvent struct {
	Type         EventType `json:"type"`
	AppID        string    `json:"app_id"`
	AppName      string    `json:"app_name"`
	DeploymentID string    `json:"deployment_id"`
	Status       string    `json:"status"`
	Image        string    `json:"image,omitempty"`
	Message      string    `json:"message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// EventSubscriber receives deployment events.
type EventSubscriber struct {
	Ch   chan DeployEvent
	done chan struct{}
}

// Close stops the subscriber from receiving new events.
func (s *EventSubscriber) Close() {
	select {
	case <-s.done:
		// already closed
	default:
		close(s.done)
	}
}

// EventHub is an in-memory deployment event broadcaster.
// Producers (pipeline) publish events. Consumers (SSE handlers) subscribe.
// Unlike LogHub which is per-deployment, EventHub broadcasts globally.
type EventHub struct {
	mu          sync.RWMutex
	history     []DeployEvent      // recent events (ring buffer)
	subscribers []*EventSubscriber // active subscribers
	maxHistory  int
}

// NewEventHub creates a new EventHub with a history limit.
func NewEventHub(maxHistory int) *EventHub {
	if maxHistory <= 0 {
		maxHistory = 50
	}
	return &EventHub{
		history:    make([]DeployEvent, 0, maxHistory),
		maxHistory: maxHistory,
	}
}

// Publish sends an event to all subscribers and stores it in history.
func (h *EventHub) Publish(event DeployEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Append to history (ring buffer style)
	if len(h.history) >= h.maxHistory {
		h.history = h.history[1:]
	}
	h.history = append(h.history, event)

	// Fan out to all subscribers (non-blocking)
	alive := h.subscribers[:0]
	for _, sub := range h.subscribers {
		select {
		case <-sub.done:
			close(sub.Ch)
			continue
		default:
			select {
			case sub.Ch <- event:
			default:
				// subscriber too slow, skip this event
			}
			alive = append(alive, sub)
		}
	}
	h.subscribers = alive
}

// Subscribe returns a subscriber that receives real-time events.
// It replays recent history on connect.
func (h *EventHub) Subscribe(bufferSize int) *EventSubscriber {
	if bufferSize <= 0 {
		bufferSize = 50
	}

	sub := &EventSubscriber{
		Ch:   make(chan DeployEvent, bufferSize),
		done: make(chan struct{}),
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Replay recent history
	for _, event := range h.history {
		select {
		case sub.Ch <- event:
		default:
			// buffer full during replay, skip oldest
		}
	}

	h.subscribers = append(h.subscribers, sub)
	return sub
}

// History returns all stored events.
func (h *EventHub) History() []DeployEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]DeployEvent, len(h.history))
	copy(result, h.history)
	return result
}
