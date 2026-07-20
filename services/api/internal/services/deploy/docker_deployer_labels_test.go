package deploy

import (
	"strings"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestProxyLabels_Traefik(t *testing.T) {
	app := &entities.App{ID: "app-123", Subdomain: "my-app", Port: 8000}
	labels := proxyLabels(app, "apps.example.com", "zenith_default")

	want := map[string]string{
		"zenith.app.id":          "app-123",
		"traefik.enable":         "true",
		"traefik.docker.network": "zenith_default",
		"traefik.http.routers.my-app.rule":                      "Host(`my-app.apps.example.com`)",
		"traefik.http.routers.my-app.entrypoints":               "websecure",
		"traefik.http.routers.my-app.tls.certresolver":          "le",
		"traefik.http.services.my-app.loadbalancer.server.port": "8000",
	}
	for k, v := range want {
		if labels[k] != v {
			t.Errorf("label %q = %q, want %q", k, labels[k], v)
		}
	}
	// The Traefik migration must leave no caddy labels behind.
	for k := range labels {
		if strings.HasPrefix(k, "caddy") {
			t.Errorf("unexpected leftover caddy label %q", k)
		}
	}
}

func TestHostRule(t *testing.T) {
	if got := hostRule([]string{"a.example.com"}); got != "Host(`a.example.com`)" {
		t.Errorf("single host: got %q", got)
	}
	if got := hostRule([]string{"a.com", "b.com"}); got != "Host(`a.com`) || Host(`b.com`)" {
		t.Errorf("multi host: got %q", got)
	}
}
