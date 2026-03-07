package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// PgwebHandler handles pgweb database explorer endpoints.
type PgwebHandler struct {
	pgwebSvc *services.PgwebService
	dbRepo   ports.DatabaseRepository
}

// NewPgwebHandler creates a new PgwebHandler.
func NewPgwebHandler(pgwebSvc *services.PgwebService, dbRepo ports.DatabaseRepository) *PgwebHandler {
	return &PgwebHandler{pgwebSvc: pgwebSvc, dbRepo: dbRepo}
}

// Start creates a pgweb explorer session for a database.
// POST /api/v1/databases/:dbId/explorer
func (h *PgwebHandler) Start(c *fiber.Ctx) error {
	dbID := c.Params("dbId")
	userID, _ := c.Locals("user_id").(string)

	// Ownership check
	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	// Only PostgreSQL supported
	if db.Engine != "postgresql" {
		return fiber.NewError(fiber.StatusBadRequest, "explorer only supports PostgreSQL databases")
	}

	// Parse request body (optional)
	var body struct {
		ReadOnly *bool `json:"readonly"`
	}
	c.BodyParser(&body)

	readOnly := false // default: full access for owner
	if body.ReadOnly != nil {
		readOnly = *body.ReadOnly
	}

	session, err := h.pgwebSvc.StartSession(c.Context(), dbID, userID, readOnly)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"url":      session.URL,
		"status":   session.Status,
		"readonly": session.ReadOnly,
	})
}

// Status returns the current explorer session status.
// GET /api/v1/databases/:dbId/explorer
func (h *PgwebHandler) Status(c *fiber.Ctx) error {
	dbID := c.Params("dbId")
	userID, _ := c.Locals("user_id").(string)

	// Ownership check
	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	session, err := h.pgwebSvc.GetSession(c.Context(), dbID)
	if err != nil {
		return c.JSON(fiber.Map{
			"active": false,
		})
	}

	return c.JSON(fiber.Map{
		"active":   true,
		"url":      session.URL,
		"status":   session.Status,
		"readonly": session.ReadOnly,
	})
}

// Stop terminates an explorer session and cleans up K8s resources.
// DELETE /api/v1/databases/:dbId/explorer
func (h *PgwebHandler) Stop(c *fiber.Ctx) error {
	dbID := c.Params("dbId")
	userID, _ := c.Locals("user_id").(string)

	// Ownership check
	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	if err := h.pgwebSvc.StopSession(c.Context(), dbID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(fiber.Map{"message": "explorer session stopped"})
}
