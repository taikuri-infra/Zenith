package deploy

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestResolveEnvVars_Precedence(t *testing.T) {
	ctx := context.Background()
	appRepo := memory.NewMemoryAppRepository()
	envRepo := memory.NewMemoryEnvVarRepository()
	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{UserID: "u1", Name: "web", ImageURL: "nginx:1", DeploySource: entities.DeploySourceImage})
	app.EnvironmentID = "env-prod"

	// legacy table: compose import / db attach
	_ = appRepo.SetEnvVars(ctx, app.ID, map[string]string{"POSTGRES_PASSWORD": "audit", "SHARED": "legacy", "ONLY_LEGACY": "1"})
	// v2 app-wide (UI Environment tab without env scope)
	_ = envRepo.SetEnvVar(ctx, &entities.AppEnvVar{ID: "v1", AppID: app.ID, Key: "SHARED", Value: "app-wide"})
	_ = envRepo.SetEnvVar(ctx, &entities.AppEnvVar{ID: "v2", AppID: app.ID, Key: "AUDIT", Value: "1"})
	// v2 env-scoped
	_ = envRepo.SetEnvVar(ctx, &entities.AppEnvVar{ID: "v3", AppID: app.ID, EnvironmentID: "env-prod", Key: "AUDIT", Value: "2"})

	d := &DockerDeployer{appRepo: appRepo, envVarRepo: envRepo}
	vars, err := d.resolveEnvVars(ctx, app)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, v := range vars {
		got[v.Key] = v.Value
	}
	want := map[string]string{"POSTGRES_PASSWORD": "audit", "ONLY_LEGACY": "1", "SHARED": "app-wide", "AUDIT": "2"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %d vars, want %d: %v", len(got), len(want), got)
	}
}
