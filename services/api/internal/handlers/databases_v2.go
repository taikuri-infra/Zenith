package handlers

import (
	"context"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// DatabaseHandlerV2 manages per-app database provisioning (Phase 3 deploy engine).
type DatabaseHandlerV2 struct {
	dbSvc   *services.DatabaseService // nil when CNPG not configured (dev mode)
	dbRepo  ports.DatabaseRepository
	appRepo ports.AppRepository
}

// NewDatabaseHandlerV2 creates a new DatabaseHandlerV2.
func NewDatabaseHandlerV2(dbSvc *services.DatabaseService, dbRepo ports.DatabaseRepository, appRepo ports.AppRepository) *DatabaseHandlerV2 {
	return &DatabaseHandlerV2{dbSvc: dbSvc, dbRepo: dbRepo, appRepo: appRepo}
}

// fetchCredentials retrieves the password and connection string for a database.
func (h *DatabaseHandlerV2) fetchCredentials(ctx context.Context, db *entities.UserDatabase) (string, string) {
	if h.dbSvc != nil {
		if pw, err := h.dbSvc.GetDatabasePassword(ctx, db.ID); err == nil {
			return pw, db.ConnectionString(pw)
		}
	} else if memRepo, ok := h.dbRepo.(*memory.MemoryDatabaseRepository); ok {
		if pw, ok := memRepo.GetPassword(db.ID); ok {
			return pw, db.ConnectionString(pw)
		}
	}
	return "", ""
}

// Create provisions a new database for an app.
// POST /api/v1/apps/:appId/databases
func (h *DatabaseHandlerV2) Create(c *fiber.Ctx) error {
	appID := c.Params("appId")
	userID, _ := c.Locals("user_id").(string)

	// Verify app exists and belongs to user
	app, err := h.appRepo.GetApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "app not found")
	}
	if app.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your app")
	}

	var input dto.CreateDatabaseInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if input.Engine == "" {
		input.Engine = entities.DatabaseEnginePostgres
	}

	// Validate engine
	switch input.Engine {
	case entities.DatabaseEnginePostgres, entities.DatabaseEngineMySQL, entities.DatabaseEngineRedis,
		entities.DatabaseEngineMongoDB, entities.DatabaseEngineRabbitMQ,
		entities.DatabaseEngineKafka:
		// valid
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unsupported engine: "+string(input.Engine))
	}

	// Use DatabaseService for real provisioning if available
	if h.dbSvc != nil {
		db, err := h.dbSvc.ProvisionDatabase(c.Context(), appID, userID, &input)
		if err != nil {
			if strings.Contains(err.Error(), "not available on the") {
				return fiber.NewError(fiber.StatusForbidden, err.Error())
			}
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
				return fiber.NewError(fiber.StatusConflict, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		pw, connStr := h.fetchCredentials(c.Context(), db)
		return c.Status(fiber.StatusCreated).JSON(toDatabaseInfoV2(db, pw, connStr))
	}

	// Fallback: metadata-only (dev mode, no CNPG)
	db, err := h.dbRepo.CreateDatabase(c.Context(), appID, userID, &input)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
			return fiber.NewError(fiber.StatusConflict, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Auto-inject connection string from memory repo (dev mode)
	pw, connStr := h.fetchCredentials(c.Context(), db)
	if connStr != "" {
		envKey := envKeyForEngine(db.Engine)
		h.appRepo.SetEnvVars(c.Context(), appID, map[string]string{envKey: connStr})
	}

	return c.Status(fiber.StatusCreated).JSON(toDatabaseInfoV2(db, pw, connStr))
}

// List returns all databases for an app.
// GET /api/v1/apps/:appId/databases
func (h *DatabaseHandlerV2) List(c *fiber.Ctx) error {
	appID := c.Params("appId")

	dbs, err := h.dbRepo.ListDatabasesByApp(c.Context(), appID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]dto.DatabaseInfo, len(dbs))
	for i, db := range dbs {
		result[i] = toDatabaseInfoV2(&db, "", "")
	}
	return c.JSON(result)
}

// Get returns a single database with connection string.
// GET /api/v1/apps/:appId/databases/:dbId
func (h *DatabaseHandlerV2) Get(c *fiber.Ctx) error {
	dbID := c.Params("dbId")

	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}

	password, connStr := h.fetchCredentials(c.Context(), db)
	return c.JSON(toDatabaseInfoV2(db, password, connStr))
}

// Delete deprovisions a database.
// DELETE /api/v1/apps/:appId/databases/:dbId
func (h *DatabaseHandlerV2) Delete(c *fiber.Ctx) error {
	dbID := c.Params("dbId")
	userID, _ := c.Locals("user_id").(string)

	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	// Use DatabaseService for real cleanup if available
	if h.dbSvc != nil {
		if err := h.dbSvc.DeleteDatabase(c.Context(), dbID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"message": "database deleted"})
	}

	// Fallback: metadata-only delete (dev mode)
	if err := h.dbRepo.DeleteDatabase(c.Context(), dbID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Remove auto-injected env var
	envKey := envKeyForEngine(db.Engine)
	h.appRepo.DeleteEnvVar(c.Context(), db.AppID, envKey)

	return c.JSON(fiber.Map{"message": "database deleted"})
}

