package handlers

import (
	"log/slog"

	"github.com/dotechhq/zenith/services/api/internal/crypto"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// AdminCryptoHandler provides admin endpoints for crypto key rotation.
type AdminCryptoHandler struct {
	envCrypto  *crypto.EnvCrypto
	envVarRepo ports.EnvVarRepository
	appRepo    ports.AppRepository
}

// NewAdminCryptoHandler creates a new AdminCryptoHandler.
func NewAdminCryptoHandler(envCrypto *crypto.EnvCrypto, envVarRepo ports.EnvVarRepository, appRepo ports.AppRepository) *AdminCryptoHandler {
	return &AdminCryptoHandler{
		envCrypto:  envCrypto,
		envVarRepo: envVarRepo,
		appRepo:    appRepo,
	}
}

// RotateKeys handles POST /api/v2/admin/crypto/rotate
// Re-encrypts all env var values from old key versions to the current key version.
// This is a batch operation — call after updating SECRETS_ENCRYPTION_KEY and restarting.
func (h *AdminCryptoHandler) RotateKeys(c *fiber.Ctx) error {
	if h.envCrypto == nil {
		return NewBadRequest("encryption not configured")
	}

	// List all apps to iterate their env vars
	apps, err := h.appRepo.ListAllApps(c.Context())
	if err != nil {
		slog.Error("crypto rotate: failed to list apps", "error", err)
		return NewInternal("failed to list apps")
	}

	totalVars := 0
	rotatedVars := 0
	errors := 0

	for _, app := range apps {
		// Re-encrypt enhanced env vars (V2)
		if h.envVarRepo != nil {
			vars, vErr := h.envVarRepo.GetEnvVars(c.Context(), app.ID)
			if vErr != nil {
				slog.Warn("crypto rotate: failed to get env vars", "app_id", app.ID, "error", vErr)
				errors++
				continue
			}
			for i := range vars {
				totalVars++
				newVal, changed, reErr := h.envCrypto.ReEncrypt(app.UserID, vars[i].Value)
				if reErr != nil {
					slog.Warn("crypto rotate: re-encrypt failed", "app_id", app.ID, "var", vars[i].Key, "error", reErr)
					errors++
					continue
				}
				if changed {
					vars[i].Value = newVal
					if err := h.envVarRepo.SetEnvVar(c.Context(), &vars[i]); err != nil {
						slog.Warn("crypto rotate: save failed", "app_id", app.ID, "var", vars[i].Key, "error", err)
						errors++
						continue
					}
					rotatedVars++
				}
			}
		}

		// Re-encrypt legacy env vars (AppRepository)
		legacyVars, lErr := h.appRepo.GetEnvVars(c.Context(), app.ID)
		if lErr != nil {
			continue // some apps might not have legacy vars
		}
		for _, ev := range legacyVars {
			totalVars++
			newVal, changed, reErr := h.envCrypto.ReEncrypt(app.UserID, ev.Value)
			if reErr != nil {
				slog.Warn("crypto rotate: re-encrypt legacy var failed", "app_id", app.ID, "key", ev.Key, "error", reErr)
				errors++
				continue
			}
			if changed {
				if err := h.appRepo.SetEnvVars(c.Context(), app.ID, map[string]string{ev.Key: newVal}); err != nil {
					slog.Warn("crypto rotate: save legacy var failed", "app_id", app.ID, "key", ev.Key, "error", err)
					errors++
					continue
				}
				rotatedVars++
			}
		}

		// Registry passwords are re-encrypted on next app update (user-driven).
	}

	slog.Info("crypto rotate complete", "total_vars", totalVars, "rotated", rotatedVars, "errors", errors)

	return c.JSON(fiber.Map{
		"total_vars":   totalVars,
		"rotated":      rotatedVars,
		"errors":       errors,
		"total_apps":   len(apps),
	})
}
