package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminRBACHandler manages admin user roles and permissions.
type AdminRBACHandler struct {
	pool     *pgxpool.Pool
	userRepo ports.UserRepository
}

// NewAdminRBACHandler creates a new AdminRBACHandler.
func NewAdminRBACHandler(pool *pgxpool.Pool, userRepo ports.UserRepository) *AdminRBACHandler {
	return &AdminRBACHandler{pool: pool, userRepo: userRepo}
}

// ListAdminUsers returns all users with admin roles.
// GET /api/v1/admin/admin-users
func (h *AdminRBACHandler) ListAdminUsers(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.JSON([]entities.AdminRole{})
	}

	rows, err := h.pool.Query(c.Context(),
		`SELECT ar.id, ar.user_id, u.email, u.name, ar.admin_role, ar.permissions,
		        ar.granted_by, ar.created_at, ar.updated_at
		 FROM admin_roles ar
		 JOIN users u ON u.id = ar.user_id
		 ORDER BY ar.created_at`,
	)
	if err != nil {
		return c.JSON([]entities.AdminRole{})
	}
	defer rows.Close()

	var roles []entities.AdminRole
	for rows.Next() {
		var r entities.AdminRole
		var grantedBy *string
		if err := rows.Scan(&r.ID, &r.UserID, &r.Email, &r.Name, &r.AdminRole,
			&r.Permissions, &grantedBy, &r.CreatedAt, &r.UpdatedAt); err == nil {
			if grantedBy != nil {
				r.GrantedBy = *grantedBy
			}
			roles = append(roles, r)
		}
	}

	return c.JSON(roles)
}

// InviteAdminUser creates an admin role for an existing user.
// POST /api/v1/admin/admin-users
func (h *AdminRBACHandler) InviteAdminUser(c *fiber.Ctx) error {
	var input struct {
		Email     string `json:"email"`
		AdminRole string `json:"adminRole"`
	}
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}
	if input.Email == "" {
		return NewBadRequest("email is required")
	}
	if input.AdminRole == "" {
		input.AdminRole = "viewer"
	}

	// Validate role
	validRoles := map[string]bool{"owner": true, "admin": true, "support": true, "viewer": true}
	if !validRoles[input.AdminRole] {
		return NewBadRequest("invalid admin role")
	}

	// Find user
	user, err := h.userRepo.GetByEmail(c.Context(), input.Email)
	if err != nil {
		return NewNotFound("user")
	}

	granterID, _ := c.Locals("user_id").(string)
	permissions := entities.DefaultAdminPermissions(input.AdminRole)

	if h.pool != nil {
		var id string
		err := h.pool.QueryRow(c.Context(),
			`INSERT INTO admin_roles (user_id, admin_role, permissions, granted_by)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (user_id) DO UPDATE SET admin_role = $2, permissions = $3, updated_at = now()
			 RETURNING id`,
			user.ID, input.AdminRole, permissions, granterID,
		).Scan(&id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to assign admin role")
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":     "admin role assigned",
		"email":       input.Email,
		"adminRole":   input.AdminRole,
		"permissions": permissions,
	})
}

// UpdateAdminRole updates an admin user's role.
// PUT /api/v1/admin/admin-users/:id/role
func (h *AdminRBACHandler) UpdateAdminRole(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("admin user id is required")
	}

	var input struct {
		AdminRole string `json:"adminRole"`
	}
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	validRoles := map[string]bool{"owner": true, "admin": true, "support": true, "viewer": true}
	if !validRoles[input.AdminRole] {
		return NewBadRequest("invalid admin role")
	}

	permissions := entities.DefaultAdminPermissions(input.AdminRole)

	if h.pool != nil {
		_, err := h.pool.Exec(c.Context(),
			`UPDATE admin_roles SET admin_role = $1, permissions = $2, updated_at = now() WHERE id = $3`,
			input.AdminRole, permissions, id,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to update role")
		}
	}

	return c.JSON(fiber.Map{"message": "role updated", "adminRole": input.AdminRole})
}

// RemoveAdminUser removes admin access for a user.
// DELETE /api/v1/admin/admin-users/:id
func (h *AdminRBACHandler) RemoveAdminUser(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("admin user id is required")
	}

	if h.pool != nil {
		_, err := h.pool.Exec(c.Context(),
			"DELETE FROM admin_roles WHERE id = $1", id,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to remove admin role")
		}
	}

	return c.JSON(fiber.Map{"message": "admin role removed"})
}
