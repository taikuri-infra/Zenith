package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// SessionHandler manages user session operations.
type SessionHandler struct {
	sessionRepo store.SessionRepository
}

// NewSessionHandler creates a new SessionHandler.
func NewSessionHandler(sessionRepo store.SessionRepository) *SessionHandler {
	return &SessionHandler{sessionRepo: sessionRepo}
}

// List returns all active sessions for the authenticated user.
// GET /api/v1/auth/sessions
func (h *SessionHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	sessions, err := h.sessionRepo.ListSessionsByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"items": sessions, "total": len(sessions)})
}

// Revoke deletes a specific session.
// DELETE /api/v1/auth/sessions/:sessionId
func (h *SessionHandler) Revoke(c *fiber.Ctx) error {
	sessionID := c.Params("sessionId")
	userID, _ := c.Locals("user_id").(string)

	session, err := h.sessionRepo.GetSession(c.Context(), sessionID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "session not found")
	}
	if session.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your session")
	}

	if err := h.sessionRepo.DeleteSession(c.Context(), sessionID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "session revoked"})
}

// RevokeAll deletes all sessions for the authenticated user.
// DELETE /api/v1/auth/sessions
func (h *SessionHandler) RevokeAll(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	if err := h.sessionRepo.DeleteAllUserSessions(c.Context(), userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "all sessions revoked"})
}
