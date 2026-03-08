package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// UserAuditHandler handles user-scoped audit log endpoints (Business+).
type UserAuditHandler struct {
	adminRepo ports.AdminRepository
	planRepo  ports.UserPlanRepository
}

// NewUserAuditHandler creates a new UserAuditHandler.
func NewUserAuditHandler(adminRepo ports.AdminRepository, planRepo ports.UserPlanRepository) *UserAuditHandler {
	return &UserAuditHandler{adminRepo: adminRepo, planRepo: planRepo}
}

func (h *UserAuditHandler) requireBusinessPlus(c *fiber.Ctx) (string, error) {
	userID, _ := c.Locals("user_id").(string)
	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return "", fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier != entities.PlanBusiness && plan.Tier != entities.PlanEnterprise {
		return "", fiber.NewError(fiber.StatusForbidden, "audit log requires Business plan or higher")
	}
	return userID, nil
}

// List returns audit log entries filtered to the current user.
// GET /api/v1/audit?limit=50&offset=0&action=deploy
func (h *UserAuditHandler) List(c *fiber.Ctx) error {
	userID, err := h.requireBusinessPlus(c)
	if err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	actionFilter := c.Query("action")
	search := c.Query("search")

	if limit > 1000 {
		limit = 1000
	}

	// Fetch a larger window and filter by actor
	entries, err := h.adminRepo.ListAuditLog(c.Context(), limit*10, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list audit log")
	}

	var filtered []entities.AuditEntry
	for _, entry := range entries {
		if entry.Actor != userID {
			continue
		}
		if actionFilter != "" && entry.Action != actionFilter {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(entry.Action), strings.ToLower(search)) {
			continue
		}
		filtered = append(filtered, entry)
	}

	// Apply offset/limit
	total := len(filtered)
	if offset >= total {
		filtered = []entities.AuditEntry{}
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		filtered = filtered[offset:end]
	}

	return c.JSON(fiber.Map{"items": filtered, "total": total})
}

// ExportCSV exports the user's audit log entries as CSV.
// GET /api/v1/audit/export/csv?action=deploy&limit=5000
func (h *UserAuditHandler) ExportCSV(c *fiber.Ctx) error {
	userID, err := h.requireBusinessPlus(c)
	if err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.Query("limit", "5000"))
	if limit > 10000 {
		limit = 10000
	}
	actionFilter := c.Query("action")

	entries, err := h.adminRepo.ListAuditLog(c.Context(), limit*10, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list audit log")
	}

	var csv strings.Builder
	csv.WriteString("time,actor,action,cluster\n")

	for _, entry := range entries {
		if entry.Actor != userID {
			continue
		}
		if actionFilter != "" && entry.Action != actionFilter {
			continue
		}
		csv.WriteString(fmt.Sprintf("%s,%s,%s,%s\n",
			escapeCSV(entry.Time),
			escapeCSV(entry.Actor),
			escapeCSV(entry.Action),
			escapeCSV(entry.Cluster),
		))
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=audit-log-%s.csv", time.Now().Format("2006-01-02")))
	return c.SendString(csv.String())
}

// ExportJSON exports the user's audit log entries as JSON.
// GET /api/v1/audit/export/json?action=deploy&limit=5000
func (h *UserAuditHandler) ExportJSON(c *fiber.Ctx) error {
	userID, err := h.requireBusinessPlus(c)
	if err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.Query("limit", "5000"))
	if limit > 10000 {
		limit = 10000
	}
	actionFilter := c.Query("action")

	entries, err := h.adminRepo.ListAuditLog(c.Context(), limit*10, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list audit log")
	}

	var filtered []entities.AuditEntry
	for _, entry := range entries {
		if entry.Actor != userID {
			continue
		}
		if actionFilter != "" && entry.Action != actionFilter {
			continue
		}
		filtered = append(filtered, entry)
	}

	return c.JSON(fiber.Map{"items": filtered, "total": len(filtered)})
}
