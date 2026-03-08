package memory

import (
	"context"
	"log/slog"
	"strings"
	"sync"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// MemoryEventBus implements ports.EventBus using in-memory channels.
// Used in dev/test mode when NATS is not available.
type MemoryEventBus struct {
	mu   sync.RWMutex
	subs []memSub
}

type memSub struct {
	pattern string
	handler func(event *entities.PlatformEvent)
}

// NewMemoryEventBus creates a new in-memory event bus.
func NewMemoryEventBus() *MemoryEventBus {
	return &MemoryEventBus{}
}

// Publish dispatches the event to all matching subscribers synchronously.
func (b *MemoryEventBus) Publish(_ context.Context, event *entities.PlatformEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subject := string(event.Subject)
	for _, sub := range b.subs {
		if matchSubject(sub.pattern, subject) {
			go func(h func(*entities.PlatformEvent)) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("event handler panic", "recovered", r)
					}
				}()
				h(event)
			}(sub.handler)
		}
	}
	return nil
}

// Subscribe registers a handler for a subject pattern.
func (b *MemoryEventBus) Subscribe(subject string, handler func(event *entities.PlatformEvent)) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subs = append(b.subs, memSub{pattern: subject, handler: handler})
	slog.Info("event bus subscribed", "subject", subject)
	return nil
}

// Close is a no-op for the memory event bus.
func (b *MemoryEventBus) Close() error { return nil }

// matchSubject implements NATS-style subject matching with ">" wildcard.
func matchSubject(pattern, subject string) bool {
	if pattern == subject {
		return true
	}
	// "zenith.>" matches "zenith.deploy.started"
	if strings.HasSuffix(pattern, ".>") {
		prefix := strings.TrimSuffix(pattern, ">")
		return strings.HasPrefix(subject, prefix)
	}
	// "zenith.deploy.*" matches "zenith.deploy.started"
	if strings.Contains(pattern, "*") {
		pParts := strings.Split(pattern, ".")
		sParts := strings.Split(subject, ".")
		if len(pParts) != len(sParts) {
			return false
		}
		for i, p := range pParts {
			if p != "*" && p != sParts[i] {
				return false
			}
		}
		return true
	}
	return false
}
