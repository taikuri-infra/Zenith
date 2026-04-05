package deploy

import (
	"sync"
	"testing"
	"time"
)

func TestNewEventHub_DefaultMaxHistory(t *testing.T) {
	hub := NewEventHub(0)
	if hub.maxHistory != 50 {
		t.Errorf("Expected default maxHistory 50, got %d", hub.maxHistory)
	}
}

func TestNewEventHub_NegativeMaxHistory(t *testing.T) {
	hub := NewEventHub(-10)
	if hub.maxHistory != 50 {
		t.Errorf("Expected default maxHistory 50, got %d", hub.maxHistory)
	}
}

func TestNewEventHub_CustomMaxHistory(t *testing.T) {
	hub := NewEventHub(100)
	if hub.maxHistory != 100 {
		t.Errorf("Expected maxHistory 100, got %d", hub.maxHistory)
	}
}

func TestEventHub_PublishAndHistory(t *testing.T) {
	hub := NewEventHub(50)

	hub.Publish(DeployEvent{
		Type:    EventDeploymentStarted,
		AppID:   "app-1",
		AppName: "web",
		Message: "deploy started",
	})

	hub.Publish(DeployEvent{
		Type:    EventBuildComplete,
		AppID:   "app-1",
		AppName: "web",
		Message: "build done",
	})

	history := hub.History()
	if len(history) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(history))
	}
	if history[0].Type != EventDeploymentStarted {
		t.Errorf("Expected first event type %s, got %s", EventDeploymentStarted, history[0].Type)
	}
	if history[1].Type != EventBuildComplete {
		t.Errorf("Expected second event type %s, got %s", EventBuildComplete, history[1].Type)
	}
}

func TestEventHub_PublishSetsTimestamp(t *testing.T) {
	hub := NewEventHub(50)

	before := time.Now()
	hub.Publish(DeployEvent{
		Type:    EventDeployStarted,
		AppID:   "app-1",
		Message: "deploying",
	})
	after := time.Now()

	history := hub.History()
	if len(history) != 1 {
		t.Fatal("Expected 1 event")
	}
	ts := history[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("Timestamp %v not between %v and %v", ts, before, after)
	}
}

func TestEventHub_PublishPreservesExistingTimestamp(t *testing.T) {
	hub := NewEventHub(50)

	customTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	hub.Publish(DeployEvent{
		Type:      EventDeployComplete,
		AppID:     "app-1",
		Timestamp: customTime,
	})

	history := hub.History()
	if !history[0].Timestamp.Equal(customTime) {
		t.Errorf("Expected custom timestamp preserved, got %v", history[0].Timestamp)
	}
}

func TestEventHub_RingBuffer(t *testing.T) {
	hub := NewEventHub(3)

	for i := 0; i < 5; i++ {
		hub.Publish(DeployEvent{
			Type:    EventBuildProgress,
			Message: string(rune('A' + i)),
		})
	}

	history := hub.History()
	if len(history) != 3 {
		t.Fatalf("Expected 3 events (ring buffer), got %d", len(history))
	}
	// Oldest should be 'C' (index 2), then 'D', then 'E'
	if history[0].Message != "C" {
		t.Errorf("Expected oldest event 'C', got '%s'", history[0].Message)
	}
	if history[2].Message != "E" {
		t.Errorf("Expected newest event 'E', got '%s'", history[2].Message)
	}
}

