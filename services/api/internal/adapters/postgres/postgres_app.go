package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// dnsLabelRegex matches only valid DNS label characters (lowercase alphanumeric + hyphens).
var dnsLabelRegex = regexp.MustCompile(`[^a-z0-9-]`)

// reservedNames contains names that conflict with infrastructure services.
var reservedNames = map[string]bool{
	"apisix-gateway-bridge": true,
	"apisix-gateway-proxy":  true,
	"apisix":                true,
	"traefik":               true,
	"keycloak":              true,
	"harbor":                true,
	"argocd":                true,
	"grafana":               true,
	"prometheus":            true,
	"loki":                  true,
	"keda":                  true,
	"cert-manager":          true,
	"cold-start":            true,
	"cold-start-errors":     true,
	"external-dns":          true,
	"hubble":                true,
	"cilium":                true,
	"cnpg":                  true,
	"nats":                  true,
	"redis":                 true,
	"admin":                 true,
	"api":                   true,
	"www":                   true,
	"mail":                  true,
	"ftp":                   true,
	"localhost":             true,
}

// sanitizeSlug generates a DNS-safe slug from an app name.
// Only allows [a-z0-9-], max 48 chars (leaving room for 5-char suffix "-xxxx").
func sanitizeSlug(name string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = dnsLabelRegex.ReplaceAllString(s, "")
	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	if s == "" {
		return "", fmt.Errorf("app name '%s' produces an empty slug after sanitization", name)
	}
	// Enforce max length (DNS label max = 63, minus suffix "-xxxx" = 58, leave some margin)
	if len(s) > 48 {
		s = s[:48]
		s = strings.TrimRight(s, "-")
	}
	return s, nil
}

// PostgresAppRepository is a PostgreSQL-backed AppRepository.
type PostgresAppRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresAppRepository creates a new PostgreSQL AppRepository.
func NewPostgresAppRepository(pool *pgxpool.Pool) *PostgresAppRepository {
	return &PostgresAppRepository{pool: pool}
}

// --- Apps ---

