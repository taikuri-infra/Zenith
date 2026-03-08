package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

type RoleHandler struct {
	roleRepo ports.RoleRepository
	planRepo ports.UserPlanRepository
}

func NewRoleHandler(roleRepo ports.RoleRepository, planRepo ports.UserPlanRepository) *RoleHandler {
	return &RoleHandler{roleRepo: roleRepo, planRepo: planRepo}
}

func (h *RoleHandler) Create(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier != entities.PlanTeam && plan.Tier != entities.PlanBusiness && plan.Tier != entities.PlanEnterprise {
		return fiber.NewError(fiber.StatusForbidden, "custom roles require Team plan or higher")
	}

	var body struct {
		Name        string                `json:"name"`
		Description string                `json:"description"`
		Permissions []entities.Permission `json:"permissions"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if len(body.Permissions) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "at least one permission is required")
	}

	role, err := h.roleRepo.CreateRole(c.Context(), userID, body.Name, body.Description, body.Permissions)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(role)
}

func (h *RoleHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	roles, err := h.roleRepo.ListRolesByUser(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if roles == nil {
		roles = []entities.CustomRole{}
	}
	return c.JSON(fiber.Map{"items": roles})
}

func (h *RoleHandler) Update(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	roleID := c.Params("roleId")

	role, err := h.roleRepo.GetRole(c.Context(), roleID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "role not found")
	}
	if role.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your role")
	}

	var body struct {
		Name        *string               `json:"name,omitempty"`
		Description *string               `json:"description,omitempty"`
		Permissions []entities.Permission  `json:"permissions,omitempty"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	updated, err := h.roleRepo.UpdateRole(c.Context(), roleID, body.Name, body.Description, body.Permissions)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(updated)
}

func (h *RoleHandler) Delete(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	roleID := c.Params("roleId")

	role, err := h.roleRepo.GetRole(c.Context(), roleID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "role not found")
	}
	if role.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your role")
	}

	if err := h.roleRepo.DeleteRole(c.Context(), roleID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *RoleHandler) AssignRole(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	roleID := c.Params("roleId")

	role, err := h.roleRepo.GetRole(c.Context(), roleID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "role not found")
	}
	if role.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your role")
	}

	var body struct {
		MemberID string `json:"member_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.MemberID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "member_id is required")
	}

	assignment, err := h.roleRepo.AssignRole(c.Context(), roleID, body.MemberID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(assignment)
}

func (h *RoleHandler) ListAssignments(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	roleID := c.Params("roleId")

	// Verify the role belongs to the authenticated user
	role, err := h.roleRepo.GetRole(c.Context(), roleID)
	if err != nil || role.UserID != userID {
		return fiber.NewError(fiber.StatusNotFound, "role not found")
	}

	assignments, err := h.roleRepo.ListAssignmentsByRole(c.Context(), roleID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if assignments == nil {
		assignments = []entities.RoleAssignment{}
	}
	return c.JSON(fiber.Map{"items": assignments})
}

func (h *RoleHandler) RemoveAssignment(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	roleID := c.Params("roleId")
	assignmentID := c.Params("assignmentId")

	// Verify the role belongs to the authenticated user
	role, err := h.roleRepo.GetRole(c.Context(), roleID)
	if err != nil || role.UserID != userID {
		return fiber.NewError(fiber.StatusNotFound, "role not found")
	}

	if err := h.roleRepo.RemoveAssignment(c.Context(), assignmentID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "assignment not found")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *RoleHandler) ListPermissions(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"permissions": entities.AllPermissions()})
}
