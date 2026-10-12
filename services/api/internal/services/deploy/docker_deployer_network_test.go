package deploy

import (
	"context"
	"reflect"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestNetworkAliases(t *testing.T) {
	cases := []struct {
		name        string
		app         entities.App
		projectSlug string
		want        []string
	}{
		{"compose service gets its bare service name",
			entities.App{Name: "shop-db", Subdomain: "shop-db-1a2b", ProjectID: "p1"}, "shop",
			[]string{"shop-db", "shop-db-1a2b", "db"}},
		{"no project slug",
			entities.App{Name: "shop-db", Subdomain: "shop-db-1a2b"}, "",
			[]string{"shop-db", "shop-db-1a2b"}},
		{"name without the project prefix",
			entities.App{Name: "web", Subdomain: "web-9", ProjectID: "p1"}, "shop",
			[]string{"web", "web-9"}},
		{"service name is sanitized",
			entities.App{Name: "shop-My_API", Subdomain: "shop-my-api-3c4d", ProjectID: "p1"}, "shop",
			[]string{"shop-my_api", "shop-my-api-3c4d", "my_api"}},
		{"name equal to the prefix adds nothing",
			entities.App{Name: "shop-", Subdomain: "shop-x", ProjectID: "p1"}, "shop",
			[]string{"shop", "shop-x"}},
	}
	for _, c := range cases {
		if got := networkAliases(&c.app, c.projectSlug); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestProjectSlugLookup(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMemoryProjectRepository()
	p, err := repo.CreateProject(ctx, "u1", "Shop", "shop", "")
	if err != nil {
		t.Fatal(err)
	}
	d := &DockerDeployer{}
	if got := d.projectSlug(ctx, &entities.App{ProjectID: p.ID}); got != "" {
		t.Errorf("no repo: got %q", got)
	}
	d.SetProjectRepo(repo)
	if got := d.projectSlug(ctx, &entities.App{ProjectID: p.ID}); got != "shop" {
		t.Errorf("got %q, want shop", got)
	}
	if got := d.projectSlug(ctx, &entities.App{}); got != "" {
		t.Errorf("no project: got %q", got)
	}
	if got := d.projectSlug(ctx, &entities.App{ProjectID: "missing"}); got != "" {
		t.Errorf("missing project: got %q", got)
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
