package handlers

import (
	"bufio"
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/crypto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// EnvVarK8sSyncer syncs env vars to K8s Secret/ConfigMap.
type EnvVarK8sSyncer interface {
	CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte, labels map[string]string) error
	CreateConfigMap(ctx context.Context, namespace, name string, data map[string]string) error
}

// AppRestarter redeploys an app with the same image to apply updated env vars.
type AppRestarter interface {
	DeployApp(ctx context.Context, app *entities.App, imageTag string) error
}

// EnvVarHandler handles enhanced environment variable endpoints.
type EnvVarHandler struct {
	appRepo    ports.AppRepository
	envVarRepo ports.EnvVarRepository
	k8s        EnvVarK8sSyncer
	envCrypto  *crypto.EnvCrypto
	restarter  AppRestarter
	namespace  string
}

// NewEnvVarHandler creates a new EnvVarHandler.
func NewEnvVarHandler(appRepo ports.AppRepository, envVarRepo ports.EnvVarRepository) *EnvVarHandler {
	return &EnvVarHandler{appRepo: appRepo, envVarRepo: envVarRepo, namespace: "zenith-apps"}
}

// SetK8sClient sets the K8s client for env var syncing.
func (h *EnvVarHandler) SetK8sClient(k8s EnvVarK8sSyncer) {
	h.k8s = k8s
}

// SetEnvCrypto injects the encryption helper for secret values.
func (h *EnvVarHandler) SetEnvCrypto(c *crypto.EnvCrypto) {
	h.envCrypto = c
}

// SetRestarter injects the deployer used to trigger rolling restarts.
func (h *EnvVarHandler) SetRestarter(r AppRestarter) {
	h.restarter = r
}

type setEnvVarsRequest struct {
	Vars []envVarInput `json:"vars"`
}

type envVarInput struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
}

type importDotEnvRequest struct {
	Content string `json:"content"` // raw .env file content
}

