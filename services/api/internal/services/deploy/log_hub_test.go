package deploy

import (
	"sync"
	"testing"
	"time"
)

func TestLogHubPublishAndHistory(t *testing.T) {
	hub := NewLogHub(100)

	hub.PublishInfo("deploy-1", "cloning repo...")
	hub.PublishBuild("deploy-1", "building image...")
	hub.PublishDeploy("deploy-1", "applying manifests...")

	history := hub.History("deploy-1")
	if len(history) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(history))
	}

	if history[0].Level != "info" || history[0].Message != "cloning repo..." {
		t.Errorf("unexpected first entry: %+v", history[0])
	}
	if history[1].Level != "build" {
		t.Errorf("expected build level, got %s", history[1].Level)
	}
	if history[2].Level != "deploy" {
		t.Errorf("expected deploy level, got %s", history[2].Level)
	}
}

func TestLogHubHistoryIsolation(t *testing.T) {
	hub := NewLogHub(100)

	hub.PublishInfo("deploy-1", "msg-1")
	hub.PublishInfo("deploy-2", "msg-2")

	h1 := hub.History("deploy-1")
	h2 := hub.History("deploy-2")

	if len(h1) != 1 || h1[0].Message != "msg-1" {
		t.Errorf("deploy-1 history wrong: %+v", h1)
	}
	if len(h2) != 1 || h2[0].Message != "msg-2" {
		t.Errorf("deploy-2 history wrong: %+v", h2)
	}
}

func TestLogHubRingBuffer(t *testing.T) {
	hub := NewLogHub(3) // only keep last 3

	hub.PublishInfo("d1", "line-1")
	hub.PublishInfo("d1", "line-2")
	hub.PublishInfo("d1", "line-3")
	hub.PublishInfo("d1", "line-4")

	history := hub.History("d1")
	if len(history) != 3 {
		t.Fatalf("expected 3 entries (ring buffer), got %d", len(history))
	}
	if history[0].Message != "line-2" {
		t.Errorf("expected oldest entry 'line-2', got '%s'", history[0].Message)
	}
}

func TestLogHubSubscribe(t *testing.T) {
	hub := NewLogHub(100)

	// Publish before subscribing
	hub.PublishInfo("d1", "before-sub")

	sub := hub.Subscribe("d1", 10)
	defer sub.Close()

	// Should receive replayed history
	select {
	case entry := <-sub.Ch:
		if entry.Message != "before-sub" {
			t.Errorf("expected replayed 'before-sub', got '%s'", entry.Message)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for replayed entry")
	}

	// Publish after subscribing
	hub.PublishInfo("d1", "after-sub")

	select {
	case entry := <-sub.Ch:
		if entry.Message != "after-sub" {
			t.Errorf("expected 'after-sub', got '%s'", entry.Message)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for live entry")
	}
}

func TestLogHubMultipleSubscribers(t *testing.T) {
	hub := NewLogHub(100)

	sub1 := hub.Subscribe("d1", 10)
	sub2 := hub.Subscribe("d1", 10)
	defer sub1.Close()
	defer sub2.Close()

	hub.PublishInfo("d1", "fanout-msg")

	for i, sub := range []*LogSubscriber{sub1, sub2} {
		select {
		case entry := <-sub.Ch:
			if entry.Message != "fanout-msg" {
				t.Errorf("sub%d: expected 'fanout-msg', got '%s'", i+1, entry.Message)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("sub%d: timed out", i+1)
		}
	}
}

func TestLogHubCleanup(t *testing.T) {
	hub := NewLogHub(100)

	hub.PublishInfo("d1", "before-cleanup")
	sub := hub.Subscribe("d1", 10)
	// drain replay
	<-sub.Ch

	hub.Cleanup("d1")

	// History should be empty
	if len(hub.History("d1")) != 0 {
		t.Error("expected empty history after cleanup")
	}

	// Channel should be closed
	_, open := <-sub.Ch
	if open {
		t.Error("expected subscriber channel to be closed after cleanup")
	}
}

func TestLogHubConcurrency(t *testing.T) {
	hub := NewLogHub(1000)

	var wg sync.WaitGroup

	// 10 concurrent publishers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				hub.PublishInfo("d1", "msg")
			}
		}(i)
	}

	// 5 concurrent subscribers
	subs := make([]*LogSubscriber, 5)
	for i := 0; i < 5; i++ {
		subs[i] = hub.Subscribe("d1", 50)
	}

	wg.Wait()

	for _, sub := range subs {
		sub.Close()
	}

	// Should not panic or deadlock — the test passing is the assertion
	history := hub.History("d1")
	if len(history) == 0 {
		t.Error("expected some history entries after concurrent writes")
	}
}

func TestLogHubTimestamp(t *testing.T) {
	hub := NewLogHub(100)

	before := time.Now()
	hub.PublishInfo("d1", "timestamped")
	after := time.Now()

	history := hub.History("d1")
	if len(history) != 1 {
		t.Fatal("expected 1 entry")
	}

	ts := history[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not between %v and %v", ts, before, after)
	}
}

func TestLogHubEmptyHistory(t *testing.T) {
	hub := NewLogHub(100)

	history := hub.History("nonexistent")
	if len(history) != 0 {
		t.Errorf("expected empty history for nonexistent deployment, got %d", len(history))
	}
}

func TestLogHubPublishError(t *testing.T) {
	hub := NewLogHub(100)

	hub.PublishError("d1", "something failed")

	history := hub.History("d1")
	if len(history) != 1 || history[0].Level != "error" {
		t.Errorf("expected error entry, got %+v", history)
	}
}

func TestLogHubSubscriberClose(t *testing.T) {
	hub := NewLogHub(100)

	sub := hub.Subscribe("d1", 10)
	sub.Close()

	// Double close should not panic
	sub.Close()

	// Publishing after subscriber close should not block or panic
	hub.PublishInfo("d1", "after-close")
}
