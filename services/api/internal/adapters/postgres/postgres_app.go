package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
		port = 8080
	}

	subdomain := strings.ToLower(strings.ReplaceAll(input.Name, "_", "-"))
	subdomain = strings.ReplaceAll(subdomain, " ", "-")

	id := uuid.New().String()
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO apps (id, user_id, name, deploy_source, repo_url, branch, image_url, registry_username, registry_password, framework, status, subdomain, port, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		id, input.UserID, input.Name, string(deploySource), input.RepoURL, branch,
		input.ImageURL, input.RegistryUsername, input.RegistryPassword,
		string(entities.FrameworkUnknown), string(entities.AppStatusPending), subdomain, port, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_apps_user_name") {
			return nil, fmt.Errorf("app '%s' already exists for this user", input.Name)
		}
		if strings.Contains(err.Error(), "idx_apps_subdomain") {
			return nil, fmt.Errorf("subdomain '%s' is already taken", subdomain)
		}
		return nil, fmt.Errorf("failed to create app: %w", err)
	}

	return &entities.App{
		ID:               id,
		UserID:           input.UserID,
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
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

func scanApp(scan func(dest ...interface{}) error) (*entities.App, error) {
	var app entities.App
	var framework, status, deploySource string

	err := scan(&app.ID, &app.UserID, &app.Name, &deploySource, &app.RepoURL, &app.Branch,
		&app.ImageURL, &app.RegistryUser, &app.RegistryPassword,
		&framework, &status, &app.Subdomain, &app.Port, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}

	app.DeploySource = entities.DeploySource(deploySource)
	app.Framework = entities.Framework(framework)
	app.Status = entities.AppStatus(status)
	return &app, nil
}

const appColumns = `id, user_id, name, deploy_source, repo_url, branch, image_url, registry_username, registry_password, framework, status, subdomain, port, created_at, updated_at`

func (r *PostgresAppRepository) GetApp(ctx context.Context, id string) (*entities.App, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+appColumns+` FROM apps WHERE id = $1`, id)
	app, err := scanApp(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("app not found")
	}
	return app, nil
}

func (r *PostgresAppRepository) GetAppBySubdomain(ctx context.Context, subdomain string) (*entities.App, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+appColumns+` FROM apps WHERE subdomain = $1`, subdomain)
	app, err := scanApp(row.Scan)
	if err != nil {
		return nil, fmt.Errorf("app not found for subdomain '%s'", subdomain)
	}
	return app, nil
}

func (r *PostgresAppRepository) ListAppsByUser(ctx context.Context, userID string) ([]entities.App, error) {
	query := `SELECT ` + appColumns + ` FROM apps`
	var args []interface{}
	if userID != "" {
		query += ` WHERE user_id = $1`
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

func (r *PostgresAppRepository) CountAppsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM apps WHERE user_id = $1", userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count apps: %w", err)
	}
	return count, nil
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
