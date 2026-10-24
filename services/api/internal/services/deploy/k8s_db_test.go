package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func testDB() *entities.UserDatabase {
	return &entities.UserDatabase{ID: "9f1c2d3e-aaaa-bbbb-cccc-111122223333", Engine: entities.DatabaseEnginePostgres,
		DBName: "app", DBUser: "app", MaxSizeMB: 2048}
}

func TestK8sDBBackend_ProvisionCreatesCluster(t *testing.T) {
	c := k8sclient.NewMemoryClient()
	b := NewK8sDBBackend(c, "zenith-apps")
	db := testDB()

	host, port, err := b.Provision(context.Background(), db, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if port != 5432 || host != k8sDBName(db)+"-rw.zenith-apps.svc.cluster.local" {
		t.Fatalf("host/port = %s:%d", host, port)
	}
	obj, err := c.GetCRDWithVersion(context.Background(), cnpgAPIVersion, "Cluster", "zenith-apps", k8sDBName(db))
	if err != nil || obj == nil {
		t.Fatalf("cluster not created: %v", err)
	}
	sec, err := c.GetSecret(context.Background(), "zenith-apps", k8sDBName(db)+"-auth")
	if err != nil || string(sec["password"]) != "pw" || string(sec["username"]) != "app" {
		t.Fatalf("credentials secret wrong: %v %v", sec, err)
	}
}

func TestK8sDBBackend_ProvisionIsIdempotent(t *testing.T) {
	b := NewK8sDBBackend(k8sclient.NewMemoryClient(), "zenith-apps")
	for i := 0; i < 2; i++ {
		if _, _, err := b.Provision(context.Background(), testDB(), "pw"); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
}

func TestK8sDBBackend_OnlyPostgres(t *testing.T) {
	b := NewK8sDBBackend(k8sclient.NewMemoryClient(), "zenith-apps")
	if !b.SupportsEngine(entities.DatabaseEnginePostgres) || b.SupportsEngine(entities.DatabaseEngineRedis) {
		t.Fatal("engine support wrong")
	}
	db := testDB()
	db.Engine = entities.DatabaseEngineRedis
	if _, _, err := b.Provision(context.Background(), db, "pw"); err == nil {
		t.Fatal("expected error for redis")
	}
}

func TestK8sDBBackend_DeleteRemovesAndToleratesMissing(t *testing.T) {
	c := k8sclient.NewMemoryClient()
	b := NewK8sDBBackend(c, "zenith-apps")
	db := testDB()
	if err := b.Delete(context.Background(), db, true); err != nil {
		t.Fatalf("delete of missing: %v", err)
	}
	_, _, _ = b.Provision(context.Background(), db, "pw")
	if err := b.Delete(context.Background(), db, true); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetSecret(context.Background(), "zenith-apps", k8sDBName(db)+"-auth"); err == nil {
		t.Fatal("secret should be gone")
	}
}

func TestClusterReady(t *testing.T) {
	cases := map[string]bool{
		``: false, `{}`: false, `{"readyInstances":0}`: false,
		`{"readyInstances":1}`: true, `{"phase":"Cluster in healthy state"}`: true, `not json`: false,
	}
	for in, want := range cases {
		if got := clusterReady([]byte(in)); got != want {
			t.Errorf("clusterReady(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestK8sDBBackend_WaitReadyHonoursContext(t *testing.T) {
	b := NewK8sDBBackend(k8sclient.NewMemoryClient(), "zenith-apps")
	b.pollEvery = 10 * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	if err := b.WaitReady(ctx, testDB()); err == nil {
		t.Fatal("expected timeout error")
	}
}
