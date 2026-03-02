package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// BackupHandlerV2 manages database backups (Phase 3).
type BackupHandlerV2 struct {
	backupRepo ports.BackupRepository
	dbRepo     ports.DatabaseRepository
	planRepo   ports.UserPlanRepository
}

// NewBackupHandlerV2 creates a new BackupHandlerV2.
func NewBackupHandlerV2(backupRepo ports.BackupRepository, dbRepo ports.DatabaseRepository, planRepo ports.UserPlanRepository) *BackupHandlerV2 {
	return &BackupHandlerV2{backupRepo: backupRepo, dbRepo: dbRepo, planRepo: planRepo}
}

// Create initiates a new backup for a database.
// POST /api/v1/apps/:appId/databases/:dbId/backups
func (h *BackupHandlerV2) Create(c *fiber.Ctx) error {
	dbID := c.Params("dbId")
	userID, _ := c.Locals("user_id").(string)

	// Check plan: backups require Pro+ tier
	if h.planRepo != nil {
		plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
		if err != nil || !plan.Limits.BackupsEnabled {
			return fiber.NewError(fiber.StatusForbidden, "backups require Pro plan or higher. Upgrade your plan.")
		}
	}

	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	var input dto.CreateBackupInput
	if err := c.BodyParser(&input); err != nil {
		input.Type = entities.BackupTypeManual
	}
	if input.Type == "" {
		input.Type = entities.BackupTypeManual
	}

	backup, err := h.backupRepo.CreateBackup(c.Context(), dbID, userID, input.Type)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(toBackupInfo(backup))
}

// List returns all backups for a database.
// GET /api/v1/apps/:appId/databases/:dbId/backups
func (h *BackupHandlerV2) List(c *fiber.Ctx) error {
	dbID := c.Params("dbId")

	backups, err := h.backupRepo.ListBackupsByDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]dto.BackupInfo, len(backups))
	for i, b := range backups {
		result[i] = toBackupInfo(&b)
	}
	return c.JSON(result)
}

// Get returns a single backup.
// GET /api/v1/apps/:appId/databases/:dbId/backups/:backupId
func (h *BackupHandlerV2) Get(c *fiber.Ctx) error {
	backupID := c.Params("backupId")

	backup, err := h.backupRepo.GetBackup(c.Context(), backupID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "backup not found")
	}

	return c.JSON(toBackupInfo(backup))
}

// Delete removes a backup.
// DELETE /api/v1/apps/:appId/databases/:dbId/backups/:backupId
func (h *BackupHandlerV2) Delete(c *fiber.Ctx) error {
	backupID := c.Params("backupId")
	userID, _ := c.Locals("user_id").(string)

	backup, err := h.backupRepo.GetBackup(c.Context(), backupID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "backup not found")
	}
	if backup.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your backup")
	}

	if err := h.backupRepo.DeleteBackup(c.Context(), backupID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "backup deleted"})
}

// Restore initiates a database restore from a backup.
// POST /api/v1/apps/:appId/databases/:dbId/backups/:backupId/restore
func (h *BackupHandlerV2) Restore(c *fiber.Ctx) error {
	backupID := c.Params("backupId")
	dbID := c.Params("dbId")
	userID, _ := c.Locals("user_id").(string)

	backup, err := h.backupRepo.GetBackup(c.Context(), backupID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "backup not found")
	}
	if backup.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your backup")
	}
	if backup.Status != entities.BackupStatusCompleted {
		return fiber.NewError(fiber.StatusBadRequest, "backup is not in completed state")
	}

	// Verify database still exists
	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	// In a real implementation, this would trigger a pg_restore job.
	// For now, we simulate by marking the database as provisioning briefly.
	h.dbRepo.UpdateDatabaseStatus(c.Context(), dbID, entities.DatabaseStatusProvisioning)

	// Simulate restore completion asynchronously
	go func() {
		// In production: run pg_restore from S3 backup
		h.dbRepo.UpdateDatabaseStatus(c.Context(), dbID, entities.DatabaseStatusReady)
	}()

	return c.JSON(fiber.Map{
		"message":     "restore initiated",
		"backup_id":   backupID,
		"database_id": dbID,
	})
}

// ListByUser returns all backups for the authenticated user.
// GET /api/v1/backups
func (h *BackupHandlerV2) ListByUser(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	backups, err := h.backupRepo.ListBackupsByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]dto.BackupInfo, len(backups))
	for i, b := range backups {
		result[i] = toBackupInfo(&b)
	}
	return c.JSON(result)
}

func toBackupInfo(b *entities.DatabaseBackup) dto.BackupInfo {
	return dto.BackupInfo{
		ID:         b.ID,
		DatabaseID: b.DatabaseID,
		Type:       b.Type,
		Status:     b.Status,
		SizeMB:     b.SizeMB,
		Error:      b.Error,
		CreatedAt:  b.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
