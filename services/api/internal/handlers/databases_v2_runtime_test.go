package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

type okDBBackend struct{}

func (okDBBackend) Provision(_ context.Context, db *entities.UserDatabase, _ string) (string, int, error) {
	return "zenith-db-" + db.ID[:8], 5432, nil
}
func (okDBBackend) WaitReady(context.Context, *entities.UserDatabase) error    { return nil }
func (okDBBackend) Delete(context.Context, *entities.UserDatabase, bool) error { return nil }

func TestDatabaseCreateForApp_InjectsDatabaseURL(t *testing.T) {
	ctx := context.Background()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	a, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	rt := services.NewDBRuntimeService(dbRepo, okDBBackend{}, nil)
	h := handlers.NewDatabaseHandlerV2(nil, dbRepo, appRepo, nil)
	h.SetRuntime(rt)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/api/v1/apps/:appId/databases", injectUserID("user-1"), h.Create)

	req := httptest.NewRequest("POST", "/api/v1/apps/"+a.ID+"/databases", bytes.NewBufferString(`{"engine":"postgresql","name":"main"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req, -1)
	if resp.StatusCode != 201 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var info dto.DatabaseInfo
	_ = json.NewDecoder(resp.Body).Decode(&info)
	rt.Wait(info.ID)

	// Injection runs in a background goroutine after provisioning; poll for it.
	var url string
	for deadline := time.Now().Add(2 * time.Second); url == "" && time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		vars, _ := appRepo.GetEnvVars(ctx, a.ID)
		for _, v := range vars {
			if v.Key == "DATABASE_URL" {
				url = v.Value
			}
		}
	}
	if url == "" || !bytes.Contains([]byte(url), []byte("@zenith-db-")) || !bytes.Contains([]byte(url), []byte(":5432/")) {
		t.Errorf("DATABASE_URL = %q", url)
	}
}

func TestDatabaseCreateStandalone_UsesRuntime(t *testing.T) {
	dbRepo := memory.NewMemoryDatabaseRepository()
	rt := services.NewDBRuntimeService(dbRepo, okDBBackend{}, nil)
	h := handlers.NewDatabaseHandlerV2(nil, dbRepo, memory.NewMemoryAppRepository(), nil)
	h.SetRuntime(rt)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Post("/api/v1/databases", injectUserID("user-1"), h.CreateStandalone)

	req := httptest.NewRequest("POST", "/api/v1/databases", bytes.NewBufferString(`{"engine":"mysql","name":"shop"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var info dto.DatabaseInfo
	_ = json.NewDecoder(resp.Body).Decode(&info)
	rt.Wait(info.ID)
	got, err := dbRepo.GetDatabase(context.Background(), info.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != entities.DatabaseStatusReady || got.Host == "" {
		t.Errorf("got %+v", got)
	}
}

type recDBBackend struct {
	okDBBackend
	deleted []string
}

func (r *recDBBackend) Delete(_ context.Context, db *entities.UserDatabase, _ bool) error {
	r.deleted = append(r.deleted, db.ID)
	return nil
}

func TestDatabaseGetForApp_RejectsForeignDatabase(t *testing.T) {
	ctx := context.Background()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	a, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	rt := services.NewDBRuntimeService(dbRepo, okDBBackend{}, nil)
	other, err := rt.Create(ctx, "", "user-2", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "theirs"})
	if err != nil {
		t.Fatal(err)
	}
	rt.Wait(other.ID)
	h := handlers.NewDatabaseHandlerV2(nil, dbRepo, appRepo, nil)
	h.SetRuntime(rt)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Get("/api/v1/apps/:appId/databases/:dbId", injectUserID("user-1"), h.Get)
	app.Delete("/api/v1/apps/:appId/databases/:dbId", injectUserID("user-1"), h.Delete)

	pw, _ := rt.Password(ctx, other)
	for _, method := range []string{"GET", "DELETE"} {
		resp, err := app.Test(httptest.NewRequest(method, "/api/v1/apps/"+a.ID+"/databases/"+other.ID, nil), -1)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 404 {
			t.Fatalf("%s status = %d, want 404", method, resp.StatusCode)
		}
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		if pw != "" && bytes.Contains(buf.Bytes(), []byte(pw)) {
			t.Errorf("%s response leaked the password", method)
		}
	}
	if _, err := dbRepo.GetDatabase(ctx, other.ID); err != nil {
		t.Errorf("foreign database must survive: %v", err)
	}
}

func TestDatabaseDeleteForApp_UsesRuntime(t *testing.T) {
	ctx := context.Background()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	a, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	be := &recDBBackend{}
	rt := services.NewDBRuntimeService(dbRepo, be, nil)
	db, err := rt.Create(ctx, a.ID, "user-1", &dto.CreateDatabaseInput{Engine: entities.DatabaseEnginePostgres, Name: "main"})
	if err != nil {
		t.Fatal(err)
	}
	rt.Wait(db.ID)
	h := handlers.NewDatabaseHandlerV2(nil, dbRepo, appRepo, nil)
	h.SetRuntime(rt)
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Delete("/api/v1/apps/:appId/databases/:dbId", injectUserID("user-1"), h.Delete)

	resp, err := app.Test(httptest.NewRequest("DELETE", "/api/v1/apps/"+a.ID+"/databases/"+db.ID, nil), -1)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("delete: %v status=%v", err, resp)
	}
	if len(be.deleted) != 1 || be.deleted[0] != db.ID {
		t.Errorf("backend deletes = %v", be.deleted)
	}
	if _, err := dbRepo.GetDatabase(ctx, db.ID); err == nil {
		t.Error("record should be gone")
	}
}
