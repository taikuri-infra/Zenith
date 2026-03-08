package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/services"
	"github.com/gofiber/fiber/v2"
)

// TeamMemberHandler manages team member HTTP endpoints.
type TeamMemberHandler struct {
	teamSvc *services.TeamMemberService
}

// NewTeamMemberHandler creates a new TeamMemberHandler.
func NewTeamMemberHandler(teamSvc *services.TeamMemberService) *TeamMemberHandler {
	return &TeamMemberHandler{teamSvc: teamSvc}
}

// InviteMember invites a new team member.
// POST /api/v1/team/invite
func (h *TeamMemberHandler) InviteMember(c *fiber.Ctx) error {
	accountID, _ := c.Locals("user_id").(string)

	// Only owners can invite
	role, _ := c.Locals("role").(entities.Role)
	if role != entities.RoleOwner {
		return fiber.NewError(fiber.StatusForbidden, "only account owners can invite members")
	}

	var input struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}
	if input.Role == "" {
		input.Role = "viewer"
	}

	member, err := h.teamSvc.InviteMember(c.Context(), accountID, input.Email, entities.Role(input.Role))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(member)
}

// ListMembers lists all team members for the account.
// GET /api/v1/team/members
func (h *TeamMemberHandler) ListMembers(c *fiber.Ctx) error {
	accountID, _ := c.Locals("user_id").(string)

	members, err := h.teamSvc.ListMembers(c.Context(), accountID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if members == nil {
		members = []entities.TeamMember{}
	}

	return c.JSON(fiber.Map{"items": members, "total": len(members)})
}

// UpdateRole changes a member's role.
// PUT /api/v1/team/members/:id/role
func (h *TeamMemberHandler) UpdateRole(c *fiber.Ctx) error {
	accountID, _ := c.Locals("user_id").(string)
	memberID := c.Params("id")

	role, _ := c.Locals("role").(entities.Role)
	if role != entities.RoleOwner {
		return fiber.NewError(fiber.StatusForbidden, "only account owners can change roles")
	}

	var input struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Role == "" {
		return fiber.NewError(fiber.StatusBadRequest, "role is required")
	}

	if err := h.teamSvc.UpdateMemberRole(c.Context(), accountID, memberID, entities.Role(input.Role)); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "role updated"})
}

// RemoveMember removes a team member.
// DELETE /api/v1/team/members/:id
func (h *TeamMemberHandler) RemoveMember(c *fiber.Ctx) error {
	accountID, _ := c.Locals("user_id").(string)
	memberID := c.Params("id")

	role, _ := c.Locals("role").(entities.Role)
	if role != entities.RoleOwner {
		return fiber.NewError(fiber.StatusForbidden, "only account owners can remove members")
	}

	if err := h.teamSvc.RemoveMember(c.Context(), accountID, memberID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"message": "member removed"})
}

// AcceptInvite accepts a team invitation (public endpoint, no auth required).
// POST /api/v1/team/accept-invite
func (h *TeamMemberHandler) AcceptInvite(c *fiber.Ctx) error {
	var input struct {
		Token    string `json:"token"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := c.BodyParser(&input); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if input.Token == "" {
		return fiber.NewError(fiber.StatusBadRequest, "token is required")
	}
	if input.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required")
	}
	if input.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "password is required")
	}

	tokens, err := h.teamSvc.AcceptInvite(c.Context(), input.Token, input.Email, input.Password, input.Name)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
	})
}
