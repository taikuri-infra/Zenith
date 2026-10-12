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
	"github.com/dotechhq/zenith/services/api/internal/services/deploy"
	"github.com/gofiber/fiber/v2"
)

func TestLogHistory_FallsBackToBuildLog(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewMemoryAppRepository()
	a, _ := repo.CreateApp(ctx, &dto.CreateAppInput{UserID: "user-1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	d, _ := repo.CreateDeployment(ctx, a.ID, "nginx:1")
	_ = repo.UpdateDeploymentStatus(ctx, d.ID, entities.DeployStatusActive,
		"2026-10-12T08:00:00Z\tinfo\tDeploying web\n2026-10-12T08:00:05Z\tdeploy\t✓ web is live\n", "")

	h := handlers.NewLogHandler(repo, deploy.NewLogHub(10)) // empty hub: as after an API restart
	app := fiber.New(fiber.Config{ErrorHandler: handlers.ErrorHandler})
	app.Get("/api/v1/apps/:appId/deployments/:did/logs/history", injectUserID("user-1"), h.GetLogs)

	resp, _ := app.Test(httptest.NewRequest("GET", "/api/v1/apps/"+a.ID+"/deployments/"+d.ID+"/logs/history", nil), -1)
	var out struct {
		Items []struct{ Level, Message string } `json:"items"`
		Total int                               `json:"total"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Total != 2 || out.Items[1].Message != "✓ web is live" || out.Items[1].Level != "deploy" {
		t.Errorf("got %+v", out)
	}
}
