package handlers

import (
	"strconv"
	"strings"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// MeteringHandler serves metering and usage endpoints.
type MeteringHandler struct {
	metering  ports.MeteringRepository
	customers ports.CustomerRepository
}

// NewMeteringHandler creates a new MeteringHandler.
func NewMeteringHandler(metering ports.MeteringRepository, customers ports.CustomerRepository) *MeteringHandler {
	return &MeteringHandler{
		metering:  metering,
		customers: customers,
	}
}

// RecordUsage records a resource usage snapshot (internal endpoint).
// POST /api/v1/internal/metering
func (h *MeteringHandler) RecordUsage(c *fiber.Ctx) error {
	var input dto.MeteringInput
	if err := c.BodyParser(&input); err != nil {
		return NewBadRequest("invalid request body")
	}

	if input.CustomerID == "" {
		return NewBadRequest("customerId is required")
	}

	// Verify customer exists
	if _, err := h.customers.GetCustomer(c.Context(), input.CustomerID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewBadRequest("customer not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to verify customer")
	}

	entry, err := h.metering.RecordUsage(c.Context(), &input)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to record usage")
	}

	return c.Status(fiber.StatusCreated).JSON(entry)
}

// safePercent computes (value / ceiling) * 100, returning 0 when ceiling is 0.
func safePercent(value float64, ceiling int) float64 {
	if ceiling == 0 {
		return 0
	}
	p := (value / float64(ceiling)) * 100
	// Round to 1 decimal place
	return float64(int(p*10)) / 10
}

// GetCustomerUsage returns the latest usage with plan ceilings and percentages.
// GET /api/v1/admin/customers/:id/usage
func (h *MeteringHandler) GetCustomerUsage(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	customer, err := h.customers.GetCustomer(c.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get customer")
	}

	plan := customer.Plan
	if plan == nil {
		return fiber.NewError(fiber.StatusInternalServerError, "customer has no plan")
	}

	latest, err := h.metering.GetLatestUsage(c.Context(), id)
	if err != nil {
		// No usage data yet — return zero usage with ceilings
		return c.JSON(dto.CustomerUsage{
			CPUCeiling: plan.CPUCores,
			RAMCeiling: plan.RAMGB,
			S3Ceiling:  plan.S3TB,
			DBCeiling:  plan.DBStorageGB,
			VolCeiling: plan.VolumeGB,
			LBCeiling:  plan.LBCount,
		})
	}

	usage := dto.CustomerUsage{
		CPUCores:    latest.CPUCores,
		CPUCeiling:  plan.CPUCores,
		CPUPercent:  safePercent(latest.CPUCores, plan.CPUCores),
		RAMGB:       latest.RAMGB,
		RAMCeiling:  plan.RAMGB,
		RAMPercent:  safePercent(latest.RAMGB, plan.RAMGB),
		S3TB:        latest.S3TB,
		S3Ceiling:   plan.S3TB,
		S3Percent:   safePercent(latest.S3TB, plan.S3TB),
		DBStorageGB: latest.DBStorageGB,
		DBCeiling:   plan.DBStorageGB,
		DBPercent:   safePercent(latest.DBStorageGB, plan.DBStorageGB),
		VolumeGB:    latest.VolumeGB,
		VolCeiling:  plan.VolumeGB,
		VolPercent:  safePercent(latest.VolumeGB, plan.VolumeGB),
		LBCount:     latest.LBCount,
		LBCeiling:   plan.LBCount,
		LBPercent:   safePercent(float64(latest.LBCount), plan.LBCount),
		RecordedAt:  latest.RecordedAt,
	}

	return c.JSON(usage)
}

// GetCustomerUsageHistory returns daily aggregated usage history.
// GET /api/v1/admin/customers/:id/usage/history?days=30
func (h *MeteringHandler) GetCustomerUsageHistory(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return NewBadRequest("customer id is required")
	}

	// Verify customer exists
	if _, err := h.customers.GetCustomer(c.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return NewNotFound("customer")
		}
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get customer")
	}

	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	history, err := h.metering.GetUsageHistory(c.Context(), id, days)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get usage history")
	}

	return c.JSON(history)
}

// GetPlatformUsageSummary returns aggregate usage across all customers.
// GET /api/v1/admin/dashboard/usage
func (h *MeteringHandler) GetPlatformUsageSummary(c *fiber.Ctx) error {
	summary, err := h.metering.GetPlatformUsageSummary(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to get platform usage summary")
	}

	return c.JSON(summary)
}
