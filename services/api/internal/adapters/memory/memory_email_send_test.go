package memory

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func newEmailSend(userID, template string) *entities.EmailSend {
	return &entities.EmailSend{
		UserID:      userID,
		TemplateKey: template,
	}
}

func TestEmailSendRecord(t *testing.T) {
	repo := NewMemoryEmailSendRepository()
	ctx := context.Background()

	err := repo.Record(ctx, newEmailSend("user-1", "welcome"))
	if err != nil {
		t.Fatalf("Record: expected no error, got %v", err)
	}

	sent, err := repo.HasSent(ctx, "user-1", "welcome")
	if err != nil {
		t.Fatalf("HasSent: %v", err)
	}
	if !sent {
		t.Error("HasSent: expected true after Record")
	}
}

func TestEmailSendDuplicate(t *testing.T) {
	repo := NewMemoryEmailSendRepository()
	ctx := context.Background()

	repo.Record(ctx, newEmailSend("user-1", "welcome"))

	// Recording again should be idempotent (no error)
	err := repo.Record(ctx, newEmailSend("user-1", "welcome"))
	if err != nil {
		t.Fatalf("Duplicate Record: expected no error, got %v", err)
	}
}

func TestEmailSendHasSentFalse(t *testing.T) {
	repo := NewMemoryEmailSendRepository()
	ctx := context.Background()

	sent, err := repo.HasSent(ctx, "user-1", "day1_deploy")
	if err != nil {
		t.Fatalf("HasSent: %v", err)
	}
	if sent {
		t.Error("HasSent: expected false for unsent template")
	}
}

func TestEmailSendMarkOpened(t *testing.T) {
	repo := NewMemoryEmailSendRepository()
	ctx := context.Background()

	send := newEmailSend("user-1", "welcome")
	repo.Record(ctx, send)

	err := repo.MarkOpened(ctx, send.ID)
	if err != nil {
		t.Fatalf("MarkOpened: %v", err)
	}
}

func TestEmailSendMarkClicked(t *testing.T) {
	repo := NewMemoryEmailSendRepository()
	ctx := context.Background()

	send := newEmailSend("user-1", "welcome")
	repo.Record(ctx, send)

	err := repo.MarkClicked(ctx, send.ID)
	if err != nil {
		t.Fatalf("MarkClicked: %v", err)
	}
}

func TestEmailSendGetStats(t *testing.T) {
	repo := NewMemoryEmailSendRepository()
	ctx := context.Background()

	s1 := newEmailSend("user-1", "welcome")
	s2 := newEmailSend("user-2", "welcome")
	s3 := newEmailSend("user-1", "day1_deploy")
	repo.Record(ctx, s1)
	repo.Record(ctx, s2)
	repo.Record(ctx, s3)
	repo.MarkOpened(ctx, s1.ID)
	repo.MarkClicked(ctx, s1.ID)

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.Sent != 3 {
		t.Errorf("GetStats TotalSent: expected 3, got %d", stats.Sent)
	}
	if stats.Opened != 1 {
		t.Errorf("GetStats TotalOpened: expected 1, got %d", stats.Opened)
	}
	if stats.Clicked != 1 {
		t.Errorf("GetStats TotalClicked: expected 1, got %d", stats.Clicked)
	}
	if stats.ByTemplate["welcome"] != 2 {
		t.Errorf("GetStats ByTemplate[welcome]: expected 2, got %d", stats.ByTemplate["welcome"])
	}
}

func TestEmailSendListByUser(t *testing.T) {
	repo := NewMemoryEmailSendRepository()
	ctx := context.Background()

	repo.Record(ctx, newEmailSend("user-1", "welcome"))
	repo.Record(ctx, newEmailSend("user-1", "day1_deploy"))
	repo.Record(ctx, newEmailSend("user-2", "welcome"))

	sends, err := repo.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(sends) != 2 {
		t.Errorf("ListByUser: expected 2, got %d", len(sends))
	}
}
