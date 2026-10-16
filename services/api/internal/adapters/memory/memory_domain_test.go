package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestDomainCheckRoundTrip(t *testing.T) {
	r := NewMemoryDomainRepository()
	ctx := context.Background()
	d, err := r.AddDomain(ctx, "app-1", "user-1", "a.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != entities.DomainStatusActive || d.TLSReady {
		t.Fatalf("AddDomain default = %s/%v, want active/false (the handler decides pending)", d.Status, d.TLSReady)
	}
	at := time.Date(2026, 10, 16, 19, 0, 0, 0, time.UTC)
	if err := r.UpdateDomainCheck(ctx, d.ID, entities.DomainStatusPending, false, "DNS: no A/AAAA record for a.example.com; create an A record to 1.2.3.4", at); err != nil {
		t.Fatal(err)
	}
	got, _ := r.GetDomain(ctx, d.ID)
	if got.Status != entities.DomainStatusPending || got.StatusMessage == "" || got.LastCheckedAt == nil || !got.LastCheckedAt.Equal(at) {
		t.Fatalf("got %+v", got)
	}
	all, _ := r.ListAllDomains(ctx)
	if len(all) != 1 || all[0].ID != d.ID {
		t.Fatalf("ListAllDomains = %+v", all)
	}
	if err := r.UpdateDomainCheck(ctx, "missing", entities.DomainStatusActive, true, "", at); err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestAddDomainDoesNotFlipStatusByItself(t *testing.T) {
	r := NewMemoryDomainRepository()
	d, _ := r.AddDomain(context.Background(), "app-1", "user-1", "b.example.com")
	time.Sleep(10 * time.Millisecond)
	got, _ := r.GetDomain(context.Background(), d.ID)
	if got.Status != entities.DomainStatusActive {
		t.Fatalf("status changed to %s without a check", got.Status)
	}
}
