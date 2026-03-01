package deploy

import (
	"sync"
	"time"
)

// LogEntry represents a single log line from a build or deployment.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // "info", "warn", "error", "build", "deploy"
	Message   string    `json:"message"`
}

// LogSubscriber receives log entries from a deployment.
type LogSubscriber struct {
	Ch   chan LogEntry
	done chan struct{}
}

// Close stops the subscriber from receiving new entries.
func (s *LogSubscriber) Close() {
	select {
	case <-s.done:
		// already closed
	default:
		close(s.done)
	}
}

// LogHub is an in-memory log broadcaster.
// Producers (pipeline, builder, deployer) publish log entries per deployment ID.
// Consumers (WebSocket handlers) subscribe to a deployment's logs in real time.
type LogHub struct {
	mu          sync.RWMutex
	history     map[string][]LogEntry    // deploymentID -> past entries
	subscribers map[string][]*LogSubscriber // deploymentID -> active subscribers
	maxHistory  int
}

// NewLogHub creates a new LogHub with a per-deployment history limit.
func NewLogHub(maxHistory int) *LogHub {
	if maxHistory <= 0 {
		maxHistory = 500
	}
	return &LogHub{
		history:     make(map[string][]LogEntry),
		subscribers: make(map[string][]*LogSubscriber),
		maxHistory:  maxHistory,
	}
}

// Publish sends a log entry to all subscribers and stores it in history.
func (h *LogHub) Publish(deploymentID string, entry LogEntry) {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Append to history (ring buffer style)
	hist := h.history[deploymentID]
	if len(hist) >= h.maxHistory {
		hist = hist[1:]
	}
	h.history[deploymentID] = append(hist, entry)

	// Fan out to all subscribers (non-blocking)
	subs := h.subscribers[deploymentID]
	alive := subs[:0]
	for _, sub := range subs {
		select {
		case <-sub.done:
			// Subscriber closed — remove from list (Ch closed by Cleanup)
			continue
		default:
			select {
			case sub.Ch <- entry:
			default:
				// subscriber too slow, skip this entry
			}
			alive = append(alive, sub)
		}
	}
	h.subscribers[deploymentID] = alive
}

// PublishInfo is a convenience method for info-level log entries.
func (h *LogHub) PublishInfo(deploymentID, message string) {
	h.Publish(deploymentID, LogEntry{Level: "info", Message: message})
}

// PublishError is a convenience method for error-level log entries.
func (h *LogHub) PublishError(deploymentID, message string) {
	h.Publish(deploymentID, LogEntry{Level: "error", Message: message})
}

// PublishBuild is a convenience method for build-level log entries.
func (h *LogHub) PublishBuild(deploymentID, message string) {
	h.Publish(deploymentID, LogEntry{Level: "build", Message: message})
}

// PublishDeploy is a convenience method for deploy-level log entries.
func (h *LogHub) PublishDeploy(deploymentID, message string) {
	h.Publish(deploymentID, LogEntry{Level: "deploy", Message: message})
}

// Subscribe returns a subscriber that receives real-time log entries
// for the given deployment. It also replays existing history.
func (h *LogHub) Subscribe(deploymentID string, bufferSize int) *LogSubscriber {
	if bufferSize <= 0 {
		bufferSize = 100
	}

	sub := &LogSubscriber{
		Ch:   make(chan LogEntry, bufferSize),
		done: make(chan struct{}),
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Replay history
	for _, entry := range h.history[deploymentID] {
		select {
		case sub.Ch <- entry:
		default:
			// buffer full during replay, skip oldest
		}
	}

	h.subscribers[deploymentID] = append(h.subscribers[deploymentID], sub)
	return sub
}

// History returns all stored log entries for a deployment.
func (h *LogHub) History(deploymentID string) []LogEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	hist := h.history[deploymentID]
	result := make([]LogEntry, len(hist))
	copy(result, hist)
	return result
}

// Cleanup removes history and subscribers for a deployment.
func (h *LogHub) Cleanup(deploymentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, sub := range h.subscribers[deploymentID] {
		sub.Close()
		close(sub.Ch)
	}

	delete(h.history, deploymentID)
	delete(h.subscribers, deploymentID)
}
