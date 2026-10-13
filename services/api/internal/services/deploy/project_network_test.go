package deploy

import (
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestProjectNetworkName(t *testing.T) {
	if got := projectNetworkName("7f3e2a10-9b1c-4d2e-8f00-123456789abc"); got != "zenith-proj-7f3e2a10-9b1" {
		t.Fatalf("got %q", got)
	}
	if got := projectNetworkName(""); got != "" {
		t.Fatalf("empty project id gave %q", got)
	}
}

func TestProjectAlias_ServiceNameWins(t *testing.T) {
	app := &entities.App{Name: "shop-db", ServiceName: "DB"}
	if got := projectAlias(app, "shop"); got != "db" {
		t.Fatalf("got %q", got)
	}
}

func TestProjectAlias_PrefixFallback(t *testing.T) {
	app := &entities.App{Name: "shop-db"}
	if got := projectAlias(app, "shop"); got != "db" {
		t.Fatalf("got %q", got)
	}
}

func TestProjectAlias_None(t *testing.T) {
	for _, tc := range []struct {
		app  *entities.App
		slug string
	}{
		{&entities.App{Name: "standalone"}, "shop"},
		{&entities.App{Name: "shop-"}, "shop"},
		{&entities.App{Name: "shop-db"}, ""},
	} {
		if got := projectAlias(tc.app, tc.slug); got != "" {
			t.Errorf("app %q slug %q: got %q, want none", tc.app.Name, tc.slug, got)
		}
	}
}
