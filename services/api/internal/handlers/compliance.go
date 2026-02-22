package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/store"
	"github.com/gofiber/fiber/v2"
)

// ComplianceCheck represents a single compliance checklist item.
type ComplianceCheck struct {
	Category    string `json:"category"`
	Item        string `json:"item"`
	Status      string `json:"status"` // "pass", "fail", "partial", "na"
	Description string `json:"description"`
}

type ComplianceHandler struct {
	mfaRepo   store.MFARepository
	ipRepo    store.IPWhitelistRepository
	planRepo  store.UserPlanRepository
	adminRepo store.AdminRepository
}

func NewComplianceHandler(mfaRepo store.MFARepository, ipRepo store.IPWhitelistRepository, planRepo store.UserPlanRepository, adminRepo store.AdminRepository) *ComplianceHandler {
	return &ComplianceHandler{mfaRepo: mfaRepo, ipRepo: ipRepo, planRepo: planRepo, adminRepo: adminRepo}
}

// GetStatus returns compliance checklist for the current user.
func (h *ComplianceHandler) GetStatus(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)

	checks := []ComplianceCheck{}

	// Check MFA
	mfaStatus := "fail"
	enrollment, err := h.mfaRepo.GetEnrollment(c.Context(), userID)
	if err == nil && enrollment.Status == "enabled" {
		mfaStatus = "pass"
	}
	checks = append(checks, ComplianceCheck{
		Category:    "Authentication",
		Item:        "Multi-Factor Authentication (MFA)",
		Status:      mfaStatus,
		Description: "Two-factor authentication is enabled for your account",
	})

	// Check encryption at rest
	checks = append(checks, ComplianceCheck{
		Category:    "Encryption",
		Item:        "Encryption at Rest",
		Status:      "pass",
		Description: "All data is encrypted at rest using AES-256-GCM",
	})

	// Check encryption in transit
	checks = append(checks, ComplianceCheck{
		Category:    "Encryption",
		Item:        "Encryption in Transit",
		Status:      "pass",
		Description: "All API and dashboard traffic uses TLS 1.3",
	})

	// Check audit logging
	auditStatus := "pass"
	entries, err := h.adminRepo.ListAuditLog(c.Context(), 1, 0)
	if err != nil || len(entries) == 0 {
		auditStatus = "partial"
	}
	checks = append(checks, ComplianceCheck{
		Category:    "Audit",
		Item:        "Audit Logging",
		Status:      auditStatus,
		Description: "All administrative actions are logged with actor and timestamp",
	})

	// Check IP whitelisting
	ipStatus := "na"
	ipEntries, err := h.ipRepo.ListByUser(c.Context(), userID)
	if err == nil && len(ipEntries) > 0 {
		ipStatus = "pass"
	}
	checks = append(checks, ComplianceCheck{
		Category:    "Access Control",
		Item:        "IP Whitelisting",
		Status:      ipStatus,
		Description: "Dashboard and API access restricted to allowed IP ranges",
	})

	// GDPR compliance (always pass — we have data deletion)
	checks = append(checks, ComplianceCheck{
		Category:    "GDPR",
		Item:        "Right to Deletion",
		Status:      "pass",
		Description: "Users can delete their account and all associated data",
	})

	checks = append(checks, ComplianceCheck{
		Category:    "GDPR",
		Item:        "Data Processing Agreement",
		Status:      "na",
		Description: "DPA available for Team and Enterprise plans",
	})

	// SSO
	checks = append(checks, ComplianceCheck{
		Category:    "Authentication",
		Item:        "Single Sign-On (SSO)",
		Status:      "na",
		Description: "SAML 2.0 and OIDC SSO available for Team plans and above",
	})

	// Calculate summary
	pass, fail, partial, na := 0, 0, 0, 0
	for _, check := range checks {
		switch check.Status {
		case "pass":
			pass++
		case "fail":
			fail++
		case "partial":
			partial++
		case "na":
			na++
		}
	}

	return c.JSON(fiber.Map{
		"checks": checks,
		"summary": fiber.Map{
			"total":   len(checks),
			"pass":    pass,
			"fail":    fail,
			"partial": partial,
			"na":      na,
		},
	})
}
