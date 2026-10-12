// services/api/internal/services/db_runtime_test.go
package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
)

type fakeDBBackend struct {
	waitErr error
	gotPW   string
}

func (f *fakeDBBackend) Provision(_ context.Context, db *entities.UserDatabase, pw string) (string, int, error) {
	f.gotPW = pw
	return "zenith-db-" + db.ID[:8], 6379, nil
}
func (f *fakeDBBackend) WaitReady(context.Context, *entities.UserDatabase) error    { return f.waitErr }
func (f *fakeDBBackend) Delete(context.Context, *entities.UserDatabase, bool) error { return nil }

func TestDBService_ProvisionsAndStoresCredentials(t *testing.T) {
	repo := memory.NewMemoryDatabaseRepository()
	be := &fakeDBBackend{}
	s := NewDBRuntimeService(repo, be, nil)
	db, err := s.Create(context.Background(), "app-1", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEngineRedis, Name: "cache"})
	if err != nil {
		t.Fatal(err)
	}
	s.Wait(db.ID)
	got, _ := repo.GetDatabase(context.Background(), db.ID)
	if got.Status != entities.DatabaseStatusReady || got.Host == "" || got.Port != 6379 {
		t.Fatalf("got %+v", got)
	}
	pw, err := s.Password(context.Background(), got)
	if err != nil || pw == "" || pw != be.gotPW || len(pw) < 24 {
		t.Errorf("password %q (%v), backend saw %q", pw, err, be.gotPW)
	}
}

func TestDBService_ProvisionFailureMarksError(t *testing.T) {
	repo := memory.NewMemoryDatabaseRepository()
	s := NewDBRuntimeService(repo, &fakeDBBackend{waitErr: errors.New("unhealthy")}, nil)
	db, _ := s.Create(context.Background(), "app-1", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "pg"})
	s.Wait(db.ID)
	got, _ := repo.GetDatabase(context.Background(), db.ID)
	if got.Status != entities.DatabaseStatusError {
		t.Errorf("status = %q, want error", got.Status)
	}
}

func TestDBService_RejectsUnsupportedEngine(t *testing.T) {
	s := NewDBRuntimeService(memory.NewMemoryDatabaseRepository(), &fakeDBBackend{}, nil)
	_, err := s.Create(context.Background(), "app-1", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEngineKafka})
	if err == nil {
		t.Error("kafka must be rejected in the Docker runtime")
	}
}

type ctxBlockBackend struct{ fakeDBBackend }

func (b *ctxBlockBackend) WaitReady(ctx context.Context, _ *entities.UserDatabase) error {
	<-ctx.Done()
	return ctx.Err()
}

// ctxStrictRepo fails status updates made on an already-expired context,
// like a real database driver would.
type ctxStrictRepo struct{ ports.DatabaseRepository }

func (r ctxStrictRepo) UpdateDatabaseStatus(ctx context.Context, id string, st entities.DatabaseStatus) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.DatabaseRepository.UpdateDatabaseStatus(ctx, id, st)
}

func TestDBService_TimeoutStillMarksError(t *testing.T) {
	repo := ctxStrictRepo{memory.NewMemoryDatabaseRepository()}
	s := NewDBRuntimeService(repo, &ctxBlockBackend{}, nil)
	s.provisionTimeout = 50 * time.Millisecond

	db, err := s.Create(context.Background(), "app-1", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "pg"})
	if err != nil {
		t.Fatal(err)
	}
	s.Wait(db.ID)
	got, _ := repo.GetDatabase(context.Background(), db.ID)
	if got.Status != entities.DatabaseStatusError {
		t.Errorf("status = %q, want error", got.Status)
	}
}
