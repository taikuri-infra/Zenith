package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestUserEventTrack(t *testing.T) {
	repo := NewMemoryUserEventRepository()
	ctx := context.Background()

	event := &entities.UserEvent{
		UserID:    "user-1",
		EventType: entities.EventSignup,
		Properties: map[string]interface{}{
			"source": "organic",
		},
		IPAddress: "1.2.3.4",
		UserAgent: "test-agent",
	}

	err := repo.Track(ctx, event)
	if err != nil {
		t.Fatalf("Track: expected no error, got %v", err)
	}
	if event.ID == "" {
		t.Error("Track: expected ID to be auto-generated")
	}
	if event.CreatedAt.IsZero() {
		t.Error("Track: expected CreatedAt to be set")
	}
}

func TestUserEventListByUser(t *testing.T) {
	repo := NewMemoryUserEventRepository()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		repo.Track(ctx, &entities.UserEvent{UserID: "user-1", EventType: entities.EventLogin})
	}
	repo.Track(ctx, &entities.UserEvent{UserID: "user-2", EventType: entities.EventLogin})

	events, err := repo.ListByUser(ctx, "user-1", 10, 0)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(events) != 5 {
		t.Errorf("ListByUser: expected 5 events, got %d", len(events))
	}

	// Test limit
	events, err = repo.ListByUser(ctx, "user-1", 2, 0)
	if err != nil {
		t.Fatalf("ListByUser limit: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("ListByUser limit: expected 2 events, got %d", len(events))
	}
}

func TestUserEventListByType(t *testing.T) {
	repo := NewMemoryUserEventRepository()
	ctx := context.Background()

	repo.Track(ctx, &entities.UserEvent{UserID: "u1", EventType: entities.EventSignup})
	repo.Track(ctx, &entities.UserEvent{UserID: "u2", EventType: entities.EventSignup})
	repo.Track(ctx, &entities.UserEvent{UserID: "u1", EventType: entities.EventLogin})

	events, err := repo.ListByType(ctx, entities.EventSignup, 10, 0)
	if err != nil {
		t.Fatalf("ListByType: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("ListByType: expected 2 signup events, got %d", len(events))
	}
}

func TestUserEventCountByType(t *testing.T) {
	repo := NewMemoryUserEventRepository()
	ctx := context.Background()

	since := time.Now().Add(-1 * time.Hour)
	repo.Track(ctx, &entities.UserEvent{UserID: "u1", EventType: entities.EventAppCreate})
	repo.Track(ctx, &entities.UserEvent{UserID: "u2", EventType: entities.EventAppCreate})
	repo.Track(ctx, &entities.UserEvent{UserID: "u3", EventType: entities.EventLogin})

	count, err := repo.CountByType(ctx, entities.EventAppCreate, since)
	if err != nil {
		t.Fatalf("CountByType: %v", err)
	}
	if count != 2 {
		t.Errorf("CountByType: expected 2, got %d", count)
	}
}

func TestUserEventGetFunnelData(t *testing.T) {
	repo := NewMemoryUserEventRepository()
	ctx := context.Background()

	since := time.Now().Add(-1 * time.Hour)
	repo.Track(ctx, &entities.UserEvent{UserID: "u1", EventType: entities.EventSignup})
	repo.Track(ctx, &entities.UserEvent{UserID: "u2", EventType: entities.EventSignup})
	repo.Track(ctx, &entities.UserEvent{UserID: "u1", EventType: entities.EventAppCreate})

	funnel, err := repo.GetFunnelData(ctx, []string{entities.EventSignup, entities.EventAppCreate}, since)
	if err != nil {
		t.Fatalf("GetFunnelData: %v", err)
	}
	if funnel[entities.EventSignup] != 2 {
		t.Errorf("funnel signup: expected 2, got %d", funnel[entities.EventSignup])
	}
	if funnel[entities.EventAppCreate] != 1 {
		t.Errorf("funnel app.create: expected 1, got %d", funnel[entities.EventAppCreate])
	}
}

func TestUserEventPurgeOlderThan(t *testing.T) {
	repo := NewMemoryUserEventRepository()
	ctx := context.Background()

	// Track old events
	old := &entities.UserEvent{UserID: "u1", EventType: entities.EventLogin, CreatedAt: time.Now().Add(-100 * 24 * time.Hour)}
	old.ID = "old-1"
	repo.Track(ctx, old)

	// Track recent event
	repo.Track(ctx, &entities.UserEvent{UserID: "u1", EventType: entities.EventLogin})

	purged, err := repo.PurgeOlderThan(ctx, time.Now().Add(-90*24*time.Hour))
	if err != nil {
		t.Fatalf("PurgeOlderThan: %v", err)
	}
	if purged != 1 {
		t.Errorf("PurgeOlderThan: expected 1 purged, got %d", purged)
	}

	remaining, _ := repo.ListByUser(ctx, "u1", 10, 0)
	if len(remaining) != 1 {
		t.Errorf("after purge: expected 1 remaining, got %d", len(remaining))
	}
}
