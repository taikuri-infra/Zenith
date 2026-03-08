package memory

import (
	"context"
	"testing"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
)

func TestCreateApp(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, err := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "my-api",
		RepoURL: "https://github.com/user/repo",
		Branch:  "main",
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if app.ID == "" {
		t.Error("Expected ID to be set")
	}
	if app.Name != "my-api" {
		t.Errorf("Expected name 'my-api', got '%s'", app.Name)
	}
	if app.Status != entities.AppStatusPending {
		t.Errorf("Expected status 'pending', got '%s'", app.Status)
	}
	if app.Framework != entities.FrameworkUnknown {
		t.Errorf("Expected framework 'unknown', got '%s'", app.Framework)
	}
	if app.Subdomain != "my-api" {
		t.Errorf("Expected subdomain 'my-api', got '%s'", app.Subdomain)
	}
	if app.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", app.Port)
	}
	if app.Branch != "main" {
		t.Errorf("Expected branch 'main', got '%s'", app.Branch)
	}
}

func TestCreateAppDefaultBranch(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, err := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:  "user-1",
		Name:    "web",
		RepoURL: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if app.Branch != "main" {
		t.Errorf("Expected default branch 'main', got '%s'", app.Branch)
	}
}

func TestCreateAppDuplicateName(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	_, err := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	_, err = repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo2",
	})
	if err == nil {
		t.Fatal("Expected error on duplicate name for same user")
	}
}

func TestCreateAppSameNameDifferentUser(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	_, err := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", ProjectID: "proj-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	_, err = repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-2", ProjectID: "proj-2", Name: "web", RepoURL: "https://github.com/user/repo",
	})
	if err != nil {
		t.Fatalf("Should allow same name in different project, got: %v", err)
	}
}

func TestCreateAppValidation(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	// Missing name
	_, err := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", RepoURL: "https://github.com/user/repo",
	})
	if err == nil {
		t.Error("Expected error for missing name")
	}

	// Missing repo URL
	_, err = repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web",
	})
	if err == nil {
		t.Error("Expected error for missing repo URL")
	}

	// Missing user ID
	_, err = repo.CreateApp(ctx, &dto.CreateAppInput{
		Name: "web", RepoURL: "https://github.com/user/repo",
	})
	if err == nil {
		t.Error("Expected error for missing user ID")
	}
}

func TestGetApp(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	created, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	app, err := repo.GetApp(ctx, created.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if app.Name != "web" {
		t.Errorf("Expected name 'web', got '%s'", app.Name)
	}
}

func TestGetAppNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	_, err := repo.GetApp(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestGetAppBySubdomain(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	_, _ = repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "my_app", RepoURL: "https://github.com/user/repo",
	})

	app, err := repo.GetAppBySubdomain(ctx, "my-app")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if app.Name != "my_app" {
		t.Errorf("Expected name 'my_app', got '%s'", app.Name)
	}
}

func TestGetAppBySubdomainNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	_, err := repo.GetAppBySubdomain(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent subdomain")
	}
}

func TestListAppsByUser(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	// Create 2 apps for user-1
	repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "api", RepoURL: "https://github.com/user/repo1",
	})
	repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo2",
	})

	// Create 1 app for user-2
	repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-2", Name: "admin", RepoURL: "https://github.com/user/repo3",
	})

	apps, err := repo.ListAppsByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(apps) != 2 {
		t.Errorf("Expected 2 apps for user-1, got %d", len(apps))
	}

	apps2, _ := repo.ListAppsByUser(ctx, "user-2")
	if len(apps2) != 1 {
		t.Errorf("Expected 1 app for user-2, got %d", len(apps2))
	}
}

func TestListAppsByUserEmpty(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	apps, err := repo.ListAppsByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(apps) != 0 {
		t.Errorf("Expected 0 apps, got %d", len(apps))
	}
}