func (r *PostgresAppRepository) CreateApp(ctx context.Context, input *dto.CreateAppInput) (*entities.App, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("app name is required")
	}
	if input.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	deploySource := input.DeploySource
	if deploySource == "" {
		deploySource = entities.DeploySourceGit
	}

	if deploySource == entities.DeploySourceGit && input.RepoURL == "" {
		return nil, fmt.Errorf("repo_url is required for git deploys")
	}
	if deploySource == entities.DeploySourceImage && input.ImageURL == "" {
		return nil, fmt.Errorf("image_url is required for image deploys")
	}

	branch := input.Branch
	if branch == "" && deploySource == entities.DeploySourceGit {
		branch = "main"
	}

	port := input.Port
	if port == 0 {
		port = 8080 // handler should resolve well-known ports before reaching here
	}

	// Generate subdomain: DNS-safe slug + 4-char hex suffix (deterministic, no collisions)
	slug, err := sanitizeSlug(input.Name)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(input.ProjectID + input.Name))
	suffix := hex.EncodeToString(hash[:2]) // 4 hex chars
	subdomain := slug + "-" + suffix

	// Block reserved infrastructure names
	if reservedNames[slug] || reservedNames[subdomain] {
		return nil, fmt.Errorf("app name '%s' is reserved and cannot be used", input.Name)
	}

	id := uuid.New().String()
	now := time.Now()

	appType := input.AppType
	if appType == "" {
		appType = entities.AppTypeWeb
	}

	exposure := input.Exposure
	if exposure == "" {
		exposure = entities.ExposurePublic
	}

	healthCheckPath := input.HealthCheckPath
	if healthCheckPath == "" {
		healthCheckPath = "/"
	}

	// Use nil for empty environment_id (nullable TEXT column)
	var envID interface{}
	if input.EnvironmentID != "" {
		envID = input.EnvironmentID
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO apps (id, user_id, project_id, name, deploy_source, repo_url, branch, image_url, registry_username, registry_password, framework, status, subdomain, port, app_type, command, cron_schedule, exposure, environment_id, health_check_path, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)`,
		id, input.UserID, input.ProjectID, input.Name, string(deploySource), input.RepoURL, branch,
		input.ImageURL, input.RegistryUsername, input.RegistryPassword,
		string(entities.FrameworkUnknown), string(entities.AppStatusPending), subdomain, port,
		string(appType), input.Command, input.CronSchedule, string(exposure),
		envID, healthCheckPath, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_apps_project_name") {
			return nil, fmt.Errorf("app '%s' already exists in this project", input.Name)
		}
		if strings.Contains(err.Error(), "idx_apps_subdomain") {
			return nil, fmt.Errorf("subdomain '%s' is already taken", subdomain)
		}
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return &entities.App{
		ID:               id,
		UserID:           input.UserID,
		ProjectID:        input.ProjectID,
		EnvironmentID:    input.EnvironmentID,
		Name:             input.Name,
		DeploySource:     deploySource,
		RepoURL:          input.RepoURL,
		Branch:           branch,
		ImageURL:         input.ImageURL,
		RegistryUser:     input.RegistryUsername,
		RegistryPassword: input.RegistryPassword,
		Framework:        entities.FrameworkUnknown,
		Status:           entities.AppStatusPending,
		Subdomain:        subdomain,
		Port:             port,
		AppType:          appType,
		Command:          input.Command,
		CronSchedule:     input.CronSchedule,
		Exposure:         exposure,
		Replicas:         1,
		HealthCheckPath:  healthCheckPath,
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

func scanApp(scan func(dest ...interface{}) error) (*entities.App, error) {
	var app entities.App
	var framework, status, deploySource, appType, exposure string
	var autoGatewayID, environmentID *string

	err := scan(&app.ID, &app.UserID, &app.ProjectID, &app.Name, &deploySource, &app.RepoURL, &app.Branch,
		&app.ImageURL, &app.RegistryUser, &app.RegistryPassword,
		&framework, &status, &app.Subdomain, &app.Port,
		&appType, &app.Command, &app.CronSchedule,
		&exposure, &autoGatewayID, &environmentID, &app.Replicas,
		&app.HealthCheckPath, &app.DeletedAt,
		&app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}

	app.DeploySource = entities.DeploySource(deploySource)
	app.Framework = entities.Framework(framework)
	app.Status = entities.AppStatus(status)
	app.AppType = entities.AppType(appType)
	if app.AppType == "" {
		app.AppType = entities.AppTypeWeb
	}
	app.Exposure = entities.AppExposure(exposure)
	if app.Exposure == "" {
		app.Exposure = entities.ExposurePublic
	}
	if autoGatewayID != nil {
		app.AutoGatewayID = *autoGatewayID
	}
	if environmentID != nil {
		app.EnvironmentID = *environmentID
	}
	if app.Replicas <= 0 {
		app.Replicas = 1
	}
	if app.HealthCheckPath == "" {
		app.HealthCheckPath = "/"
	}
	return &app, nil
}

const appColumns = `id, user_id, project_id, name, deploy_source, repo_url, branch, image_url, registry_username, registry_password, framework, status, subdomain, port, app_type, command, cron_schedule, exposure, auto_gateway_id, environment_id, replicas, health_check_path, deleted_at, created_at, updated_at`

func (r *PostgresAppRepository) GetApp(ctx context.Context, id string) (*entities.App, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+appColumns+` FROM apps WHERE id = $1 AND deleted_at IS NULL`, id)
	app, err := scanApp(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("app not found")
	}
	return app, nil
}

func (r *PostgresAppRepository) GetAppBySubdomain(ctx context.Context, subdomain string) (*entities.App, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+appColumns+` FROM apps WHERE subdomain = $1 AND deleted_at IS NULL`, subdomain)
	app, err := scanApp(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("app not found for subdomain '%s'", subdomain)
	}
	return app, nil
}

func (r *PostgresAppRepository) ListAppsByUser(ctx context.Context, userID string) ([]entities.App, error) {
	query := `SELECT ` + appColumns + ` FROM apps WHERE deleted_at IS NULL`
	var args []interface{}
	if userID != "" {
		query += ` AND user_id = $1`
		args = append(args, userID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}
	defer rows.Close()

	var apps []entities.App
	for rows.Next() {
		app, err := scanApp(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan app: %w", err)
		}
		apps = append(apps, *app)
	}

	return apps, nil
}

func (r *PostgresAppRepository) ListAppsByProject(ctx context.Context, projectID string) ([]entities.App, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+appColumns+` FROM apps WHERE project_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list apps by project: %w", err)
	}
	defer rows.Close()

	var apps []entities.App
	for rows.Next() {
		app, err := scanApp(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan app: %w", err)
		}
		apps = append(apps, *app)
	}
	return apps, nil
}

