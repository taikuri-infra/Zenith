package handlers

import (
	"fmt"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/pquerna/otp/totp"
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
		"backup_codes": len(enrollment.BackupCodes) - len(enrollment.UsedCodes),
	})
}

// Enable begins MFA enrollment — returns TOTP secret + provisioning URI + backup codes.
func (h *MFAHandler) Enable(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	email, _ := c.Locals("email").(string)

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

	// Build otpauth URI for QR code generation using the user's email
	label := email
	if label == "" {
		label = userID
	}
	otpauthURI := fmt.Sprintf("otpauth://totp/Zenith:%s?secret=%s&issuer=Zenith&digits=6&period=30", label, enrollment.Secret)

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

	// Get the pending enrollment to access the secret
	enrollment, err := h.mfaRepo.GetEnrollment(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "no pending MFA enrollment")
	}
	if enrollment.Status != entities.MFAStatusPending {
		return fiber.NewError(fiber.StatusBadRequest, "no pending MFA enrollment")
	}

	// Validate TOTP code against the secret
	if !totp.Validate(body.Code, enrollment.Secret) {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid TOTP code")
	}

	confirmed, err := h.mfaRepo.ConfirmEnrollment(c.Context(), userID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	return c.JSON(fiber.Map{
		"status":     confirmed.Status,
		"enabled_at": confirmed.EnabledAt,
	})
}

// Disable turns off MFA for the authenticated user after validating TOTP or backup code.
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

	enrollment, err := h.mfaRepo.GetEnrollment(c.Context(), userID)
	if err != nil || enrollment.Status != entities.MFAStatusEnabled {
		return fiber.NewError(fiber.StatusBadRequest, "MFA is not enabled")
	}

	// Try TOTP validation first (6-digit code)
	if len(body.Code) == 6 {
		if !totp.Validate(body.Code, enrollment.Secret) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid TOTP code")
		}
	} else {
		// Try backup code
		used, err := h.mfaRepo.UseBackupCode(c.Context(), userID, body.Code)
		if err != nil || !used {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid backup code")
		}
	}

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

// ValidateTOTP validates a TOTP code against the user's secret. Used by login MFA challenge.
func ValidateTOTP(code, secret string) bool {
	return totp.Validate(code, secret)
}