func TestUpdateAppStatus(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	created, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	status := entities.AppStatusRunning
	updated, err := repo.UpdateApp(ctx, created.ID, &dto.UpdateAppInput{
		Status: &status,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if updated.Status != entities.AppStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", updated.Status)
	}
}

func TestUpdateAppFramework(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	created, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	fw := entities.FrameworkNextJS
	updated, err := repo.UpdateApp(ctx, created.ID, &dto.UpdateAppInput{
		Framework: &fw,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if updated.Framework != entities.FrameworkNextJS {
		t.Errorf("Expected framework 'nextjs', got '%s'", updated.Framework)
	}
}

func TestUpdateAppNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	status := entities.AppStatusRunning
	_, err := repo.UpdateApp(ctx, "nonexistent", &dto.UpdateAppInput{
		Status: &status,
	})
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestDeleteApp(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	created, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	err := repo.DeleteApp(ctx, created.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should not be found after delete
	_, err = repo.GetApp(ctx, created.ID)
	if err == nil {
		t.Error("Expected error after delete")
	}
}

func TestDeleteAppNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	err := repo.DeleteApp(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestDeleteAppCascadesDeployments(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})
	repo.CreateDeployment(ctx, app.ID, "abc123")

	repo.DeleteApp(ctx, app.ID)

	deployments, _ := repo.ListDeployments(ctx, app.ID, 10)
	if len(deployments) != 0 {
		t.Error("Expected deployments to be cascade-deleted")
	}
}

func TestDeleteAppCascadesEnvVars(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})
	repo.SetEnvVars(ctx, app.ID, map[string]string{"KEY": "val"})

	repo.DeleteApp(ctx, app.ID)

	vars, _ := repo.GetEnvVars(ctx, app.ID)
	if len(vars) != 0 {
		t.Error("Expected env vars to be cascade-deleted")
	}
}

func TestCountAppsByUser(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	count, _ := repo.CountAppsByUser(ctx, "user-1")
	if count != 0 {
		t.Errorf("Expected 0, got %d", count)
	}

	repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "api", RepoURL: "https://github.com/user/repo1",
	})
	repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo2",
	})

	count, _ = repo.CountAppsByUser(ctx, "user-1")
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

// --- Deployment Tests ---

func TestCreateDeployment(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	d, err := repo.CreateDeployment(ctx, app.ID, "abc123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if d.ID == "" {
		t.Error("Expected ID to be set")
	}
	if d.AppID != app.ID {
		t.Errorf("Expected app ID '%s', got '%s'", app.ID, d.AppID)
	}
	if d.GitSHA != "abc123" {
		t.Errorf("Expected git SHA 'abc123', got '%s'", d.GitSHA)
	}
	if d.Status != entities.DeployStatusPending {
		t.Errorf("Expected status 'pending', got '%s'", d.Status)
	}
}

func TestCreateDeploymentAppNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	_, err := repo.CreateDeployment(ctx, "nonexistent", "abc123")
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestListDeployments(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.CreateDeployment(ctx, app.ID, "sha1")
	repo.CreateDeployment(ctx, app.ID, "sha2")
	repo.CreateDeployment(ctx, app.ID, "sha3")

	deployments, err := repo.ListDeployments(ctx, app.ID, 10)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(deployments) != 3 {
		t.Errorf("Expected 3 deployments, got %d", len(deployments))
	}
}

func TestListDeploymentsWithLimit(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.CreateDeployment(ctx, app.ID, "sha1")
	repo.CreateDeployment(ctx, app.ID, "sha2")
	repo.CreateDeployment(ctx, app.ID, "sha3")

	deployments, _ := repo.ListDeployments(ctx, app.ID, 2)
	if len(deployments) != 2 {
		t.Errorf("Expected 2 deployments (limit), got %d", len(deployments))
	}
}

func TestUpdateDeploymentStatus(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})
	d, _ := repo.CreateDeployment(ctx, app.ID, "abc123")

	err := repo.UpdateDeploymentStatus(ctx, d.ID, entities.DeployStatusActive, "build log content", "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	updated, _ := repo.GetDeployment(ctx, d.ID)
	if updated.Status != entities.DeployStatusActive {
		t.Errorf("Expected status 'active', got '%s'", updated.Status)
	}
	if updated.BuildLog != "build log content" {
		t.Errorf("Expected build log set, got '%s'", updated.BuildLog)
	}
}

func TestUpdateDeploymentStatusFailed(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})
	d, _ := repo.CreateDeployment(ctx, app.ID, "abc123")

	err := repo.UpdateDeploymentStatus(ctx, d.ID, entities.DeployStatusFailed, "", "syntax error at main.go:15")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	updated, _ := repo.GetDeployment(ctx, d.ID)
	if updated.Status != entities.DeployStatusFailed {
		t.Errorf("Expected status 'failed', got '%s'", updated.Status)
	}
	if updated.Error != "syntax error at main.go:15" {
		t.Errorf("Expected error message set, got '%s'", updated.Error)
	}
}

func TestGetActiveDeployment(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})
	d, _ := repo.CreateDeployment(ctx, app.ID, "abc123")
	repo.UpdateDeploymentStatus(ctx, d.ID, entities.DeployStatusActive, "", "")

	active, err := repo.GetActiveDeployment(ctx, app.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if active.ID != d.ID {
		t.Errorf("Expected active deployment ID '%s', got '%s'", d.ID, active.ID)
	}
}

func TestGetActiveDeploymentNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	_, err := repo.GetActiveDeployment(ctx, app.ID)
	if err == nil {
		t.Error("Expected error when no active deployment")
	}
}

// --- Env Var Tests ---

func TestSetEnvVars(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	err := repo.SetEnvVars(ctx, app.ID, map[string]string{
		"DATABASE_URL": "postgres://...",
		"API_KEY":      "secret123",
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	vars, _ := repo.GetEnvVars(ctx, app.ID)
	if len(vars) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(vars))
	}
}

func TestSetEnvVarsAppNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	err := repo.SetEnvVars(ctx, "nonexistent", map[string]string{"KEY": "val"})
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}

func TestSetEnvVarsUpsert(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.SetEnvVars(ctx, app.ID, map[string]string{"KEY": "old_value"})
	repo.SetEnvVars(ctx, app.ID, map[string]string{"KEY": "new_value"})

	vars, _ := repo.GetEnvVars(ctx, app.ID)
	if len(vars) != 1 {
		t.Fatalf("Expected 1 env var (upsert), got %d", len(vars))
	}
	if vars[0].Value != "new_value" {
		t.Errorf("Expected value 'new_value', got '%s'", vars[0].Value)
	}
}

func TestGetEnvVarsEmpty(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	vars, err := repo.GetEnvVars(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("Expected 0 env vars, got %d", len(vars))
	}
}

func TestGetEnvVarsSorted(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.SetEnvVars(ctx, app.ID, map[string]string{
		"Z_VAR": "z",
		"A_VAR": "a",
		"M_VAR": "m",
	})

	vars, _ := repo.GetEnvVars(ctx, app.ID)
	if len(vars) != 3 {
		t.Fatalf("Expected 3 env vars, got %d", len(vars))
	}
	if vars[0].Key != "A_VAR" {
		t.Errorf("Expected first var key 'A_VAR', got '%s'", vars[0].Key)
	}
	if vars[2].Key != "Z_VAR" {
		t.Errorf("Expected last var key 'Z_VAR', got '%s'", vars[2].Key)
	}
}

func TestDeleteEnvVar(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.SetEnvVars(ctx, app.ID, map[string]string{"KEY1": "val1", "KEY2": "val2"})

	err := repo.DeleteEnvVar(ctx, app.ID, "KEY1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	vars, _ := repo.GetEnvVars(ctx, app.ID)
	if len(vars) != 1 {
		t.Errorf("Expected 1 env var after delete, got %d", len(vars))
	}
	if vars[0].Key != "KEY2" {
		t.Errorf("Expected remaining var 'KEY2', got '%s'", vars[0].Key)
	}
}

func TestDeleteEnvVarNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	err := repo.DeleteEnvVar(ctx, app.ID, "NONEXISTENT")
	if err == nil {
		t.Error("Expected error for nonexistent env var")
	}
}

// --- Secrets Tests ---

