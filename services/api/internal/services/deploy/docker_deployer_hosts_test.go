package deploy

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestAppHosts(t *testing.T) {
	ctx := context.Background()
	app := &entities.App{ID: "app-1", Name: "web", Subdomain: "web"}
	d := &DockerDeployer{baseDomain: "example.com"}

	if got := d.appHosts(ctx, app); !reflect.DeepEqual(got, []string{"web.example.com"}) {
		t.Fatalf("nil domainRepo: got %v", got)
	}

	repo := memory.NewMemoryDomainRepository()
	d.SetDomainRepo(repo)
	ids := map[string]string{}
	for _, c := range []struct {
		name   string
		status entities.DomainStatus
	}{
		{"a.example.org", entities.DomainStatusActive},
		{"b.example.org", entities.DomainStatusVerified},
		{"c.example.org", entities.DomainStatusPending},
		{"d.example.org", entities.DomainStatusFailed},
	} {
		cd, err := repo.AddDomain(ctx, app.ID, "user-1", c.name)
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.UpdateDomainStatus(ctx, cd.ID, c.status, false); err != nil {
			t.Fatal(err)
		}
		ids[c.name] = cd.ID
	}

	want := []string{"web.example.com", "a.example.org", "b.example.org", "c.example.org", "d.example.org"}
	if got := sortedCustom(d.appHosts(ctx, app)); !reflect.DeepEqual(got, want) {
		t.Fatalf("hosts = %v, want %v", got, want)
	}

	if err := repo.DeleteDomain(ctx, ids["a.example.org"]); err != nil {
		t.Fatal(err)
	}
	want = []string{"web.example.com", "b.example.org", "c.example.org", "d.example.org"}
	if got := sortedCustom(d.appHosts(ctx, app)); !reflect.DeepEqual(got, want) {
		t.Fatalf("after delete hosts = %v, want %v", got, want)
	}
}

func TestAppHostsSkipsInvalidStoredDomain(t *testing.T) {
	ctx := context.Background()
	app := &entities.App{ID: "app-1", Name: "web", Subdomain: "web"}
	d := &DockerDeployer{baseDomain: "example.com"}
	repo := memory.NewMemoryDomainRepository()
	d.SetDomainRepo(repo)
	for _, name := range []string{"x`) || PathPrefix(`/", "ok.example.org"} {
		cd, err := repo.AddDomain(ctx, app.ID, "user-1", name)
		if err != nil {
			t.Fatal(err)
		}
		_ = repo.UpdateDomainStatus(ctx, cd.ID, entities.DomainStatusActive, false)
	}
	want := []string{"web.example.com", "ok.example.org"}
	if got := d.appHosts(ctx, app); !reflect.DeepEqual(got, want) {
		t.Fatalf("hosts = %v, want %v", got, want)
	}
}

// sortedCustom sorts the custom domains after the generated host; the memory
// domain repo lists in map order.
func sortedCustom(hosts []string) []string {
	if len(hosts) > 1 {
		sort.Strings(hosts[1:])
	}
	return hosts
}
