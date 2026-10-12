package handlers_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/gofiber/fiber/v2"
)

// projectDBRepo returns every database for any project id, as a project
// shared between (or guessed by) two users would.
type projectDBRepo struct {
	*memory.MemoryDatabaseRepository
}

func (r projectDBRepo) ListDatabasesByProject(ctx context.Context, _ string) ([]entities.UserDatabase, error) {
	a, _ := r.ListDatabasesByUser(ctx, "user-1")
	b, _ := r.ListDatabasesByUser(ctx, "user-2")
	return append(a, b...), nil
}

func TestDatabaseListByUser_ProjectFilterHidesOtherUsers(t *testing.T) {
	ctx := context.Background()
	repo := projectDBRepo{memory.NewMemoryDatabaseRepository()}
	if _, err := repo.CreateDatabase(ctx, "", "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "mine"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateDatabase(ctx, "", "user-2", &dto.CreateDatabaseInput{Engine: entities.DatabaseEngineRedis, Name: "theirs"}); err != nil {
		t.Fatal(err)
	}
	h := handlers.NewDatabaseHandlerV2(nil, repo, memory.NewMemoryAppRepository(), nil)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Get("/api/v1/databases", injectUserID("user-1"), h.ListByUser)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/databases?project_id=p-other", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	var got []dto.DatabaseInfo
	_ = json.NewDecoder(resp.Body).Decode(&got)
	if len(got) != 1 || got[0].Name != "mine" {
		t.Fatalf("got %+v, want only user-1's database", got)
	}
}