func (r *PostgresAppRepository) UpdateApp(ctx context.Context, id string, input *dto.UpdateAppInput) (*entities.App, error) {
	// Build dynamic SET clause
	sets := []string{"updated_at = now()"}
	args := []interface{}{}
	argIdx := 1

	if input.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*input.Status))
		argIdx++
	}
	if input.Framework != nil {
		sets = append(sets, fmt.Sprintf("framework = $%d", argIdx))
		args = append(args, string(*input.Framework))
		argIdx++
	}
	if input.Port != nil {
		sets = append(sets, fmt.Sprintf("port = $%d", argIdx))
		args = append(args, *input.Port)
		argIdx++
	}
	if input.Branch != nil {
		sets = append(sets, fmt.Sprintf("branch = $%d", argIdx))
		args = append(args, *input.Branch)
		argIdx++
	}
	if input.Replicas != nil {
		sets = append(sets, fmt.Sprintf("replicas = $%d", argIdx))
		args = append(args, *input.Replicas)
		argIdx++
	}
	if input.HealthCheckPath != nil {
		sets = append(sets, fmt.Sprintf("health_check_path = $%d", argIdx))
		args = append(args, *input.HealthCheckPath)
		argIdx++
	}
	if input.EnvironmentID != nil {
		sets = append(sets, fmt.Sprintf("environment_id = $%d", argIdx))
		args = append(args, *input.EnvironmentID)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE apps SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update app: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("app not found")
	}

	return r.GetApp(ctx, id)
}

func (r *PostgresAppRepository) DeleteApp(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM apps WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete app: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("app not found")
	}
	return nil
}

func (r *PostgresAppRepository) SoftDeleteApp(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE apps SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL", id)
	if err != nil {
		return fmt.Errorf("failed to soft delete app: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("app not found")
	}
	return nil
}

func (r *PostgresAppRepository) RestoreApp(ctx context.Context, id string) (*entities.App, error) {
	tag, err := r.pool.Exec(ctx,
		"UPDATE apps SET deleted_at = NULL, updated_at = now() WHERE id = $1 AND deleted_at IS NOT NULL", id)
	if err != nil {
		return nil, fmt.Errorf("failed to restore app: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("app not found or not deleted")
	}
	row := r.pool.QueryRow(ctx, `SELECT `+appColumns+` FROM apps WHERE id = $1`, id)
	return scanApp(row.Scan)
}

func (r *PostgresAppRepository) ListDeletedAppsByUser(ctx context.Context, userID string) ([]entities.App, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+appColumns+` FROM apps WHERE user_id = $1 AND deleted_at IS NOT NULL ORDER BY deleted_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list deleted apps: %w", err)
	}
	defer rows.Close()

	var apps []entities.App
	for rows.Next() {
		app, err := scanApp(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan app: %w", err)
		}
		apps = append(apps, *app)
	}
	return apps, nil
}

func (r *PostgresAppRepository) SetAutoGatewayID(ctx context.Context, appID, gatewayID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE apps SET auto_gateway_id = $1, updated_at = now() WHERE id = $2`,
		gatewayID, appID,
	)
	if err != nil {
		return fmt.Errorf("failed to set auto_gateway_id: %w", err)
	}
	return nil
}

func (r *PostgresAppRepository) CountAppsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM apps WHERE user_id = $1 AND deleted_at IS NULL", userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count apps: %w", err)
	}
	return count, nil
}

func (r *PostgresAppRepository) CountApps(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM apps WHERE deleted_at IS NULL").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count apps: %w", err)
	}
	return count, nil
}

