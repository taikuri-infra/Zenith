//go:build integration
// +build integration

package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/postgres/migrations"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testDB holds a shared Postgres container and pool for all integration tests.
type testDB struct {
	pool      *pgxpool.Pool
	container testcontainers.Container
	dsn       string
}

// setupTestDB starts a Postgres container, runs migrations, and returns a pool.
// Call teardown() when done.
func setupTestDB(t *testing.T) *testDB {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("zenith_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Run all migrations
	if err := RunMigrations(connStr, migrations.FS); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	return &testDB{pool: pool, container: container, dsn: connStr}
}

func (db *testDB) teardown(t *testing.T) {
	t.Helper()
	db.pool.Close()
	if err := db.container.Terminate(context.Background()); err != nil {
		t.Logf("warning: failed to terminate container: %v", err)
	}
}

// cleanTable truncates a table between tests.
func (db *testDB) cleanTable(t *testing.T, tables ...string) {
	t.Helper()
	for _, table := range tables {
		_, err := db.pool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Fatalf("failed to truncate %s: %v", table, err)
		}
	}
}

// --- Migration Tests ---

func TestMigrationsApply(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()

	// Verify core tables exist
	tables := []string{"users", "projects", "apps", "deployments", "databases", "storage_buckets",
		"api_keys", "gateways", "gateway_routes", "team_members", "support_tickets",
		"subscriptions", "environments", "deploy_tokens", "deploy_hooks", "app_env_vars"}

	for _, table := range tables {
		var exists bool
		err := db.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`,
			table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("query failed for table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("expected table %s to exist after migrations", table)
		}
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	// Running migrations again should not error (ErrNoChange)
	if err := RunMigrations(db.dsn, migrations.FS); err != nil {
		t.Fatalf("second migration run should be idempotent: %v", err)
	}
}

// --- Project Repository Tests ---

func TestProjectCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	repo := NewPostgresProjectRepository(db.pool)

	// Create user first (projects require user_id FK)
	userID := createTestUser(t, db, "project-test@test.com")

	// Create
	proj, err := repo.CreateProject(ctx, userID, "Test Project", "test-project", "A test project")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if proj.ID == "" || proj.Name != "Test Project" || proj.Slug != "test-project" {
		t.Errorf("unexpected project: %+v", proj)
	}
	if proj.Status != entities.ProjectStatusDraft {
		t.Errorf("expected draft status, got %s", proj.Status)
	}

	// Get
	got, err := repo.GetProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.Name != "Test Project" {
		t.Errorf("GetProject name: got %s, want Test Project", got.Name)
	}

	// List
	list, err := repo.ListProjectsByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListProjectsByUser: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListProjectsByUser: got %d projects, want 1", len(list))
	}

	// Duplicate slug
	_, err = repo.CreateProject(ctx, userID, "Another", "test-project", "")
	if err == nil {
		t.Error("expected error for duplicate slug")
	}

	// Delete
	err = repo.DeleteProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	_, err = repo.GetProject(ctx, proj.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

// --- App Repository Tests ---

func TestAppCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	appRepo := NewPostgresAppRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)

	userID := createTestUser(t, db, "app-test@test.com")
	proj, _ := projRepo.CreateProject(ctx, userID, "App Project", "app-project", "")

	// Create app with image deploy
	app, err := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:          userID,
		ProjectID:       proj.ID,
		Name:            "my-app",
		DeploySource:    entities.DeploySourceImage,
		ImageURL:        "nginx:latest",
		Port:            80,
		HealthCheckPath: "/healthz",
	})
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}
	if app.ID == "" || app.Name != "my-app" || app.Subdomain == "" {
		t.Errorf("unexpected app: %+v", app)
	}
	if app.HealthCheckPath != "/healthz" {
		t.Errorf("health_check_path: got %s, want /healthz", app.HealthCheckPath)
	}
	if app.Replicas != 1 {
		t.Errorf("replicas: got %d, want 1", app.Replicas)
	}

	// Get
	got, err := appRepo.GetApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetApp: %v", err)
	}
	if got.Name != "my-app" || got.ImageURL != "nginx:latest" {
		t.Errorf("GetApp mismatch: %+v", got)
	}

	// List by project
	apps, err := appRepo.ListAppsByProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("ListAppsByProject: %v", err)
	}
	if len(apps) != 1 {
		t.Errorf("ListAppsByProject: got %d, want 1", len(apps))
	}

	// List by user
	userApps, err := appRepo.ListAppsByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListAppsByUser: %v", err)
	}
	if len(userApps) != 1 {
		t.Errorf("ListAppsByUser: got %d, want 1", len(userApps))
	}

	// Update
	newReplicas := 3
	updated, err := appRepo.UpdateApp(ctx, app.ID, &dto.UpdateAppInput{
		Replicas: &newReplicas,
	})
	if err != nil {
		t.Fatalf("UpdateApp: %v", err)
	}
	if updated.Replicas != 3 {
		t.Errorf("UpdateApp replicas: got %d, want 3", updated.Replicas)
	}

	// Count
	count, err := appRepo.CountAppsByUser(ctx, userID)
	if err != nil {
		t.Fatalf("CountAppsByUser: %v", err)
	}
	if count != 1 {
		t.Errorf("CountAppsByUser: got %d, want 1", count)
	}
}

func TestAppSoftDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	appRepo := NewPostgresAppRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)

	userID := createTestUser(t, db, "softdel-test@test.com")
	proj, _ := projRepo.CreateProject(ctx, userID, "SD Project", "sd-project", "")

	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:       userID,
		ProjectID:    proj.ID,
		Name:         "deleteme",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "nginx:latest",
	})

	// Soft delete
	err := appRepo.SoftDeleteApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("SoftDeleteApp: %v", err)
	}

	// GetApp should fail (filters deleted_at IS NULL)
	_, err = appRepo.GetApp(ctx, app.ID)
	if err == nil {
		t.Error("GetApp should fail after soft delete")
	}

	// ListAppsByProject should exclude it
	apps, _ := appRepo.ListAppsByProject(ctx, proj.ID)
	if len(apps) != 0 {
		t.Errorf("ListAppsByProject should exclude soft-deleted, got %d", len(apps))
	}

	// ListDeletedAppsByUser should include it
	deleted, err := appRepo.ListDeletedAppsByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListDeletedAppsByUser: %v", err)
	}
	if len(deleted) != 1 {
		t.Errorf("ListDeletedAppsByUser: got %d, want 1", len(deleted))
	}
	if deleted[0].DeletedAt == nil {
		t.Error("deleted_at should be set")
	}

	// Restore
	restored, err := appRepo.RestoreApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("RestoreApp: %v", err)
	}
	if restored.DeletedAt != nil {
		t.Error("deleted_at should be nil after restore")
	}

	// GetApp should work again
	_, err = appRepo.GetApp(ctx, app.ID)
	if err != nil {
		t.Errorf("GetApp should work after restore: %v", err)
	}

	// Hard delete
	err = appRepo.DeleteApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("DeleteApp (hard): %v", err)
	}
	_, err = appRepo.GetApp(ctx, app.ID)
	if err == nil {
		t.Error("GetApp should fail after hard delete")
	}
}

func TestAppSubdomainUniqueness(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	appRepo := NewPostgresAppRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)

	userID := createTestUser(t, db, "subdomain-test@test.com")
	proj, _ := projRepo.CreateProject(ctx, userID, "Sub Project", "sub-project", "")

	// Create first app
	_, err := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:       userID,
		ProjectID:    proj.ID,
		Name:         "unique-app",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "nginx:latest",
	})
	if err != nil {
		t.Fatalf("first CreateApp: %v", err)
	}

	// Same name in same project should fail
	_, err = appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:       userID,
		ProjectID:    proj.ID,
		Name:         "unique-app",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "nginx:latest",
	})
	if err == nil {
		t.Error("expected duplicate name error")
	}
}

func TestAppReservedNames(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	appRepo := NewPostgresAppRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)

	userID := createTestUser(t, db, "reserved-test@test.com")
	proj, _ := projRepo.CreateProject(ctx, userID, "Res Project", "res-project", "")

	reserved := []string{"traefik", "argocd", "grafana", "keycloak", "admin", "api"}
	for _, name := range reserved {
		_, err := appRepo.CreateApp(ctx, &dto.CreateAppInput{
			UserID:       userID,
			ProjectID:    proj.ID,
			Name:         name,
			DeploySource: entities.DeploySourceImage,
			ImageURL:     "nginx:latest",
		})
		if err == nil {
			t.Errorf("expected error for reserved name %q", name)
		}
	}
}

// --- Deployment Tests ---

func TestDeploymentCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	appRepo := NewPostgresAppRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)

	userID := createTestUser(t, db, "deploy-test@test.com")
	proj, _ := projRepo.CreateProject(ctx, userID, "Deploy Project", "deploy-project", "")
	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:       userID,
		ProjectID:    proj.ID,
		Name:         "deployable",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "nginx:latest",
	})

	// Create deployment
	dep, err := appRepo.CreateDeployment(ctx, app.ID, "nginx:1.25")
	if err != nil {
		t.Fatalf("CreateDeployment: %v", err)
	}
	if dep.ID == "" || dep.AppID != app.ID || dep.ImageTag != "nginx:1.25" {
		t.Errorf("unexpected deployment: %+v", dep)
	}
	if dep.Status != entities.DeployStatusPending {
		t.Errorf("expected pending status, got %s", dep.Status)
	}

	// Get deployment
	got, err := appRepo.GetDeployment(ctx, dep.ID)
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	if got.ImageTag != "nginx:1.25" {
		t.Errorf("GetDeployment image: got %s", got.ImageTag)
	}

	// Update status
	err = appRepo.UpdateDeploymentStatus(ctx, dep.ID, entities.DeployStatusActive, "", "")
	if err != nil {
		t.Fatalf("UpdateDeploymentStatus: %v", err)
	}
	got, _ = appRepo.GetDeployment(ctx, dep.ID)
	if got.Status != entities.DeployStatusActive {
		t.Errorf("expected active, got %s", got.Status)
	}

	// List deployments
	deps, err := appRepo.ListDeployments(ctx, app.ID, 100)
	if err != nil {
		t.Fatalf("ListDeployments: %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("ListDeployments: got %d, want 1", len(deps))
	}

	// Create second deployment
	dep2, _ := appRepo.CreateDeployment(ctx, app.ID, "nginx:1.26")
	deps, _ = appRepo.ListDeployments(ctx, app.ID, 100)
	if len(deps) != 2 {
		t.Errorf("ListDeployments after second: got %d, want 2", len(deps))
	}
	_ = dep2
}

// --- Env Var Tests ---

func TestEnvVarCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	appRepo := NewPostgresAppRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)

	userID := createTestUser(t, db, "envvar-test@test.com")
	proj, _ := projRepo.CreateProject(ctx, userID, "Env Project", "env-project", "")
	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:       userID,
		ProjectID:    proj.ID,
		Name:         "env-app",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "nginx:latest",
	})

	// Set env vars
	err := appRepo.SetEnvVars(ctx, app.ID, map[string]string{
		"DB_HOST": "localhost",
		"DB_PORT": "5432",
	})
	if err != nil {
		t.Fatalf("SetEnvVars: %v", err)
	}

	// Get env vars
	vars, err := appRepo.GetEnvVars(ctx, app.ID)
	if err != nil {
		t.Fatalf("GetEnvVars: %v", err)
	}
	if len(vars) != 2 {
		t.Errorf("GetEnvVars: got %d, want 2", len(vars))
	}

	found := map[string]string{}
	for _, v := range vars {
		found[v.Key] = v.Value
	}
	if found["DB_HOST"] != "localhost" || found["DB_PORT"] != "5432" {
		t.Errorf("env var values mismatch: %v", found)
	}

	// Overwrite
	err = appRepo.SetEnvVars(ctx, app.ID, map[string]string{
		"DB_HOST": "prod-db.internal",
	})
	if err != nil {
		t.Fatalf("SetEnvVars overwrite: %v", err)
	}

	vars, _ = appRepo.GetEnvVars(ctx, app.ID)
	found = map[string]string{}
	for _, v := range vars {
		found[v.Key] = v.Value
	}
	if found["DB_HOST"] != "prod-db.internal" {
		t.Errorf("overwrite failed: got %s", found["DB_HOST"])
	}

	// Delete
	err = appRepo.DeleteEnvVar(ctx, app.ID, "DB_HOST")
	if err != nil {
		t.Fatalf("DeleteEnvVar: %v", err)
	}
	vars, _ = appRepo.GetEnvVars(ctx, app.ID)
	for _, v := range vars {
		if v.Key == "DB_HOST" {
			t.Error("DB_HOST should be deleted")
		}
	}
}

// --- Deploy Hook Tests ---

func TestDeployHookCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	appRepo := NewPostgresAppRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)
	hookRepo := NewPostgresDeployHookRepository(db.pool)

	userID := createTestUser(t, db, "hook-test@test.com")
	proj, _ := projRepo.CreateProject(ctx, userID, "Hook Project", "hook-project", "")
	app, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID:       userID,
		ProjectID:    proj.ID,
		Name:         "hook-app",
		DeploySource: entities.DeploySourceImage,
		ImageURL:     "nginx:latest",
	})

	// Create hook
	hook, err := hookRepo.CreateHook(ctx, &entities.DeployHook{
		AppID:  app.ID,
		Name:   "notify",
		Type:   "http",
		URL:    "https://example.com/webhook",
		Active: true,
	})
	if err != nil {
		t.Fatalf("CreateHook: %v", err)
	}
	if hook.ID == "" || hook.Name != "notify" {
		t.Errorf("unexpected hook: %+v", hook)
	}

	// List hooks
	hooks, err := hookRepo.ListHooksByApp(ctx, app.ID)
	if err != nil {
		t.Fatalf("ListHooksByApp: %v", err)
	}
	if len(hooks) != 1 {
		t.Errorf("ListHooksByApp: got %d, want 1", len(hooks))
	}

	// Update
	newName := "updated-notify"
	updated, err := hookRepo.UpdateHook(ctx, hook.ID, &newName, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("UpdateHook: %v", err)
	}
	if updated.Name != "updated-notify" {
		t.Errorf("UpdateHook name: got %s", updated.Name)
	}

	// Delete
	err = hookRepo.DeleteHook(ctx, hook.ID)
	if err != nil {
		t.Fatalf("DeleteHook: %v", err)
	}
	hooks, _ = hookRepo.ListHooksByApp(ctx, app.ID)
	if len(hooks) != 0 {
		t.Errorf("after delete: got %d hooks, want 0", len(hooks))
	}
}

// --- Deploy Token Tests ---

func TestDeployTokenCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	tokenRepo := NewPostgresDeployTokenRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)

	userID := createTestUser(t, db, "token-test@test.com")
	proj, _ := projRepo.CreateProject(ctx, userID, "Token Project", "token-project", "")

	// Create token
	expiresAt := time.Now().Add(365 * 24 * time.Hour)
	token, err := tokenRepo.CreateDeployToken(ctx, userID, proj.ID, "ci-deploy", []string{"deploy", "status"}, &expiresAt)
	if err != nil {
		t.Fatalf("CreateDeployToken: %v", err)
	}
	if token.ID == "" || token.Name != "ci-deploy" {
		t.Errorf("unexpected token: %+v", token)
	}

	// List
	tokens, err := tokenRepo.ListDeployTokensByProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("ListDeployTokensByProject: %v", err)
	}
	if len(tokens) != 1 {
		t.Errorf("ListDeployTokensByProject: got %d, want 1", len(tokens))
	}

	// Revoke
	err = tokenRepo.RevokeDeployToken(ctx, token.ID)
	if err != nil {
		t.Fatalf("RevokeDeployToken: %v", err)
	}

	// Revoked token should not appear in list
	tokens, _ = tokenRepo.ListDeployTokensByProject(ctx, proj.ID)
	if len(tokens) != 0 {
		t.Errorf("revoked token should not appear in list, got %d", len(tokens))
	}
}

// --- Environment Tests ---

func TestEnvironmentCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	envRepo := NewPostgresEnvironmentRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)

	userID := createTestUser(t, db, "env-test@test.com")
	proj, _ := projRepo.CreateProject(ctx, userID, "Env Project2", "env-project2", "")

	// Create production environment
	prodEnv := &entities.Environment{
		ID:        "env-prod-" + proj.ID[:8],
		ProjectID: proj.ID,
		Name:      entities.EnvironmentProduction,
		Slug:      "production",
		Status:    entities.EnvironmentStatusActive,
		IsDefault: true,
	}
	err := envRepo.CreateEnvironment(ctx, prodEnv)
	if err != nil {
		t.Fatalf("CreateEnvironment: %v", err)
	}

	// Create staging environment
	stagingEnv := &entities.Environment{
		ID:        "env-staging-" + proj.ID[:8],
		ProjectID: proj.ID,
		Name:      entities.EnvironmentStaging,
		Slug:      "staging",
		Status:    entities.EnvironmentStatusActive,
	}
	err = envRepo.CreateEnvironment(ctx, stagingEnv)
	if err != nil {
		t.Fatalf("CreateEnvironment staging: %v", err)
	}

	// Get by name
	got, err := envRepo.GetEnvironmentByName(ctx, proj.ID, entities.EnvironmentProduction)
	if err != nil {
		t.Fatalf("GetEnvironmentByName: %v", err)
	}
	if got.Name != entities.EnvironmentProduction {
		t.Errorf("expected production, got %s", got.Name)
	}

	// List
	envs, err := envRepo.ListEnvironmentsByProject(ctx, proj.ID)
	if err != nil {
		t.Fatalf("ListEnvironmentsByProject: %v", err)
	}
	if len(envs) != 2 {
		t.Errorf("ListEnvironmentsByProject: got %d, want 2", len(envs))
	}

	// Delete
	err = envRepo.DeleteEnvironment(ctx, stagingEnv.ID)
	if err != nil {
		t.Fatalf("DeleteEnvironment: %v", err)
	}
	envs, _ = envRepo.ListEnvironmentsByProject(ctx, proj.ID)
	if len(envs) != 1 {
		t.Errorf("after delete: got %d, want 1", len(envs))
	}
}

// --- Cross-User Isolation Tests ---

func TestCrossUserIsolation(t *testing.T) {
	db := setupTestDB(t)
	defer db.teardown(t)

	ctx := context.Background()
	appRepo := NewPostgresAppRepository(db.pool)
	projRepo := NewPostgresProjectRepository(db.pool)

	userA := createTestUser(t, db, "user-a@test.com")
	userB := createTestUser(t, db, "user-b@test.com")

	projA, _ := projRepo.CreateProject(ctx, userA, "A Project", "a-project", "")
	projB, _ := projRepo.CreateProject(ctx, userB, "B Project", "b-project", "")

	appA, _ := appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: userA, ProjectID: projA.ID, Name: "app-a",
		DeploySource: entities.DeploySourceImage, ImageURL: "nginx:latest",
	})
	_, _ = appRepo.CreateApp(ctx, &dto.CreateAppInput{
		UserID: userB, ProjectID: projB.ID, Name: "app-b",
		DeploySource: entities.DeploySourceImage, ImageURL: "nginx:latest",
	})

	// User A should only see their own apps
	appsA, _ := appRepo.ListAppsByUser(ctx, userA)
	if len(appsA) != 1 || appsA[0].Name != "app-a" {
		t.Errorf("user A should only see app-a, got %v", appsA)
	}

	// User B should only see their own apps
	appsB, _ := appRepo.ListAppsByUser(ctx, userB)
	if len(appsB) != 1 || appsB[0].Name != "app-b" {
		t.Errorf("user B should only see app-b, got %v", appsB)
	}

	// User A's projects should not include B's
	projsA, _ := projRepo.ListProjectsByUser(ctx, userA)
	if len(projsA) != 1 {
		t.Errorf("user A should have 1 project, got %d", len(projsA))
	}

	// User B should not be able to access User A's app by ID
	gotApp, err := appRepo.GetApp(ctx, appA.ID)
	if err != nil {
		t.Fatalf("GetApp should succeed (no ownership filter at repo level): %v", err)
	}
	// The repo returns the app — ownership check happens in middleware, not repo
	if gotApp.UserID != userA {
		t.Errorf("app belongs to userA but got userID=%s", gotApp.UserID)
	}
}

// --- Helper ---

func createTestUser(t *testing.T, db *testDB, email string) string {
	t.Helper()
	id := fmt.Sprintf("user-%d", time.Now().UnixNano())
	_, err := db.pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())`,
		id, email, "hashed", "user",
	)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return id
}
