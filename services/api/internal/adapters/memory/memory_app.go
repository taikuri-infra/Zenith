package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
)

// MemoryAppRepository is an in-memory AppRepository for testing and development.
type MemoryAppRepository struct {
	mu          sync.RWMutex
	apps        map[string]*entities.App
	deployments map[string]*entities.Deployment
	envVars     map[string]*entities.EnvVar
	secrets     map[string]*memSecret // key: id
	releases    map[string]*entities.Release
}

type memSecret struct {
	entities.Secret
	ValueEnc []byte
}

// NewMemoryAppRepository creates a new in-memory AppRepository.
func NewMemoryAppRepository() *MemoryAppRepository {
	return &MemoryAppRepository{
		apps:        make(map[string]*entities.App),
		deployments: make(map[string]*entities.Deployment),
		envVars:     make(map[string]*entities.EnvVar),
		secrets:     make(map[string]*memSecret),
		releases:    make(map[string]*entities.Release),
	}
}

// --- Apps ---

func (r *MemoryAppRepository) CreateApp(ctx context.Context, input *dto.CreateAppInput) (*entities.App, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

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

	// Check for duplicate name under same project
	for _, a := range r.apps {
		if a.ProjectID == input.ProjectID && a.Name == input.Name {
			return nil, fmt.Errorf("app '%s' already exists in this project", input.Name)
		}
	}

	branch := input.Branch
	if branch == "" && deploySource == entities.DeploySourceGit {
		branch = "main"
	}

	port := input.Port
	if port == 0 {
		port = 8080
	}

	// Derive subdomain from name (lowercase, replace spaces/underscores with dash)
	subdomain := strings.ToLower(strings.ReplaceAll(input.Name, "_", "-"))
	subdomain = strings.ReplaceAll(subdomain, " ", "-")

	appType := input.AppType
	if appType == "" {
		appType = entities.AppTypeWeb
	}

	now := time.Now()
	app := &entities.App{
		ID:               uuid.New().String(),
		UserID:           input.UserID,
		ProjectID:        input.ProjectID,
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
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	r.apps[app.ID] = app
	return app, nil
}

func (r *MemoryAppRepository) GetApp(ctx context.Context, id string) (*entities.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	app, ok := r.apps[id]
	if !ok || app.DeletedAt != nil {
		return nil, fmt.Errorf("app not found")
	}
	return app, nil
}

func (r *MemoryAppRepository) GetAppBySubdomain(ctx context.Context, subdomain string) (*entities.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, a := range r.apps {
		if a.Subdomain == subdomain && a.DeletedAt == nil {
			return a, nil
		}
	}
	return nil, fmt.Errorf("app not found for subdomain '%s'", subdomain)
}

func (r *MemoryAppRepository) ListAppsByUser(ctx context.Context, userID string) ([]entities.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.App
	for _, a := range r.apps {
		if a.UserID == userID && a.DeletedAt == nil {
			result = append(result, *a)
		}
	}

	// Sort by created_at desc for deterministic ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

func (r *MemoryAppRepository) ListAppsByProject(ctx context.Context, projectID string) ([]entities.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.App
	for _, a := range r.apps {
		if a.ProjectID == projectID && a.DeletedAt == nil {
			result = append(result, *a)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (r *MemoryAppRepository) UpdateApp(ctx context.Context, id string, input *dto.UpdateAppInput) (*entities.App, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	app, ok := r.apps[id]
	if !ok {
		return nil, fmt.Errorf("app not found")
	}

	if input.Status != nil {
		app.Status = *input.Status
	}
	if input.Framework != nil {
		app.Framework = *input.Framework
	}
	if input.Port != nil {
		app.Port = *input.Port
	}
	if input.Branch != nil {
		app.Branch = *input.Branch
	}
	if input.Replicas != nil {
		app.Replicas = *input.Replicas
	}
	if input.HealthCheckPath != nil {
		app.HealthCheckPath = *input.HealthCheckPath
	}
	if input.EnvironmentID != nil {
		app.EnvironmentID = *input.EnvironmentID
	}
	app.UpdatedAt = time.Now()

	return app, nil
}

func (r *MemoryAppRepository) DeleteApp(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.apps[id]; !ok {
		return fmt.Errorf("app not found")
	}

	delete(r.apps, id)

	// Cascade delete deployments
	for dID, d := range r.deployments {
		if d.AppID == id {
			delete(r.deployments, dID)
		}
	}

	// Cascade delete env vars
	for eID, e := range r.envVars {
		if e.AppID == id {
			delete(r.envVars, eID)
		}
	}

	// Cascade delete secrets
	for sID, s := range r.secrets {
		if s.AppID == id {
			delete(r.secrets, sID)
		}
	}

	return nil
}

func (r *MemoryAppRepository) SoftDeleteApp(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	app, ok := r.apps[id]
	if !ok || app.DeletedAt != nil {
		return fmt.Errorf("app not found")
	}
	now := time.Now()
	app.DeletedAt = &now
	return nil
}

func (r *MemoryAppRepository) RestoreApp(_ context.Context, id string) (*entities.App, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	app, ok := r.apps[id]
	if !ok || app.DeletedAt == nil {
		return nil, fmt.Errorf("app not found or not deleted")
	}
	app.DeletedAt = nil
	app.UpdatedAt = time.Now()
	return app, nil
}

func (r *MemoryAppRepository) ListDeletedAppsByUser(_ context.Context, userID string) ([]entities.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []entities.App
	for _, a := range r.apps {
		if a.UserID == userID && a.DeletedAt != nil {
			result = append(result, *a)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].DeletedAt.After(*result[j].DeletedAt)
	})
	return result, nil
}

func (r *MemoryAppRepository) SetAutoGatewayID(_ context.Context, appID, gatewayID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	app, ok := r.apps[appID]
	if !ok {
		return fmt.Errorf("app not found")
	}
	app.AutoGatewayID = gatewayID
	r.apps[appID] = app
	return nil
}

func (r *MemoryAppRepository) CountAppsByUser(ctx context.Context, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, a := range r.apps {
		if a.UserID == userID && a.DeletedAt == nil {
			count++
		}
	}
	return count, nil
}

func (r *MemoryAppRepository) CountApps(ctx context.Context) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, a := range r.apps {
		if a.DeletedAt == nil {
			count++
		}
	}
	return count, nil
}

func (r *MemoryAppRepository) ListAllApps(_ context.Context) ([]entities.App, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var apps []entities.App
	for _, a := range r.apps {
		if a.DeletedAt == nil {
			apps = append(apps, *a)
		}
	}
	return apps, nil
}

// --- Deployments ---

func (r *MemoryAppRepository) CreateDeployment(ctx context.Context, appID, gitSHA string) (*entities.Deployment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.apps[appID]; !ok {
		return nil, fmt.Errorf("app not found")
	}

	d := &entities.Deployment{
		ID:        uuid.New().String(),
		AppID:     appID,
		GitSHA:    gitSHA,
		Status:    entities.DeployStatusPending,
		CreatedAt: time.Now(),
	}

	r.deployments[d.ID] = d
	return d, nil
}

func (r *MemoryAppRepository) GetDeployment(ctx context.Context, id string) (*entities.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, ok := r.deployments[id]
	if !ok {
		return nil, fmt.Errorf("deployment not found")
	}
	return d, nil
}

func (r *MemoryAppRepository) ListDeployments(ctx context.Context, appID string, limit int) ([]entities.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.Deployment
	for _, d := range r.deployments {
		if d.AppID == appID {
			result = append(result, *d)
		}
	}

	// Sort by created_at desc (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (r *MemoryAppRepository) UpdateDeploymentStatus(ctx context.Context, id string, status entities.DeploymentStatus, buildLog, errMsg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.deployments[id]
	if !ok {
		return fmt.Errorf("deployment not found")
	}

	d.Status = status
	if buildLog != "" {
		d.BuildLog = buildLog
	}
	if errMsg != "" {
		d.Error = errMsg
	}

	return nil
}

func (r *MemoryAppRepository) GetActiveDeployment(ctx context.Context, appID string) (*entities.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, d := range r.deployments {
		if d.AppID == appID && d.Status == entities.DeployStatusActive {
			return d, nil
		}
	}
	return nil, fmt.Errorf("no active deployment found for app")
}

// --- Env Vars ---

func (r *MemoryAppRepository) SetEnvVars(ctx context.Context, appID string, vars map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.apps[appID]; !ok {
		return fmt.Errorf("app not found")
	}

	for key, value := range vars {
		// Check if env var already exists — update if so
		found := false
		for _, e := range r.envVars {
			if e.AppID == appID && e.Key == key {
				e.Value = value
				found = true
				break
			}
		}
		if !found {
			ev := &entities.EnvVar{
				ID:    uuid.New().String(),
				AppID: appID,
				Key:   key,
				Value: value,
			}
			r.envVars[ev.ID] = ev
		}
	}

	return nil
}

func (r *MemoryAppRepository) GetEnvVars(ctx context.Context, appID string) ([]entities.EnvVar, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.EnvVar
	for _, e := range r.envVars {
		if e.AppID == appID {
			result = append(result, *e)
		}
	}

	// Sort by key for deterministic order
	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})

	return result, nil
}

func (r *MemoryAppRepository) DeleteEnvVar(ctx context.Context, appID, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, e := range r.envVars {
		if e.AppID == appID && e.Key == key {
			delete(r.envVars, id)
			return nil
		}
	}
	return fmt.Errorf("env var '%s' not found for app", key)
}

// --- Secrets ---

func (r *MemoryAppRepository) SetSecret(ctx context.Context, appID, key string, encryptedValue []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.apps[appID]; !ok {
		return fmt.Errorf("app not found")
	}

	for _, s := range r.secrets {
		if s.AppID == appID && s.Key == key {
			s.ValueEnc = encryptedValue
			return nil
		}
	}

	s := &memSecret{
		Secret: entities.Secret{
			ID:        uuid.New().String(),
			AppID:     appID,
			Key:       key,
			CreatedAt: time.Now(),
		},
		ValueEnc: encryptedValue,
	}
	r.secrets[s.ID] = s
	return nil
}

func (r *MemoryAppRepository) GetSecrets(ctx context.Context, appID string) ([]entities.Secret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.Secret
	for _, s := range r.secrets {
		if s.AppID == appID {
			result = append(result, s.Secret)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})
	return result, nil
}

func (r *MemoryAppRepository) GetSecretValue(ctx context.Context, appID, key string) ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, s := range r.secrets {
		if s.AppID == appID && s.Key == key {
			return s.ValueEnc, nil
		}
	}
	return nil, fmt.Errorf("secret '%s' not found", key)
}