func (r *PostgresAppRepository) ListAllApps(ctx context.Context) ([]entities.App, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+appColumns+` FROM apps WHERE deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list all apps: %w", err)
	}
	defer rows.Close()

	var apps []entities.App
	for rows.Next() {
		app, err := scanApp(rows.Scan)
		if err != nil {
			return nil, err
		}
		apps = append(apps, *app)
	}
	return apps, nil
}

// --- Deployments ---

func (r *PostgresAppRepository) CreateDeployment(ctx context.Context, appID, gitSHA string) (*entities.Deployment, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO deployments (id, app_id, git_sha, status, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, appID, gitSHA, string(entities.DeployStatusPending), now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	return &entities.Deployment{
		ID:        id,
		AppID:     appID,
		GitSHA:    gitSHA,
		Status:    entities.DeployStatusPending,
		CreatedAt: now,
	}, nil
}

func (r *PostgresAppRepository) GetDeployment(ctx context.Context, id string) (*entities.Deployment, error) {
	var d entities.Deployment
	var status string

	err := r.pool.QueryRow(ctx,
		`SELECT id, app_id, image_tag, git_sha, status, build_log, error, created_at
		 FROM deployments WHERE id = $1`, id,
	).Scan(&d.ID, &d.AppID, &d.ImageTag, &d.GitSHA, &status, &d.BuildLog, &d.Error, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("deployment not found")
	}

	d.Status = entities.DeploymentStatus(status)
	return &d, nil
}

func (r *PostgresAppRepository) ListDeployments(ctx context.Context, appID string, limit int) ([]entities.Deployment, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, app_id, image_tag, git_sha, status, build_log, error, created_at
		 FROM deployments WHERE app_id = $1 ORDER BY created_at DESC LIMIT $2`, appID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	defer rows.Close()

	var deployments []entities.Deployment
	for rows.Next() {
		var d entities.Deployment
		var status string
		if err := rows.Scan(&d.ID, &d.AppID, &d.ImageTag, &d.GitSHA, &status, &d.BuildLog, &d.Error, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan deployment: %w", err)
		}
		d.Status = entities.DeploymentStatus(status)
		deployments = append(deployments, d)
	}

	return deployments, nil
}

func (r *PostgresAppRepository) UpdateDeploymentStatus(ctx context.Context, id string, status entities.DeploymentStatus, buildLog, errMsg string) error {
	sets := []string{fmt.Sprintf("status = $1")}
	args := []interface{}{string(status)}
	argIdx := 2

	if buildLog != "" {
		sets = append(sets, fmt.Sprintf("build_log = $%d", argIdx))
		args = append(args, buildLog)
		argIdx++
	}
	if errMsg != "" {
		sets = append(sets, fmt.Sprintf("error = $%d", argIdx))
		args = append(args, errMsg)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE deployments SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("deployment not found")
	}
	return nil
}

func (r *PostgresAppRepository) GetActiveDeployment(ctx context.Context, appID string) (*entities.Deployment, error) {
	var d entities.Deployment
	var status string

	err := r.pool.QueryRow(ctx,
		`SELECT id, app_id, image_tag, git_sha, status, build_log, error, created_at
		 FROM deployments WHERE app_id = $1 AND status = $2
		 ORDER BY created_at DESC LIMIT 1`, appID, string(entities.DeployStatusActive),
	).Scan(&d.ID, &d.AppID, &d.ImageTag, &d.GitSHA, &status, &d.BuildLog, &d.Error, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("no active deployment found for app")
	}

	d.Status = entities.DeploymentStatus(status)
	return &d, nil
}

// --- Env Vars ---

func (r *PostgresAppRepository) SetEnvVars(ctx context.Context, appID string, vars map[string]string) error {
	for key, value := range vars {
		id := uuid.New().String()
		_, err := r.pool.Exec(ctx,
			`INSERT INTO app_env_vars (id, app_id, key, value)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (app_id, key) DO UPDATE SET value = $4`,
			id, appID, key, value,
		)
		if err != nil {
			return fmt.Errorf("failed to set env var '%s': %w", key, err)
		}
	}
	return nil
}