// CreateStandalone provisions a standalone database (not tied to an app).
// POST /api/v1/databases
func (h *DatabaseHandlerV2) CreateStandalone(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var input dto.CreateDatabaseInput
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if input.Engine == "" {
		input.Engine = entities.DatabaseEnginePostgres
	}

	switch input.Engine {
	case entities.DatabaseEnginePostgres, entities.DatabaseEngineMySQL, entities.DatabaseEngineRedis:
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unsupported engine: "+string(input.Engine))
	}

	// Use DatabaseService for real provisioning if available
	if h.dbSvc != nil {
		db, err := h.dbSvc.ProvisionDatabase(c.Context(), "", userID, &input)
		if err != nil {
			if strings.Contains(err.Error(), "not available on the") {
				return fiber.NewError(fiber.StatusForbidden, err.Error())
			}
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
				return fiber.NewError(fiber.StatusConflict, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		pw, connStr := h.fetchCredentials(c.Context(), db)
		return c.Status(fiber.StatusCreated).JSON(toDatabaseInfoV2(db, pw, connStr))
	}

	// Fallback: metadata-only (dev mode)
	db, err := h.dbRepo.CreateDatabase(c.Context(), "", userID, &input)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
			return fiber.NewError(fiber.StatusConflict, err.Error())
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	pw, connStr := h.fetchCredentials(c.Context(), db)
	return c.Status(fiber.StatusCreated).JSON(toDatabaseInfoV2(db, pw, connStr))
}

// GetStandalone returns a single standalone database with connection string.
// GET /api/v1/databases/:dbId
func (h *DatabaseHandlerV2) GetStandalone(c *fiber.Ctx) error {
	dbID := c.Params("dbId")
	userID, _ := c.Locals("user_id").(string)

	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	password, connStr := h.fetchCredentials(c.Context(), db)
	return c.JSON(toDatabaseInfoV2(db, password, connStr))
}

// DeleteStandalone deprovisions a standalone database.
// DELETE /api/v1/databases/:dbId
func (h *DatabaseHandlerV2) DeleteStandalone(c *fiber.Ctx) error {
	dbID := c.Params("dbId")
	userID, _ := c.Locals("user_id").(string)

	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	if h.dbSvc != nil {
		if err := h.dbSvc.DeleteDatabase(c.Context(), dbID); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"message": "database deleted"})
	}

	if err := h.dbRepo.DeleteDatabase(c.Context(), dbID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"message": "database deleted"})
}

// ListByUser returns all databases for the authenticated user.
// GET /api/v1/databases
func (h *DatabaseHandlerV2) ListByUser(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var dbs []entities.UserDatabase
	var err error
	projectID := c.Query("project_id")
	if projectID != "" {
		dbs, err = h.dbRepo.ListDatabasesByProject(c.Context(), projectID)
	} else {
		dbs, err = h.dbRepo.ListDatabasesByUser(c.Context(), userID)
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	result := make([]dto.DatabaseInfo, len(dbs))
	for i, db := range dbs {
		result[i] = toDatabaseInfoV2(&db, "", "")
	}
	return c.JSON(result)
}

// ResetPassword generates a new password for a database.
// POST /api/v1/apps/:appId/databases/:dbId/reset-password
// POST /api/v1/databases/:dbId/reset-password
func (h *DatabaseHandlerV2) ResetPassword(c *fiber.Ctx) error {
	dbID := c.Params("dbId")
	userID, _ := c.Locals("user_id").(string)

	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	if h.dbSvc == nil {
		return fiber.NewError(fiber.StatusNotImplemented, "database service not configured")
	}

	newPassword, connStr, err := h.dbSvc.ResetDatabasePassword(c.Context(), dbID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"db_password":       newPassword,
		"connection_string": connStr,
	})
}

func envKeyForEngine(engine entities.DatabaseEngine) string {
	switch engine {
	case entities.DatabaseEngineRedis:
		return "REDIS_URL"
	case entities.DatabaseEngineMySQL:
		return "MYSQL_URL"
	case entities.DatabaseEngineMongoDB:
		return "MONGODB_URL"
	case entities.DatabaseEngineRabbitMQ:
		return "RABBITMQ_URL"
	case entities.DatabaseEngineKafka:
		return "KAFKA_BROKERS"
	default:
		return "DATABASE_URL"
	}
}

func toDatabaseInfoV2(db *entities.UserDatabase, password, connStr string) dto.DatabaseInfo {
	return dto.DatabaseInfo{
		ID:               db.ID,
		AppID:            db.AppID,
		Name:             db.Name,
		Engine:           db.Engine,
		Host:             db.Host,
		Port:             db.Port,
		DBName:           db.DBName,
		DBUser:           db.DBUser,
		Password:         password,
		ConnectionString: connStr,
		SizeMB:           db.SizeMB,
		MaxSizeMB:        db.MaxSizeMB,
		Status:           db.Status,
		CreatedAt:        db.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
