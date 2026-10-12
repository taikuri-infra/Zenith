package deploy

import (
	"reflect"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

// Regression (F-21 reopened): every app, database and the control plane share
// one network, so a compose service must never get its bare service name as
// an alias — "postgres" or "storage" would shadow the hosts the API uses.
func TestNetworkAliasesNeverBareComposeServiceName(t *testing.T) {
	// App "shop-postgres" from a compose import into project slug "shop".
	app := &entities.App{Name: "shop-postgres", Subdomain: "shop-postgres-1a2b", ProjectID: "p1"}
	want := []string{"shop-postgres", "shop-postgres-1a2b"}
	if got := networkAliases(app); !reflect.DeepEqual(got, want) {
		t.Fatalf("aliases = %v, want exactly %v", got, want)
	}
}

func TestProxyLabels_PrivateAppHasNoRouter(t *testing.T) {
	app := &entities.App{ID: "app-9", Subdomain: "shop-db", Port: 5432, Exposure: entities.ExposureProtected}
	labels := proxyLabels(app, []string{"shop-db.example.com"}, "zenith_default", "le")
	want := map[string]string{"zenith.app.id": "app-9"}
	if !reflect.DeepEqual(labels, want) {
		t.Fatalf("private app labels = %v, want only %v", labels, want)
	}
}

func TestProxyLabels_PublicAndUnsetExposureAreRouted(t *testing.T) {
	for _, exp := range []entities.AppExposure{entities.ExposurePublic, ""} {
		app := &entities.App{ID: "a", Subdomain: "x", Port: 80, Exposure: exp}
		labels := proxyLabels(app, []string{"x.example.com"}, "zenith_default", "le")
		if labels["traefik.enable"] != "true" || labels["traefik.http.routers.x.rule"] == "" {
			t.Errorf("exposure %q: want a Traefik router, got %v", exp, labels)
		}
	}
}

func TestNetworkAliases_NeverShadowControlPlane(t *testing.T) {
	for _, name := range []string{"postgres", "storage", "api", "web", "traefik", "Zenith-Postgres-1", "zenith-db-1a2b3c4d"} {
		app := &entities.App{Name: name, Subdomain: name}
		if got := networkAliases(app); len(got) != 0 {
			t.Errorf("app %q got aliases %v, want none (shadows a control-plane host)", name, got)
		}
	}
}
