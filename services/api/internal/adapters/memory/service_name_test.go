package memory_test

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestCreateApp_KeepsServiceName(t *testing.T) {
	repo := memory.NewMemoryAppRepository()
	a, err := repo.CreateApp(context.Background(), &dto.CreateAppInput{UserID: "u1", Name: "shop-db", ServiceName: "db", ImageURL: "postgres:16", DeploySource: entities.DeploySourceImage})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetApp(context.Background(), a.ID)
	if got.ServiceName != "db" {
		t.Fatalf("ServiceName = %q, want db", got.ServiceName)
	}
}

func TestDatabaseHostPortAndStatusList(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMemoryDatabaseRepository()
	db, err := repo.CreateDatabase(ctx, "", "u1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "pg"})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetDatabaseHostPort(ctx, db.ID, 49153); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetDatabase(ctx, db.ID)
	if got.HostPort != 49153 {
		t.Fatalf("HostPort = %d", got.HostPort)
	}
	ready, _ := repo.ListDatabasesByStatus(ctx, entities.DatabaseStatusReady)
	if len(ready) != 1 || ready[0].ID != db.ID {
		t.Fatalf("ready = %+v", ready)
	}
}
