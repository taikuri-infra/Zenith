package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// AuditExportHandler handles audit log export endpoints.
type AuditExportHandler struct {
	adminRepo ports.AdminRepository
}

// NewAuditExportHandler creates a new AuditExportHandler.
func NewAuditExportHandler(adminRepo ports.AdminRepository) *AuditExportHandler {
	return &AuditExportHandler{adminRepo: adminRepo}
}

// ExportCSV exports audit log entries as CSV.
// Query params: action (filter by action substring), limit (default 1000, max 10000)
// GET /api/v1/admin/audit/export/csv
func (h *AuditExportHandler) ExportCSV(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 1000)
	if limit > 10000 {
		limit = 10000
	}

	actionFilter := c.Query("action")

	entries, err := h.adminRepo.ListAuditLog(c.Context(), limit, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list audit log")
	}

	var csv strings.Builder
	csv.WriteString("time,actor,action,cluster\n")

	for _, entry := range entries {
		// Action filter
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

// ExportJSON exports audit log entries as JSON (alternative format).
// Query params: action (filter by action substring), limit (default 1000, max 10000)
// GET /api/v1/admin/audit/export/json
func (h *AuditExportHandler) ExportJSON(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 1000)
	if limit > 10000 {
		limit = 10000
	}

	actionFilter := c.Query("action")

	entries, err := h.adminRepo.ListAuditLog(c.Context(), limit, 0)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list audit log")
	}

	if actionFilter != "" {
		var filtered []interface{}
		for _, entry := range entries {
			if entry.Action == actionFilter {
				filtered = append(filtered, entry)
			}
		}
		return c.JSON(fiber.Map{"items": filtered, "total": len(filtered)})
	}

	return c.JSON(fiber.Map{"items": entries, "total": len(entries)})
}

// escapeCSV wraps a field in quotes if it contains commas, quotes, or newlines.
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
