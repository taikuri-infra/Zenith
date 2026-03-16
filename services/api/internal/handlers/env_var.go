package handlers

import (
	"context"
	"log/slog"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// EnvVarK8sSyncer syncs env vars to K8s Secret/ConfigMap.
type EnvVarK8sSyncer interface {
	CreateSecret(ctx context.Context, namespace, name string, data map[string][]byte, labels map[string]string) error
	CreateConfigMap(ctx context.Context, namespace, name string, data map[string]string) error
}

// EnvVarHandler handles enhanced environment variable endpoints.
type EnvVarHandler struct {
	appRepo    ports.AppRepository
	envVarRepo ports.EnvVarRepository
	k8s        EnvVarK8sSyncer
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

type setEnvVarsRequest struct {
	Vars []envVarInput `json:"vars"`
}

type envVarInput struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
}

type envVarResponse struct {
	ID        string `json:"id"`
	AppID     string `json:"app_id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsSecret  bool   `json:"is_secret"`
	Source    string `json:"source"`
	SourceID  string `json:"source_id,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toEnvVarResponse(ev *entities.AppEnvVar) envVarResponse {
	value := ev.Value
	if ev.IsSecret {
		value = "••••••••"
	}
	return envVarResponse{
		ID:        ev.ID,
		AppID:     ev.AppID,
		Key:       ev.Key,
		Value:     value,
		IsSecret:  ev.IsSecret,
		Source:    string(ev.Source),
		SourceID:  ev.SourceID,
		CreatedAt: ev.CreatedAt.Format(time.RFC3339),
		UpdatedAt: ev.UpdatedAt.Format(time.RFC3339),
	}
}

// Set handles POST /apps/:appId/env
func (h *EnvVarHandler) Set(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	appID := c.Params("appId")
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}
	if app.UserID != userID {
		return NewForbidden("not your app")
	}

	var req setEnvVarsRequest
	if err := c.BodyParser(&req); err != nil {
		return NewBadRequest("invalid request body")
	}
	if len(req.Vars) == 0 {
		return NewBadRequest("vars is required and must not be empty")
	}

	// Validate no duplicate keys
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

	// Convert to entities
	envVars := make([]entities.AppEnvVar, 0, len(req.Vars))
	for _, v := range req.Vars {
		envVars = append(envVars, entities.AppEnvVar{
			AppID:    appID,
			Key:      v.Key,
			Value:    v.Value,
			IsSecret: v.IsSecret,
			Source:   entities.EnvVarSourceManual,
		})
	}

	if err := h.envVarRepo.BulkSetEnvVars(c.Context(), appID, envVars); err != nil {
		slog.Error("failed to set env vars", "error", err)
		return NewInternal("failed to set environment variables")
	}

	// Return updated list
	allVars, err := h.envVarRepo.GetEnvVars(c.Context(), appID)
	if err != nil {
		slog.Error("failed to get env vars", "error", err)
		return NewInternal("failed to retrieve environment variables")
	}

	// Sync to K8s Secret/ConfigMap (best-effort, don't fail the request)
	if h.k8s != nil {
		h.syncEnvVarsToK8s(c.Context(), appID, allVars)
	}

	items := make([]envVarResponse, 0, len(allVars))
	for i := range allVars {
		items = append(items, toEnvVarResponse(&allVars[i]))
	}

	return c.JSON(fiber.Map{
		"items": items,
		"total": len(items),
	})
}

// List handles GET /apps/:appId/env
func (h *EnvVarHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	appID := c.Params("appId")
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}
	if app.UserID != userID {
		return NewForbidden("not your app")
	}

	vars, err := h.envVarRepo.GetEnvVars(c.Context(), appID)
	if err != nil {
		slog.Error("failed to get env vars", "error", err)
		return NewInternal("failed to retrieve environment variables")
	}

	items := make([]envVarResponse, 0, len(vars))
	for i := range vars {
		items = append(items, toEnvVarResponse(&vars[i]))
	}

	return c.JSON(fiber.Map{
		"items": items,
		"total": len(items),
	})
}

// Delete handles DELETE /apps/:appId/env/:varId
func (h *EnvVarHandler) Delete(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	if userID == "" {
		return NewUnauthorized("authentication required")
	}

	appID := c.Params("appId")
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return NewNotFound("app not found")
	}
	if app.UserID != userID {
		return NewForbidden("not your app")
	}

	varID := c.Params("varId")
	if err := h.envVarRepo.DeleteEnvVar(c.Context(), varID); err != nil {
		return NewNotFound("environment variable not found")
	}

	// Re-sync remaining env vars to K8s (best-effort)
	if h.k8s != nil {
		remaining, rerr := h.envVarRepo.GetEnvVars(c.Context(), appID)
		if rerr == nil {
			h.syncEnvVarsToK8s(c.Context(), appID, remaining)
		}
	}

	return c.JSON(fiber.Map{"message": "environment variable deleted"})
}

// syncEnvVarsToK8s syncs all env vars for an app to K8s Secret (secrets) and ConfigMap (plain vars).
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
