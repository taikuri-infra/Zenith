package handlers

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/adapters/memory"
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/dotechhq/zenith/services/api/internal/services/deploy"
	"github.com/gofiber/fiber/v2"
)

// DatabaseHandlerV2 manages per-app database provisioning (Phase 3 deploy engine).
type DatabaseHandlerV2 struct {
	dbSvc   *services.DatabaseService // nil when CNPG not configured (dev mode)
	dbRepo  ports.DatabaseRepository
	appRepo ports.AppRepository
	msRepo  ports.ManagedServiceRepository // merged into ListByUser results
	rt      *services.DBRuntimeService
	// redeployer restarts an app after a database is attached so the
	// injected connection string reaches the running container.
	redeployer AppRedeployer
	// managedEnv, when set, stores injected variables as env-v2 rows.
	managedEnv *services.ManagedEnv
}

// NewDatabaseHandlerV2 creates a new DatabaseHandlerV2.
func NewDatabaseHandlerV2(dbSvc *services.DatabaseService, dbRepo ports.DatabaseRepository, appRepo ports.AppRepository, msRepo ports.ManagedServiceRepository) *DatabaseHandlerV2 {
	return &DatabaseHandlerV2{dbSvc: dbSvc, dbRepo: dbRepo, appRepo: appRepo, msRepo: msRepo}
}

// SetManagedEnv routes injected connection variables to env-v2.
func (h *DatabaseHandlerV2) SetManagedEnv(m *services.ManagedEnv) { h.managedEnv = m }

func (h *DatabaseHandlerV2) setInjectedEnv(ctx context.Context, appID, key, value, dbID string) error {
	if h.managedEnv != nil {
		return h.managedEnv.Set(ctx, appID, key, value, dbID)
	}
	return h.appRepo.SetEnvVars(ctx, appID, map[string]string{key: value})
}

func (h *DatabaseHandlerV2) clearInjectedEnv(ctx context.Context, appID, key, dbID string) {
	if h.managedEnv != nil {
		_ = h.managedEnv.Clear(ctx, appID, dbID)
		return
	}
	_ = h.appRepo.DeleteEnvVar(ctx, appID, key)
}

// SetRuntime makes the handler provision real instances (Docker in standalone).
func (h *DatabaseHandlerV2) SetRuntime(rt *services.DBRuntimeService) { h.rt = rt }

// SetRedeployer makes attaching a database redeploy the app's current image.
func (h *DatabaseHandlerV2) SetRedeployer(r AppRedeployer) { h.redeployer = r }

// injectWhenReady writes the connection string into the app's env once the
// database is provisioned. Runs in the background.
func (h *DatabaseHandlerV2) injectWhenReady(appID string, db *entities.UserDatabase) {
	go func() {
		h.rt.Wait(db.ID)
		ctx := context.Background()
		fresh, err := h.dbRepo.GetDatabase(ctx, db.ID)
		if err != nil || fresh.Status != entities.DatabaseStatusReady {
			return
		}
		pw, err := h.rt.Password(ctx, fresh)
		if err != nil {
			slog.Warn("database runtime: reading password failed", "database", db.ID, "error", err)
			return
		}
		if err := h.setInjectedEnv(ctx, appID, envKeyForEngine(fresh.Engine), fresh.ConnectionString(pw), db.ID); err != nil {
			slog.Warn("database runtime: injecting connection string failed", "database", db.ID, "app", appID, "error", err)
			return
		}
		h.redeployApp(ctx, appID)
	}()
}

// redeployApp restarts the app on its current image so new env vars apply.
// An app that has never been deployed has nothing to restart.
func (h *DatabaseHandlerV2) redeployApp(ctx context.Context, appID string) {
	if h.redeployer == nil {
		return
	}
	imageTag, err := deploy.CurrentImage(ctx, h.appRepo, appID)
	if errors.Is(err, deploy.ErrNoImage) {
		slog.Info("database attached; app not deployed yet, skipping redeploy", "app", appID)
		return
	}
	if err != nil {
		slog.Warn("database attached; resolving current image failed", "app", appID, "error", err)
		return
	}
	app, err := h.appRepo.GetApp(ctx, appID)
	if err != nil {
		slog.Warn("database attached; loading app failed", "app", appID, "error", err)
		return
	}
	if err := h.redeployer.DeployApp(ctx, app, imageTag); err != nil {
		slog.Warn("database attached; redeploy failed", "app", appID, "error", err)
	}
}

