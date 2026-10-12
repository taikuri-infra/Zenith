package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/handlers"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

type attachRedeployer struct {
	mu     sync.Mutex
	images []string
	envSet bool // DATABASE_URL was present when DeployApp ran
	repo   *memory.MemoryAppRepository
}

func (r *attachRedeployer) DeployApp(ctx context.Context, app *entities.App, image string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	vars, _ := r.repo.GetEnvVars(ctx, app.ID)
	for _, v := range vars {
		if v.Key == "DATABASE_URL" {
			r.envSet = true
		}
	}
	r.images = append(r.images, image)
	return nil
}

func (r *attachRedeployer) seen() ([]string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.images...), r.envSet
}

func attachDB(t *testing.T, deployed bool) *attachRedeployer {
	t.Helper()
	ctx := context.Background()
	appRepo := memory.NewMemoryAppRepository()
	dbRepo := memory.NewMemoryDatabaseRepository()
	a, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	if deployed {
		d, _ := appRepo.CreateDeployment(ctx, a.ID, "")
		_ = appRepo.SetDeploymentImage(ctx, d.ID, "docker.io/acme/web:7")
		_ = appRepo.UpdateDeploymentStatus(ctx, d.ID, entities.DeployStatusActive, "", "")
	}
	rt := services.NewDBRuntimeService(dbRepo, okDBBackend{}, nil)
	rec := &attachRedeployer{repo: appRepo}
	h := handlers.NewDatabaseHandlerV2(nil, dbRepo, appRepo, nil)
	h.SetRuntime(rt)
	h.SetRedeployer(rec)
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
	return rec
}

func TestDatabaseAttach_RedeploysCurrentImage(t *testing.T) {
	rec := attachDB(t, true)
	var images []string
	var envSet bool
	for deadline := time.Now().Add(2 * time.Second); len(images) == 0 && time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		images, envSet = rec.seen()
	}
	if len(images) != 1 || images[0] != "docker.io/acme/web:7" {
		t.Fatalf("redeployed images = %v, want [docker.io/acme/web:7]", images)
	}
	if !envSet {
		t.Error("redeploy ran before DATABASE_URL was injected")
	}
}

func TestDatabaseAttach_NoImageSkipsRedeploy(t *testing.T) {
	rec := attachDB(t, false)
	time.Sleep(100 * time.Millisecond)
	if images, _ := rec.seen(); len(images) != 0 {
		t.Fatalf("app never deployed: want no redeploy, got %v", images)
	}
}
