package memory

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestMemoryFirewallRepo(t *testing.T) {
	ctx := context.Background()
	r := NewFirewallRuleRepository()
	now := time.Now()
	add := func(id, app string, prio int) {
		if err := r.Create(ctx, &entities.NetworkPolicyRule{ID: id, AppID: app, Priority: prio, Enabled: true, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	add("r2", "A", 5)
	add("r1", "A", 1)
	add("r3", "B", 0)

	got, _ := r.ListByApp(ctx, "A")
	if len(got) != 2 || got[0].ID != "r1" || got[1].ID != "r2" {
		t.Fatalf("ListByApp(A) = %+v", got)
	}
	all, _ := r.ListAll(ctx)
	if len(all) != 3 {
		t.Fatalf("ListAll = %d", len(all))
	}
	rule, _ := r.Get(ctx, "r1")
	rule.Enabled = false
	if err := r.Update(ctx, rule); err != nil {
		t.Fatal(err)
	}
	if g, _ := r.Get(ctx, "r1"); g.Enabled {
		t.Fatal("update not stored")
	}
	if err := r.Delete(ctx, "r1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(ctx, "r1"); err == nil {
		t.Fatal("expected error for deleted rule")
	}
}
