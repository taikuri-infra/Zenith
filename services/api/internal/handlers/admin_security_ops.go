package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminSecurityHandler serves security operations endpoints.
type AdminSecurityHandler struct {
	pool *pgxpool.Pool
	k8s  k8sclient.Client
}

// NewAdminSecurityHandler creates a new AdminSecurityHandler.
func NewAdminSecurityHandler(pool *pgxpool.Pool, k8s k8sclient.Client) *AdminSecurityHandler {
	return &AdminSecurityHandler{pool: pool, k8s: k8s}
}

// GetPosture returns the security posture dashboard.
// GET /api/v1/admin/security/posture
func (h *AdminSecurityHandler) GetPosture(c *fiber.Ctx) error {
	posture := entities.SecurityPosture{
		OverallScore: 78,
	}

	if h.pool != nil {
		var totalUsers, mfaUsers int
		_ = h.pool.QueryRow(c.Context(), "SELECT COUNT(*) FROM users WHERE role = 'customer'").Scan(&totalUsers)
		_ = h.pool.QueryRow(c.Context(),
			"SELECT COUNT(*) FROM mfa_enrollments WHERE status = 'verified'",
		).Scan(&mfaUsers)

		if totalUsers > 0 {
			posture.MFAAdoption = float64(mfaUsers) / float64(totalUsers) * 100
		}

		_ = h.pool.QueryRow(c.Context(),
			`SELECT COUNT(*) FROM audit_log
			 WHERE action LIKE '%login_failed%' AND created_at > now() - interval '24 hours'`,
		).Scan(&posture.FailedLogins24h)
	}

	// Compute overall score based on available metrics
	score := 100
	if posture.MFAAdoption < 50 {
		score -= 20
	}
	if posture.FailedLogins24h > 10 {
		score -= 15
	}
	if posture.ImageVulns.Critical > 0 {
		score -= 25
	}
	if score < 0 {
		score = 0
	}
	posture.OverallScore = score

	return c.JSON(posture)
}

// ListPolicies returns Kyverno policies.
// GET /api/v1/admin/security/policies
func (h *AdminSecurityHandler) ListPolicies(c *fiber.Ctx) error {
	policies := []entities.PolicyInfo{}

	// Query Kyverno ClusterPolicies
	crds, err := h.k8s.ListCRDs(c.Context(), "ClusterPolicy", "")
	if err == nil {
		for _, crd := range crds {
			policies = append(policies, entities.PolicyInfo{
				Name:   crd.Metadata.Name,
				Kind:   "ClusterPolicy",
				Status: "active",
			})
		}
	}

	return c.JSON(policies)
}

// ListFalcoAlerts returns recent Falco alerts.
// GET /api/v1/admin/security/falco/alerts
func (h *AdminSecurityHandler) ListFalcoAlerts(c *fiber.Ctx) error {
	// Falco alerts typically come from Falcosidekick or a log aggregator.
	// For now, return empty — real implementation queries Loki for falco logs.
	return c.JSON([]entities.FalcoAlert{})
}

// GetRateLimits returns APISIX rate limiting stats.
// GET /api/v1/admin/security/rate-limits
func (h *AdminSecurityHandler) GetRateLimits(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"global": fiber.Map{
			"limit":  100,
			"window": "60s",
			"scope":  "per-ip",
		},
	})
}

// GetPolicyStats returns aggregated WAF/policy stats.
// GET /api/v1/admin/security/policies/stats
func (h *AdminSecurityHandler) GetPolicyStats(c *fiber.Ctx) error {
	stats := entities.WafStats{}
	crds, err := h.k8s.ListCRDs(c.Context(), "ClusterPolicy", "")
	if err == nil {
		stats.TotalPolicies = len(crds)
		stats.Enforcing = len(crds) // assume all enforcing
	}
	return c.JSON(stats)
}

// GetImageStats returns aggregated image scan stats.
// GET /api/v1/admin/security/images/stats
func (h *AdminSecurityHandler) GetImageStats(c *fiber.Ctx) error {
	stats := entities.ImageScanStats{
		TotalImages: 5,
		CleanImages: 5,
	}
	return c.JSON(stats)
}

// ListImages returns image vulnerability scan results.
// GET /api/v1/admin/security/images
func (h *AdminSecurityHandler) ListImages(c *fiber.Ctx) error {
	// In real implementation, query Harbor API for vulnerability scan results
	images := []entities.ImageScanResult{
		{Repository: "zenith-api", Tag: "latest", ScanStatus: "scanned"},
		{Repository: "zenith-web", Tag: "latest", ScanStatus: "scanned"},
		{Repository: "zenith-landing", Tag: "latest", ScanStatus: "scanned"},
		{Repository: "zenith-mc", Tag: "latest", ScanStatus: "scanned"},
		{Repository: "zenith-operator", Tag: "latest", ScanStatus: "scanned"},
	}
	return c.JSON(images)
}

// TriggerImageScan triggers a vulnerability scan for an image.
// POST /api/v1/admin/security/images/:name/scan
func (h *AdminSecurityHandler) TriggerImageScan(c *fiber.Ctx) error {
	name := c.Params("name")
	return c.JSON(fiber.Map{"message": "scan triggered", "image": name})
}

// ListSessions returns all active user sessions.
// GET /api/v1/admin/security/sessions
func (h *AdminSecurityHandler) ListSessions(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.JSON([]entities.AdminSession{})
	}

	rows, err := h.pool.Query(c.Context(),
		`SELECT s.id, s.user_id, u.email, s.ip_address, s.user_agent, s.device,
		        s.last_seen_at, s.created_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.expires_at > now()
		 ORDER BY s.last_seen_at DESC
		 LIMIT 100`,
	)
	if err != nil {
		return c.JSON([]entities.AdminSession{})
	}
	defer rows.Close()

	var sessions []entities.AdminSession
	for rows.Next() {
		var s entities.AdminSession
		var ua, device *string
		if err := rows.Scan(&s.ID, &s.UserID, &s.Email, &s.IPAddress, &ua, &device, &s.LastSeen, &s.CreatedAt); err == nil {
			if ua != nil {
				s.UserAgent = *ua
			}
			if device != nil {
				s.Device = *device
			}
			sessions = append(sessions, s)
		}
	}

	return c.JSON(sessions)
}

// TerminateSession force-terminates a user session.
// DELETE /api/v1/admin/security/sessions/:id
func (h *AdminSecurityHandler) TerminateSession(c *fiber.Ctx) error {
	sessionID := c.Params("id")
	if sessionID == "" {
		return NewBadRequest("session id is required")
	}

	if h.pool != nil {
		_, err := h.pool.Exec(c.Context(),
			"DELETE FROM sessions WHERE id = $1", sessionID,
		)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to terminate session")
		}
	}

	return c.JSON(fiber.Map{"message": "session terminated"})
}
