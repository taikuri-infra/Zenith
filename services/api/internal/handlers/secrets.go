package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/dotechhq/zenith/services/api/pkg/crypto"
	"github.com/gofiber/fiber/v2"
)

// SecretHandler handles encrypted key-value secrets per app.
type SecretHandler struct {
	appRepo    store.AppRepository
	cryptoKey  []byte
}

// NewSecretHandler creates a SecretHandler with the decoded AES key.
// Returns nil if the key is empty (dev mode — secrets disabled).
func NewSecretHandler(appRepo store.AppRepository, hexKey string) (*SecretHandler, error) {
	if hexKey == "" {
		return nil, nil //nolint:nilnil // intentional: dev mode
	}
	key, err := crypto.KeyFromHex(hexKey)
	if err != nil {
		return nil, err
	}
	return &SecretHandler{appRepo: appRepo, cryptoKey: key}, nil
}

// ListSecrets GET /api/v1/apps/:appId/secrets
// Returns all secret keys (never values). Decrypted values require POST individual key.
func (h *SecretHandler) ListSecrets(c *fiber.Ctx) error {
	appID := c.Params("appId")
	if appID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "appId is required")
	}

	secrets, err := h.appRepo.GetSecrets(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list secrets")
	}

	return c.JSON(fiber.Map{"secrets": secrets})
}

// GetSecretValue GET /api/v1/apps/:appId/secrets/:key/value
// Returns the decrypted value of a single secret.
func (h *SecretHandler) GetSecretValue(c *fiber.Ctx) error {
	appID := c.Params("appId")
	key := c.Params("key")

	enc, err := h.appRepo.GetSecretValue(c.Context(), appID, key)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "secret not found")
	}

	value, err := crypto.Decrypt(h.cryptoKey, enc)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to decrypt secret")
	}

	return c.JSON(fiber.Map{"key": key, "value": value})
}

// SetSecret POST /api/v1/apps/:appId/secrets
// Creates or updates a secret (value is encrypted before storage).
func (h *SecretHandler) SetSecret(c *fiber.Ctx) error {
	appID := c.Params("appId")

	var input dto.CreateSecretInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Key == "" || input.Value == "" {
		return fiber.NewError(fiber.StatusBadRequest, "key and value are required")
	}

	enc, err := crypto.Encrypt(h.cryptoKey, input.Value)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to encrypt secret")
	}

	if err := h.appRepo.SetSecret(c.Context(), appID, input.Key, enc); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to store secret")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"key":    input.Key,
		"status": "created",
	})
}

// DeleteSecret DELETE /api/v1/apps/:appId/secrets/:key
func (h *SecretHandler) DeleteSecret(c *fiber.Ctx) error {
	appID := c.Params("appId")
	key := c.Params("key")

	if err := h.appRepo.DeleteSecret(c.Context(), appID, key); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "secret not found")
	}

	return c.JSON(fiber.Map{"key": key, "status": "deleted"})
}