type envVarResponse struct {
	ID            string `json:"id"`
	AppID         string `json:"app_id"`
	EnvironmentID string `json:"environment_id,omitempty"`
	Key           string `json:"key"`
	Value         string `json:"value"`
	IsSecret      bool   `json:"is_secret"`
	Source        string `json:"source"`
	SourceID      string `json:"source_id,omitempty"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

func toEnvVarResponse(ev *entities.AppEnvVar) envVarResponse {
	value := ev.Value
	if ev.IsSecret {
		value = "••••••••"
	}
	return envVarResponse{
		ID:            ev.ID,
		AppID:         ev.AppID,
		EnvironmentID: ev.EnvironmentID,
		Key:           ev.Key,
		Value:         value,
		IsSecret:      ev.IsSecret,
		Source:        string(ev.Source),
		SourceID:      ev.SourceID,
		CreatedAt:     ev.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     ev.UpdatedAt.Format(time.RFC3339),
	}
}

// envFromQuery reads the optional ?env= query param and returns the environment ID.
// Empty string means production/default.
func envFromQuery(c *fiber.Ctx) string {
	return c.Query("env", "")
}

func (h *EnvVarHandler) verifyAppOwnership(c *fiber.Ctx, appID string) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}
	if app.UserID != userID {
		return NewForbidden("not your app")
	}
	return nil
}

// encryptIfSecret encrypts value with the user's derived key when is_secret=true.
// Falls back to plaintext if crypto is not configured.
func (h *EnvVarHandler) encryptIfSecret(userID, value string, isSecret bool) (string, error) {
	if !isSecret || h.envCrypto == nil {
		return value, nil
	}
	return h.envCrypto.Encrypt(userID, value)
}

// Set handles POST /apps/:appId/env-v2
// Optional query param: ?env=<environmentID>  (omit = production/default)
func (h *EnvVarHandler) Set(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if err := h.verifyAppOwnership(c, appID); err != nil {
		return err
	}

	environmentID := envFromQuery(c)

	var req setEnvVarsRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if len(req.Vars) == 0 {
		return NewBadRequest("vars is required and must not be empty")
	}

	seen := make(map[string]bool)
	for _, v := range req.Vars {
		if v.Key == "" {
			return NewBadRequest("env var key cannot be empty")
		}
		if seen[v.Key] {
			return NewBadRequest("duplicate key: " + v.Key)
		}
		seen[v.Key] = true
	}

	userID, _ := c.Locals("user_id").(string)
	envVars := make([]entities.AppEnvVar, 0, len(req.Vars))
	for _, v := range req.Vars {
		value, err := h.encryptIfSecret(userID, v.Value, v.IsSecret)
		if err != nil {
			return NewInternal("failed to encrypt secret value")
		}
		envVars = append(envVars, entities.AppEnvVar{
			AppID:         appID,
			EnvironmentID: environmentID,
			Key:           v.Key,
			Value:         value,
			IsSecret:      v.IsSecret,
			Source:        entities.EnvVarSourceManual,
		})
	}

	if err := h.envVarRepo.BulkSetEnvVars(c.Context(), appID, envVars); err != nil {
		slog.Error("failed to set env vars", "error", err)
		return NewInternal("failed to set environment variables")
	}

	allVars, err := h.envVarRepo.GetEnvVarsByEnvironment(c.Context(), appID, environmentID)
	if err != nil {
		slog.Error("failed to get env vars", "error", err)
		return NewInternal("failed to retrieve environment variables")
	}

	if h.k8s != nil {
		h.syncEnvVarsToK8s(c.Context(), appID, allVars)
	}

	items := make([]envVarResponse, 0, len(allVars))
	for i := range allVars {
		items = append(items, toEnvVarResponse(&allVars[i]))
	}

	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// List handles GET /apps/:appId/env-v2
// Optional query param: ?env=<environmentID>
func (h *EnvVarHandler) List(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if err := h.verifyAppOwnership(c, appID); err != nil {
		return err
	}

	environmentID := envFromQuery(c)

	vars, err := h.envVarRepo.GetEnvVarsByEnvironment(c.Context(), appID, environmentID)
	if err != nil {
		slog.Error("failed to get env vars", "error", err)
		return NewInternal("failed to retrieve environment variables")
	}

	items := make([]envVarResponse, 0, len(vars))
	for i := range vars {
		items = append(items, toEnvVarResponse(&vars[i]))
	}

	return c.JSON(fiber.Map{"items": items, "total": len(items)})
}

// Delete handles DELETE /apps/:appId/env-v2/:varId
func (h *EnvVarHandler) Delete(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if err := h.verifyAppOwnership(c, appID); err != nil {
		return err
	}

	varID := c.Params("varId")
	if err := h.envVarRepo.DeleteEnvVar(c.Context(), varID); err != nil {
		return NewNotFound("environment variable not found")
	}

	// Re-sync remaining env vars to K8s (best-effort)
	if h.k8s != nil {
		environmentID := envFromQuery(c)
		remaining, rerr := h.envVarRepo.GetEnvVarsByEnvironment(c.Context(), appID, environmentID)
		if rerr == nil {
			h.syncEnvVarsToK8s(c.Context(), appID, remaining)
		}
	}

	return c.JSON(fiber.Map{"message": "environment variable deleted"})
}

// ImportDotEnv handles POST /apps/:appId/env-v2/import
// Body: { "content": ".env file content as string" }
// Optional query param: ?env=<environmentID>
func (h *EnvVarHandler) ImportDotEnv(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if err := h.verifyAppOwnership(c, appID); err != nil {
		return err
	}

	environmentID := envFromQuery(c)

	var req importDotEnvRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if strings.TrimSpace(req.Content) == "" {
		return NewBadRequest("content is required")
	}

	parsed, err := parseDotEnv(req.Content)
	if err != nil {
		return NewBadRequest("invalid .env format: " + err.Error())
	}
	if len(parsed) == 0 {
		return NewBadRequest("no valid KEY=VALUE pairs found in content")
	}
	if len(parsed) > 100 {
		return NewBadRequest("too many variables (max 100 per import)")
	}

	importUserID, _ := c.Locals("user_id").(string)
	envVars := make([]entities.AppEnvVar, 0, len(parsed))
	for key, value := range parsed {
		isSecret := looksLikeSecret(key)
		encValue, err := h.encryptIfSecret(importUserID, value, isSecret)
		if err != nil {
			return NewInternal("failed to encrypt secret value for key: " + key)
		}
		envVars = append(envVars, entities.AppEnvVar{
			AppID:         appID,
			EnvironmentID: environmentID,
			Key:           key,
			Value:         encValue,
			IsSecret:      isSecret,
			Source:        entities.EnvVarSourceManual,
		})
	}

	if err := h.envVarRepo.BulkSetEnvVars(c.Context(), appID, envVars); err != nil {
		slog.Error("failed to import .env vars", "error", err)
		return NewInternal("failed to import environment variables")
	}

	allVars, err := h.envVarRepo.GetEnvVarsByEnvironment(c.Context(), appID, environmentID)
	if err != nil {
		slog.Error("failed to get env vars after import", "error", err)
		return NewInternal("failed to retrieve environment variables")
	}

	if h.k8s != nil {
		h.syncEnvVarsToK8s(c.Context(), appID, allVars)
	}

	items := make([]envVarResponse, 0, len(allVars))
	for i := range allVars {
		items = append(items, toEnvVarResponse(&allVars[i]))
	}

	return c.JSON(fiber.Map{
		"imported": len(parsed),
		"items":    items,
		"total":    len(items),
	})
}

// parseDotEnv parses a .env file string into a key→value map.
// Supports: KEY=VALUE, KEY="VALUE", # comments, blank lines, export KEY=VALUE.
func parseDotEnv(content string) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip leading "export "
		line = strings.TrimPrefix(line, "export ")

		idx := strings.Index(line, "=")
		if idx < 1 {
			continue // skip lines without =
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Validate key: only letters, digits, underscores
		if !isValidEnvKey(key) {
			continue
		}

		// Strip surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Strip inline comments (unquoted # after space)
		if idx := strings.Index(value, " #"); idx >= 0 {
			value = strings.TrimSpace(value[:idx])
		}

		result[key] = value
	}

	return result, scanner.Err()
}

func isValidEnvKey(key string) bool {
	if len(key) == 0 {
		return false
	}
	for _, c := range key {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// looksLikeSecret returns true if the key name suggests a sensitive value.
func looksLikeSecret(key string) bool {
	lower := strings.ToLower(key)
	hints := []string{"password", "passwd", "secret", "token", "api_key", "apikey",
		"private_key", "privatekey", "auth", "credential", "cert", "key"}
	for _, h := range hints {
		if strings.Contains(lower, h) {
			return true
		}
	}
	return false
}

// syncEnvVarsToK8s syncs env vars to K8s Secret (secrets) and ConfigMap (plain vars).
func (h *EnvVarHandler) syncEnvVarsToK8s(ctx context.Context, appID string, vars []entities.AppEnvVar) {
	secretData := make(map[string][]byte)
	configData := make(map[string]string)

	for _, v := range vars {
		if v.IsSecret {
			secretData[v.Key] = []byte(v.Value)
		} else {
			configData[v.Key] = v.Value
		}
	}

	labels := map[string]string{
		"app.zenith.dev/app-id": appID,
		"app.zenith.dev/type":   "env",
	}

	secretName := "env-" + appID + "-secrets"
	configName := "env-" + appID + "-config"

	if len(secretData) > 0 {
		if err := h.k8s.CreateSecret(ctx, h.namespace, secretName, secretData, labels); err != nil {
			slog.Warn("failed to sync env secrets to K8s", "app_id", appID, "error", err)
		}
	}

	if len(configData) > 0 {
		if err := h.k8s.CreateConfigMap(ctx, h.namespace, configName, configData); err != nil {
			slog.Warn("failed to sync env config to K8s", "app_id", appID, "error", err)
		}
	}
}

// Apply handles POST /apps/:appId/env/apply
// Triggers a rolling restart of the app to pick up the latest env vars.
// Gets the current image from the last release and re-deploys with updated env vars.
func (h *EnvVarHandler) Apply(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if err := h.verifyAppOwnership(c, appID); err != nil {
		return err
	}

	if h.restarter == nil {
		return NewInternal("restart not available")
	}

	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}

	// Get current image from latest release
	releases, err := h.appRepo.ListReleases(c.Context(), appID, 1)
	if err != nil || len(releases) == 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "no deployment found — deploy the app first before applying env changes",
		})
	}

	imageTag := releases[0].Image
	slog.Info("applying env vars via rolling restart", "app", app.Name, "image", imageTag)

	if err := h.restarter.DeployApp(c.Context(), app, imageTag); err != nil {
		slog.Error("failed to apply env vars", "app", app.Name, "error", err)
		return NewInternal("failed to restart app: " + err.Error())
	}

	return c.JSON(fiber.Map{
		"message": "rolling restart triggered",
		"app":     app.Name,
		"image":   imageTag,
	})
}