func (r *MemoryAppRepository) DeleteSecret(ctx context.Context, appID, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, s := range r.secrets {
		if s.AppID == appID && s.Key == key {
			delete(r.secrets, id)
			return nil
		}
	}
	return fmt.Errorf("secret '%s' not found for app", key)
}

// --- Releases ---

func (r *MemoryAppRepository) CreateRelease(ctx context.Context, appID string, input *dto.CreateReleaseInput) (*entities.Release, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.apps[appID]; !ok {
		return nil, fmt.Errorf("app not found")
	}

	rel := &entities.Release{
		ID:        uuid.New().String(),
		AppID:     appID,
		Image:     input.Image,
		GitSHA:    input.GitSHA,
		Branch:    input.Branch,
		Message:   input.Message,
		CreatedAt: time.Now(),
	}
	r.releases[rel.ID] = rel
	return rel, nil
}

func (r *MemoryAppRepository) ListReleases(ctx context.Context, appID string, limit int) ([]entities.Release, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []entities.Release
	for _, rel := range r.releases {
		if rel.AppID == appID {
			result = append(result, *rel)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *MemoryAppRepository) GetRelease(ctx context.Context, id string) (*entities.Release, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rel, ok := r.releases[id]
	if !ok {
		return nil, fmt.Errorf("release not found")
	}
	return rel, nil
}