// fetchCredentials retrieves the password and connection string for a database.
func (h *DatabaseHandlerV2) fetchCredentials(ctx context.Context, db *entities.UserDatabase) (string, string) {
	if h.rt != nil {
		if pw, err := h.rt.Password(ctx, db); err == nil {
			return pw, db.ConnectionString(pw)
		}
		return "", ""
	}
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

	if h.rt != nil {
		db, err := h.rt.Create(c.Context(), appID, userID, &input)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
				return fiber.NewError(fiber.StatusConflict, err.Error())
			}
			if strings.Contains(err.Error(), "not available") {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		h.injectWhenReady(appID, db)
		return c.Status(fiber.StatusCreated).JSON(toDatabaseInfoV2(db, "", ""))
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
		h.setInjectedEnv(c.Context(), appID, envKey, connStr, db.ID)
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
	userID, _ := c.Locals("user_id").(string)

	db, err := h.dbRepo.GetDatabase(c.Context(), dbID)
	if err != nil || db.AppID != c.Params("appId") || db.UserID != userID {
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
	if err != nil || db.AppID != c.Params("appId") {
		return fiber.NewError(fiber.StatusNotFound, "database not found")
	}
	if db.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your database")
	}

	if h.rt != nil {
		if err := h.rt.Delete(c.Context(), db, c.QueryBool("purge", false)); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		h.clearInjectedEnv(c.Context(), db.AppID, envKeyForEngine(db.Engine), db.ID)
		return c.JSON(fiber.Map{"message": "database deleted"})
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
	h.clearInjectedEnv(c.Context(), db.AppID, envKey, db.ID)

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
	case entities.DatabaseEnginePostgres, entities.DatabaseEngineMySQL, entities.DatabaseEngineRedis,
		entities.DatabaseEngineMongoDB, entities.DatabaseEngineRabbitMQ:
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unsupported engine: "+string(input.Engine))
	}

	if h.rt != nil {
		db, err := h.rt.Create(c.Context(), "", userID, &input)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "duplicate") {
				return fiber.NewError(fiber.StatusConflict, err.Error())
			}
			if strings.Contains(err.Error(), "not available") {
				return fiber.NewError(fiber.StatusBadRequest, err.Error())
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(toDatabaseInfoV2(db, "", ""))
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

	if h.rt != nil {
		if err := h.rt.Delete(c.Context(), db, c.QueryBool("purge", false)); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"message": "database deleted"})
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
// Merges both standalone UserDatabases and ManagedServices into a single list.
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

	// project_id is caller-supplied: keep only the caller's own rows so a
	// guessed or foreign project id discloses nothing.
	result := make([]dto.DatabaseInfo, 0, len(dbs))
	for i := range dbs {
		if dbs[i].UserID != userID {
			continue
		}
		result = append(result, toDatabaseInfoV2(&dbs[i], "", ""))
	}

	// Also include managed services (provisioned via compose import)
	if h.msRepo != nil {
		var msList []entities.ManagedService
		if projectID != "" {
			msList, _ = h.msRepo.ListManagedServicesByProject(c.Context(), projectID)
		} else {
			msList, _ = h.msRepo.ListManagedServicesByUser(c.Context(), userID)
		}
		// Build a set of existing IDs to avoid duplicates
		seen := make(map[string]bool, len(result))
		for _, r := range result {
			seen[r.ID] = true
		}
		for i := range msList {
			if msList[i].UserID == userID && !seen[msList[i].ID] {
				result = append(result, managedServiceToDatabaseInfo(&msList[i]))
			}
		}
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

func envKeyForEngine(engine entities.DatabaseEngine) string { return services.EnvKeyForEngine(engine) }

// managedServiceToDatabaseInfo converts a ManagedService to DatabaseInfo so both
// standalone databases and compose-provisioned services appear in the same list.
func managedServiceToDatabaseInfo(ms *entities.ManagedService) dto.DatabaseInfo {
	engine := entities.DatabaseEngine(ms.ServiceType) // ServiceType values match DatabaseEngine values
	status := entities.DatabaseStatus(ms.Status)      // same mapping: provisioning, ready, error, deleting
	return dto.DatabaseInfo{
		ID:        ms.ID,
		Name:      ms.Name,
		Engine:    engine,
		Host:      ms.InternalHost,
		Port:      ms.Port,
		DBName:    ms.DatabaseName,
		DBUser:    ms.Username,
		SizeMB:    ms.StorageGB * 1024,
		MaxSizeMB: ms.StorageGB * 1024,
		Status:    status,
		CreatedAt: ms.CreatedAt.Format("2006-01-02T15:04:05Z"),
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
		HostPort:         db.HostPort,
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