func (r *PostgresAppRepository) GetEnvVars(ctx context.Context, appID string) ([]entities.EnvVar, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, app_id, key, value FROM app_env_vars WHERE app_id = $1 ORDER BY key`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get env vars: %w", err)
	}
	defer rows.Close()

	var envVars []entities.EnvVar
	for rows.Next() {
		var ev entities.EnvVar
		if err := rows.Scan(&ev.ID, &ev.AppID, &ev.Key, &ev.Value); err != nil {
			return nil, fmt.Errorf("failed to scan env var: %w", err)
		}
		envVars = append(envVars, ev)
	}

	return envVars, nil
}

func (r *PostgresAppRepository) DeleteEnvVar(ctx context.Context, appID, key string) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM app_env_vars WHERE app_id = $1 AND key = $2", appID, key,
	)
	if err != nil {
		return fmt.Errorf("failed to delete env var: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("env var '%s' not found for app", key)
	}
	return nil
}

// --- Secrets ---

func (r *PostgresAppRepository) SetSecret(ctx context.Context, appID, key string, encryptedValue []byte) error {
	id := uuid.New().String()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO app_secrets (id, app_id, key, value_enc)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (app_id, key) DO UPDATE SET value_enc = $4`,
		id, appID, key, encryptedValue,
	)
	if err != nil {
		return fmt.Errorf("failed to set secret '%s': %w", key, err)
	}
	return nil
}

func (r *PostgresAppRepository) GetSecrets(ctx context.Context, appID string) ([]entities.Secret, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, app_id, key, created_at FROM app_secrets WHERE app_id = $1 ORDER BY key`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	}
	defer rows.Close()

	var secrets []entities.Secret
	for rows.Next() {
		var s entities.Secret
		if err := rows.Scan(&s.ID, &s.AppID, &s.Key, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan secret: %w", err)
		}
		secrets = append(secrets, s)
	}
	return secrets, nil
}

func (r *PostgresAppRepository) GetSecretValue(ctx context.Context, appID, key string) ([]byte, error) {
	var enc []byte
	err := r.pool.QueryRow(ctx,
		`SELECT value_enc FROM app_secrets WHERE app_id = $1 AND key = $2`, appID, key,
	).Scan(&enc)
	if err != nil {
		return nil, fmt.Errorf("secret '%s' not found", key)
	}
	return enc, nil
}

func (r *PostgresAppRepository) DeleteSecret(ctx context.Context, appID, key string) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM app_secrets WHERE app_id = $1 AND key = $2", appID, key,
	)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("secret '%s' not found for app", key)
	}
	return nil
}

// --- Releases ---

func (r *PostgresAppRepository) CreateRelease(ctx context.Context, appID string, input *dto.CreateReleaseInput) (*entities.Release, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO app_releases (id, app_id, image, git_sha, branch, message, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, appID, input.Image, input.GitSHA, input.Branch, input.Message, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create release: %w", err)
	}

	return &entities.Release{
		ID:        id,
		AppID:     appID,
		Image:     input.Image,
		GitSHA:    input.GitSHA,
		Branch:    input.Branch,
		Message:   input.Message,
		CreatedAt: now,
	}, nil
}

func (r *PostgresAppRepository) ListReleases(ctx context.Context, appID string, limit int) ([]entities.Release, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, app_id, image, git_sha, branch, message, created_at
		 FROM app_releases WHERE app_id = $1 ORDER BY created_at DESC LIMIT $2`, appID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}
	defer rows.Close()

	var releases []entities.Release
	for rows.Next() {
		var rel entities.Release
		if err := rows.Scan(&rel.ID, &rel.AppID, &rel.Image, &rel.GitSHA, &rel.Branch, &rel.Message, &rel.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan release: %w", err)
		}
		releases = append(releases, rel)
	}
	return releases, nil
}

func (r *PostgresAppRepository) GetRelease(ctx context.Context, id string) (*entities.Release, error) {
	var rel entities.Release
	err := r.pool.QueryRow(ctx,
		`SELECT id, app_id, image, git_sha, branch, message, created_at
		 FROM app_releases WHERE id = $1`, id,
	).Scan(&rel.ID, &rel.AppID, &rel.Image, &rel.GitSHA, &rel.Branch, &rel.Message, &rel.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("release not found")
	}
	return &rel, nil
}