func TestEventHub_Subscribe_ReceivesLiveEvents(t *testing.T) {
	hub := NewEventHub(50)

	sub := hub.Subscribe(10)
	defer sub.Close()

	hub.Publish(DeployEvent{
		Type:    EventDeployComplete,
		AppID:   "app-1",
		Message: "live event",
	})

	select {
	case event := <-sub.Ch:
		if event.Message != "live event" {
			t.Errorf("Expected 'live event', got '%s'", event.Message)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for live event")
	}
}

func TestEventHub_Subscribe_ReplaysHistory(t *testing.T) {
	hub := NewEventHub(50)

	hub.Publish(DeployEvent{Type: EventDeploymentStarted, Message: "past-1"})
	hub.Publish(DeployEvent{Type: EventBuildComplete, Message: "past-2"})

	sub := hub.Subscribe(10)
	defer sub.Close()

	// Should receive replayed events
	for i, expected := range []string{"past-1", "past-2"} {
		select {
		case event := <-sub.Ch:
			if event.Message != expected {
				t.Errorf("Replay[%d]: expected '%s', got '%s'", i, expected, event.Message)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Timed out waiting for replayed event %d", i)
		}
	}
}

func TestEventHub_Subscribe_DefaultBufferSize(t *testing.T) {
	hub := NewEventHub(50)

	sub := hub.Subscribe(0) // should default to 50
	defer sub.Close()

	if cap(sub.Ch) != 50 {
		t.Errorf("Expected default buffer size 50, got %d", cap(sub.Ch))
	}
}

func TestEventHub_Subscribe_NegativeBufferSize(t *testing.T) {
	hub := NewEventHub(50)

	sub := hub.Subscribe(-5) // should default to 50
	defer sub.Close()

	if cap(sub.Ch) != 50 {
		t.Errorf("Expected default buffer size 50, got %d", cap(sub.Ch))
	}
}

func TestEventHub_MultipleSubscribers(t *testing.T) {
	hub := NewEventHub(50)

	sub1 := hub.Subscribe(10)
	sub2 := hub.Subscribe(10)
	defer sub1.Close()
	defer sub2.Close()

	hub.Publish(DeployEvent{Type: EventDeployComplete, Message: "fanout"})

	for i, sub := range []*EventSubscriber{sub1, sub2} {
		select {
		case event := <-sub.Ch:
			if event.Message != "fanout" {
				t.Errorf("sub%d: expected 'fanout', got '%s'", i+1, event.Message)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("sub%d: timed out", i+1)
		}
	}
}

func TestEventHub_ClosedSubscriberRemoved(t *testing.T) {
	hub := NewEventHub(50)

	sub := hub.Subscribe(10)
	sub.Close()

	// Publishing after subscriber close should not panic
	hub.Publish(DeployEvent{Type: EventDeployFailed, Message: "after-close"})

	// History should still work
	history := hub.History()
	if len(history) != 1 {
		t.Errorf("Expected 1 event in history, got %d", len(history))
	}
}

func TestEventSubscriber_DoubleClose(t *testing.T) {
	sub := &EventSubscriber{
		Ch:   make(chan DeployEvent, 5),
		done: make(chan struct{}),
	}

	sub.Close()
	sub.Close() // should not panic
}

func TestEventHub_EmptyHistory(t *testing.T) {
	hub := NewEventHub(50)
	history := hub.History()
	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d events", len(history))
	}
}

func TestEventHub_HistoryReturnsCopy(t *testing.T) {
	hub := NewEventHub(50)
	hub.Publish(DeployEvent{Type: EventDeployComplete, Message: "original"})

	history := hub.History()
	history[0].Message = "modified"

	// Original should not be affected
	history2 := hub.History()
	if history2[0].Message != "original" {
		t.Error("History should return a copy, not a reference")
	}
}

func TestEventHub_Concurrency(t *testing.T) {
	hub := NewEventHub(1000)

	var wg sync.WaitGroup

	// 10 concurrent publishers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				hub.Publish(DeployEvent{Type: EventBuildProgress, Message: "concurrent"})
			}
		}()
	}

	// 5 concurrent subscribers
	subs := make([]*EventSubscriber, 5)
	for i := 0; i < 5; i++ {
		subs[i] = hub.Subscribe(20)
	}

	wg.Wait()

	for _, sub := range subs {
		sub.Close()
	}

	history := hub.History()
	if len(history) == 0 {
		t.Error("Expected some history entries after concurrent writes")
	}
}