func TestSetSecret(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	err := repo.SetSecret(ctx, app.ID, "API_KEY", []byte("encrypted-data"))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestGetSecrets(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.SetSecret(ctx, app.ID, "API_KEY", []byte("encrypted-api-key"))
	repo.SetSecret(ctx, app.ID, "DB_PASSWORD", []byte("encrypted-db-pass"))

	secrets, err := repo.GetSecrets(ctx, app.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(secrets) != 2 {
		t.Errorf("Expected 2 secrets, got %d", len(secrets))
	}
	if secrets[0].Key != "API_KEY" {
		t.Errorf("Expected first secret key 'API_KEY', got '%s'", secrets[0].Key)
	}
	if secrets[1].Key != "DB_PASSWORD" {
		t.Errorf("Expected second secret key 'DB_PASSWORD', got '%s'", secrets[1].Key)
	}
}

func TestGetSecretValue(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	expectedValue := []byte("encrypted-secret-value")
	repo.SetSecret(ctx, app.ID, "SECRET_KEY", expectedValue)

	value, err := repo.GetSecretValue(ctx, app.ID, "SECRET_KEY")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if string(value) != string(expectedValue) {
		t.Errorf("Expected value '%s', got '%s'", expectedValue, value)
	}
}

func TestDeleteSecret(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.SetSecret(ctx, app.ID, "SECRET_KEY", []byte("encrypted-data"))

	err := repo.DeleteSecret(ctx, app.ID, "SECRET_KEY")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	secrets, _ := repo.GetSecrets(ctx, app.ID)
	if len(secrets) != 0 {
		t.Errorf("Expected 0 secrets after delete, got %d", len(secrets))
	}
}

func TestDeleteSecretNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	err := repo.DeleteSecret(ctx, app.ID, "NONEXISTENT")
	if err == nil {
		t.Error("Expected error for nonexistent secret")
	}
}

func TestSetSecretOverwrite(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.SetSecret(ctx, app.ID, "API_KEY", []byte("old-encrypted-value"))
	repo.SetSecret(ctx, app.ID, "API_KEY", []byte("new-encrypted-value"))

	secrets, _ := repo.GetSecrets(ctx, app.ID)
	if len(secrets) != 1 {
		t.Errorf("Expected 1 secret (overwrite), got %d", len(secrets))
	}

	value, _ := repo.GetSecretValue(ctx, app.ID, "API_KEY")
	if string(value) != "new-encrypted-value" {
		t.Errorf("Expected new value 'new-encrypted-value', got '%s'", value)
	}
}

func TestGetSecretsEmpty(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	secrets, err := repo.GetSecrets(ctx, app.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("Expected 0 secrets for app with none, got %d", len(secrets))
	}
}

// --- Releases Tests ---

func TestCreateRelease(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	release, err := repo.CreateRelease(ctx, app.ID, &dto.CreateReleaseInput{
		Image:   "registry/app:v1",
		GitSHA:  "abc123",
		Branch:  "main",
		Message: "initial",
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if release.ID == "" {
		t.Error("Expected ID to be set")
	}
	if release.AppID != app.ID {
		t.Errorf("Expected app ID '%s', got '%s'", app.ID, release.AppID)
	}
	if release.Image != "registry/app:v1" {
		t.Errorf("Expected image 'registry/app:v1', got '%s'", release.Image)
	}
	if release.GitSHA != "abc123" {
		t.Errorf("Expected git SHA 'abc123', got '%s'", release.GitSHA)
	}
	if release.Branch != "main" {
		t.Errorf("Expected branch 'main', got '%s'", release.Branch)
	}
	if release.Message != "initial" {
		t.Errorf("Expected message 'initial', got '%s'", release.Message)
	}
}

func TestListReleases(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.CreateRelease(ctx, app.ID, &dto.CreateReleaseInput{
		Image:  "registry/app:v1",
		GitSHA: "sha1",
		Branch: "main",
	})
	repo.CreateRelease(ctx, app.ID, &dto.CreateReleaseInput{
		Image:  "registry/app:v2",
		GitSHA: "sha2",
		Branch: "main",
	})
	repo.CreateRelease(ctx, app.ID, &dto.CreateReleaseInput{
		Image:  "registry/app:v3",
		GitSHA: "sha3",
		Branch: "main",
	})

	releases, err := repo.ListReleases(ctx, app.ID, 0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(releases) != 3 {
		t.Errorf("Expected 3 releases, got %d", len(releases))
	}
}

func TestListReleasesLimit(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	repo.CreateRelease(ctx, app.ID, &dto.CreateReleaseInput{
		Image:  "registry/app:v1",
		GitSHA: "sha1",
	})
	repo.CreateRelease(ctx, app.ID, &dto.CreateReleaseInput{
		Image:  "registry/app:v2",
		GitSHA: "sha2",
	})
	repo.CreateRelease(ctx, app.ID, &dto.CreateReleaseInput{
		Image:  "registry/app:v3",
		GitSHA: "sha3",
	})

	releases, err := repo.ListReleases(ctx, app.ID, 2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(releases) != 2 {
		t.Errorf("Expected 2 releases (limit), got %d", len(releases))
	}
}

func TestGetRelease(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	app, _ := repo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: "user-1", Name: "web", RepoURL: "https://github.com/user/repo",
	})

	created, _ := repo.CreateRelease(ctx, app.ID, &dto.CreateReleaseInput{
		Image:   "registry/app:v1",
		GitSHA:  "abc123",
		Branch:  "main",
		Message: "test release",
	})

	release, err := repo.GetRelease(ctx, created.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if release.ID != created.ID {
		t.Errorf("Expected ID '%s', got '%s'", created.ID, release.ID)
	}
	if release.Image != "registry/app:v1" {
		t.Errorf("Expected image 'registry/app:v1', got '%s'", release.Image)
	}
}

func TestGetReleaseNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	_, err := repo.GetRelease(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent release")
	}
}

func TestCreateReleaseAppNotFound(t *testing.T) {
	repo := NewMemoryAppRepository()
	ctx := context.Background()

	_, err := repo.CreateRelease(ctx, "nonexistent", &dto.CreateReleaseInput{
		Image:  "registry/app:v1",
		GitSHA: "abc123",
	})
	if err == nil {
		t.Error("Expected error for nonexistent app")
	}
}
