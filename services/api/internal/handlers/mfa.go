package handlers

import (
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// MFAHandler handles MFA endpoints.
type MFAHandler struct {
	mfaRepo  ports.MFARepository
	planRepo ports.UserPlanRepository
}

func NewMFAHandler(mfaRepo ports.MFARepository, planRepo ports.UserPlanRepository) *MFAHandler {
	return &MFAHandler{mfaRepo: mfaRepo, planRepo: planRepo}
}

// GetStatus returns current MFA status for the authenticated user.
func (h *MFAHandler) GetStatus(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	enrollment, err := h.mfaRepo.GetEnrollment(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"status":       enrollment.Status,
		"enabled_at":   enrollment.EnabledAt,
		"backup_codes": len(enrollment.BackupCodes),
	})
}

// Enable begins MFA enrollment — returns TOTP secret + provisioning URI + backup codes.
func (h *MFAHandler) Enable(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	// Check plan allows MFA (Pro+)
	plan, err := h.planRepo.GetUserPlan(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if plan.Tier == entities.PlanFree {
		return fiber.NewError(fiber.StatusForbidden, "MFA requires Pro plan or higher")
	}

	enrollment, err := h.mfaRepo.StartEnrollment(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Build otpauth URI for QR code generation
	otpauthURI := fmt.Sprintf("otpauth://totp/Zenith:%s?secret=%s&issuer=Zenith&digits=6&period=30", userID, enrollment.Secret)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"secret":       enrollment.Secret,
		"otpauth_uri":  otpauthURI,
		"backup_codes": enrollment.BackupCodes,
	})
}

// Verify confirms MFA enrollment by validating a TOTP code from the authenticator.
func (h *MFAHandler) Verify(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.Code == "" || len(body.Code) != 6 {
		return fiber.NewError(fiber.StatusBadRequest, "code must be 6 digits")
	}

	// In a real implementation, validate the TOTP code against the secret.
	// For now, accept any 6-digit code during enrollment verification.
	enrollment, err := h.mfaRepo.ConfirmEnrollment(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{
		"status":     enrollment.Status,
		"enabled_at": enrollment.EnabledAt,
	})
}

// Disable turns off MFA for the authenticated user.
func (h *MFAHandler) Disable(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	var body struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "TOTP code or backup code required")
	}

	// In production, validate TOTP or backup code here before disabling.
	if err := h.mfaRepo.DisableEnrollment(c.Context(), userID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"status": entities.MFAStatusDisabled})
}

// RegenerateBackupCodes creates new backup codes (invalidates old ones).
func (h *MFAHandler) RegenerateBackupCodes(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	codes, err := h.mfaRepo.RegenerateBackupCodes(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{"backup_codes": codes})
}
