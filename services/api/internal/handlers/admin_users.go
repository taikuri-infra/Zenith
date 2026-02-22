package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// AdminUserHandler manages platform users from the admin panel (Phase 4).
type AdminUserHandler struct {
	userRepo    ports.UserRepository
	planRepo    ports.UserPlanRepository
	appRepo     ports.AppRepository
	dbRepo      ports.DatabaseRepository
	storageRepo ports.StorageRepository
}

// NewAdminUserHandler creates a new AdminUserHandler.
func NewAdminUserHandler(
	userRepo ports.UserRepository,
	planRepo ports.UserPlanRepository,
	appRepo ports.AppRepository,
	dbRepo ports.DatabaseRepository,
	storageRepo ports.StorageRepository,
) *AdminUserHandler {
	return &AdminUserHandler{
		userRepo:    userRepo,
		planRepo:    planRepo,
		appRepo:     appRepo,
		dbRepo:      dbRepo,
		storageRepo: storageRepo,
	}
}

// AdminUserInfo is the response for an admin-visible user.
type AdminUserInfo struct {
	ID    string            `json:"id"`
	Email string            `json:"email"`
	Name  string            `json:"name"`
	Role  entities.Role     `json:"role"`
	Tier  entities.PlanTier `json:"tier"`
	Usage dto.PlanUsage     `json:"usage"`
}

// GetUser returns detailed info about a specific user.
// GET /api/v1/admin/users/:userId
func (h *AdminUserHandler) GetUser(c *fiber.Ctx) error {
	userID := c.Params("userId")

	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	plan, _ := h.planRepo.GetUserPlan(c.Context(), userID)
	appCount, _ := h.appRepo.CountAppsByUser(c.Context(), userID)
	dbCount, _ := h.dbRepo.CountDatabasesByUser(c.Context(), userID)
	bucketCount, _ := h.storageRepo.CountBucketsByUser(c.Context(), userID)

	tier := entities.PlanFree
	if plan != nil {
		tier = plan.Tier
	}

	return c.JSON(AdminUserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
		Tier:  tier,
		Usage: dto.PlanUsage{
			Apps:      appCount,
			Databases: dbCount,
			Buckets:   bucketCount,
		},
	})
}

// SetUserPlan overrides a user's plan tier (admin only).
// POST /api/v1/admin/users/:userId/plan
func (h *AdminUserHandler) SetUserPlan(c *fiber.Ctx) error {
	userID := c.Params("userId")

	var input struct {
		Tier entities.PlanTier `json:"tier"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	_, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	plan, err := h.planRepo.SetUserPlan(c.Context(), userID, input.Tier)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"user_id": userID,
		"tier":    plan.Tier,
		"limits":  plan.Limits,
	})
}

// ListUserApps returns all apps for a specific user.
// GET /api/v1/admin/users/:userId/apps
func (h *AdminUserHandler) ListUserApps(c *fiber.Ctx) error {
	userID := c.Params("userId")

	apps, err := h.appRepo.ListAppsByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"items": apps, "total": len(apps)})
}

// ListUserDatabases returns all databases for a specific user.
// GET /api/v1/admin/users/:userId/databases
func (h *AdminUserHandler) ListUserDatabases(c *fiber.Ctx) error {
	userID := c.Params("userId")

	dbs, err := h.dbRepo.ListDatabasesByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"items": dbs, "total": len(dbs)})
}
