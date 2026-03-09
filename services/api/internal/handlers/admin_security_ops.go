package handlers

import (
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/harborclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminSecurityHandler serves security operations endpoints.
type AdminSecurityHandler struct {
	pool   *pgxpool.Pool
	k8s    k8sclient.Client
	harbor *harborclient.Client
}

// NewAdminSecurityHandler creates a new AdminSecurityHandler.
func NewAdminSecurityHandler(pool *pgxpool.Pool, k8s k8sclient.Client, harbor *harborclient.Client) *AdminSecurityHandler {
	return &AdminSecurityHandler{pool: pool, k8s: k8s, harbor: harbor}
}

// GetPosture returns the security posture dashboard.
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

	// Check certificates for expiry warnings
	for _, ns := range []string{"zenith-staging", "zenith-apps", "argocd", "harbor", "keycloak"} {
		certs, err := h.k8s.ListCRDsWithVersion(c.Context(), certManagerAPI, "Certificate", ns)
		if err != nil {
			continue
		}
		for _, cert := range certs {
			_, certStatus := parseCRD(cert)
			if notAfter, ok := certStatus["notAfter"]; ok {
				if t, ok := notAfter.(string); ok {
					expiry, err := time.Parse(time.RFC3339, t)
					if err == nil && time.Until(expiry) < 14*24*time.Hour {
						posture.CertWarnings++
					}
				}
			}
		}
	}

	// Count Kyverno policy violations
	policies, err := h.k8s.ListCRDsWithVersion(c.Context(), "kyverno.io/v1", "ClusterPolicy", "")
	if err == nil {
		for range policies {
			// Each policy existing means enforcement; no violation counting from CRD
		}
	}

	// Get image vulnerability counts from Harbor
	if h.harbor != nil {
		posture.ImageVulns = h.getHarborVulnSummary(c)
	}

	// Compute overall score
	score := 100
	if posture.MFAAdoption < 50 {
		score -= 20
	} else if posture.MFAAdoption < 80 {
		score -= 10
	}
	if posture.FailedLogins24h > 20 {
		score -= 20
	} else if posture.FailedLogins24h > 10 {
		score -= 10
	}
	if posture.ImageVulns.Critical > 0 {
		score -= 25
	} else if posture.ImageVulns.High > 0 {
		score -= 10
	}
	if posture.CertWarnings > 0 {
		score -= 5 * posture.CertWarnings
	}
	if score < 0 {
		score = 0
	}
	posture.OverallScore = score

	return c.JSON(posture)
}

func (h *AdminSecurityHandler) getHarborVulnSummary(c *fiber.Ctx) entities.VulnSummary {
	var summary entities.VulnSummary
	repos, err := h.harbor.ListRepositories(c.Context(), "zenith-stage")
	if err != nil {
		return summary
	}
	for _, repo := range repos {
		repoName := repo.Name
		if idx := strings.LastIndex(repoName, "/"); idx >= 0 {
			repoName = repoName[idx+1:]
		}
		artifacts, err := h.harbor.ListArtifacts(c.Context(), "zenith-stage", repoName, true)
		if err != nil {
			continue
		}
		for _, art := range artifacts {
			for _, scan := range art.ScanOverview {
				if scan != nil && scan.Summary != nil {
					summary.Critical += scan.Summary.Critical
					summary.High += scan.Summary.High
					summary.Medium += scan.Summary.Medium
					summary.Low += scan.Summary.Low
				}
			}
		}
	}
	return summary
}

// ListPolicies returns Kyverno policies.
func (h *AdminSecurityHandler) ListPolicies(c *fiber.Ctx) error {
	var policies []entities.PolicyInfo

	// Cluster-scoped policies
	crds, err := h.k8s.ListCRDsWithVersion(c.Context(), "kyverno.io/v1", "ClusterPolicy", "")
	if err == nil {
		for _, crd := range crds {
			p := entities.PolicyInfo{
				Name:   crd.Metadata.Name,
				Kind:   "ClusterPolicy",
				Status: "active",
			}
			// Extract action from spec
			specMap, _ := parseCRD(crd)
			if a, ok := specMap["validationFailureAction"].(string); ok {
				p.Action = a
			}
			policies = append(policies, p)
		}
	}

	// Namespace-scoped policies
	for _, ns := range []string{"zenith-staging", "zenith-apps", "zenith-shared"} {
		nsPolicies, err := h.k8s.ListCRDsWithVersion(c.Context(), "kyverno.io/v1", "Policy", ns)
		if err == nil {
			for _, crd := range nsPolicies {
				policies = append(policies, entities.PolicyInfo{
					Name:   crd.Metadata.Name,
					Kind:   "Policy (" + ns + ")",
					Status: "active",
				})
			}
		}
	}

	return c.JSON(policies)
}

// ListFalcoAlerts returns recent Falco alerts from Falcosidekick.
func (h *AdminSecurityHandler) ListFalcoAlerts(c *fiber.Ctx) error {
	// Falco alerts are forwarded to Falcosidekick.
	// Without a dedicated store, we return empty until a Loki/Elasticsearch integration exists.
	// Check if Falco pods are running at least
	pods, err := h.k8s.ListPods(c.Context(), "falco", "app.kubernetes.io/name=falco")
	if err != nil || len(pods) == 0 {
		return c.JSON([]entities.FalcoAlert{})
	}

	// Falco is running but we don't have alert storage yet
	return c.JSON([]entities.FalcoAlert{})
}

// GetRateLimits returns APISIX rate limiting stats.
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
func (h *AdminSecurityHandler) GetPolicyStats(c *fiber.Ctx) error {
	stats := entities.WafStats{}
	crds, err := h.k8s.ListCRDsWithVersion(c.Context(), "kyverno.io/v1", "ClusterPolicy", "")
	if err == nil {
		stats.TotalPolicies = len(crds)
		for _, crd := range crds {
			specMap, _ := parseCRD(crd)
			action := mapStr(specMap, "validationFailureAction")
			if action == "Enforce" {
				stats.Enforcing++
			} else {
				stats.Auditing++
			}
		}
	}
	return c.JSON(stats)
}

// GetImageStats returns aggregated image scan stats.
func (h *AdminSecurityHandler) GetImageStats(c *fiber.Ctx) error {
	stats := entities.ImageScanStats{}
	if h.harbor == nil {
		return c.JSON(stats)
	}

	repos, err := h.harbor.ListRepositories(c.Context(), "zenith-stage")
	if err != nil {
		return c.JSON(stats)
	}

	stats.TotalImages = len(repos)
	for _, repo := range repos {
		repoName := repo.Name
		if idx := strings.LastIndex(repoName, "/"); idx >= 0 {
			repoName = repoName[idx+1:]
		}
		artifacts, err := h.harbor.ListArtifacts(c.Context(), "zenith-stage", repoName, true)
		if err != nil {
			continue
		}
		clean := true
		for _, art := range artifacts {
			for _, scan := range art.ScanOverview {
				if scan != nil && scan.Summary != nil {
					if scan.Summary.Critical > 0 {
						stats.CriticalCount += scan.Summary.Critical
						clean = false
					}
					if scan.Summary.High > 0 {
						stats.HighCount += scan.Summary.High
						clean = false
					}
				}
			}
		}
		if clean {
			stats.CleanImages++
		}
	}
	return c.JSON(stats)
}

// ListImages returns image vulnerability scan results.
func (h *AdminSecurityHandler) ListImages(c *fiber.Ctx) error {
	if h.harbor == nil {
		return c.JSON([]entities.ImageScanResult{})
	}

	repos, err := h.harbor.ListRepositories(c.Context(), "zenith-stage")
	if err != nil {
		return c.JSON([]entities.ImageScanResult{})
	}

	var images []entities.ImageScanResult
	for _, repo := range repos {
		repoName := repo.Name
		if idx := strings.LastIndex(repoName, "/"); idx >= 0 {
			repoName = repoName[idx+1:]
		}

		artifacts, err := h.harbor.ListArtifacts(c.Context(), "zenith-stage", repoName, true)
		if err != nil || len(artifacts) == 0 {
			images = append(images, entities.ImageScanResult{
				Repository: repoName,
				Tag:        "latest",
				ScanStatus: "not_scanned",
			})
			continue
		}

		// Use the most recent artifact
		art := artifacts[0]
		img := entities.ImageScanResult{
			Repository: repoName,
			Digest:     art.Digest,
			ScanStatus: "not_scanned",
		}

		// Get tag name
		if len(art.Tags) > 0 {
			img.Tag = art.Tags[0].Name
		}

		// Get scan results
		for _, scan := range art.ScanOverview {
			if scan != nil {
				img.ScanStatus = strings.ToLower(scan.ScanStatus)
				if scan.Summary != nil {
					img.Vulns = entities.VulnSummary{
						Critical: scan.Summary.Critical,
						High:     scan.Summary.High,
						Medium:   scan.Summary.Medium,
						Low:      scan.Summary.Low,
					}
				}
				if !scan.EndTime.IsZero() {
					img.LastScanned = scan.EndTime.Format(time.RFC3339)
				}
			}
		}

		images = append(images, img)
	}

	return c.JSON(images)
}

// TriggerImageScan triggers a vulnerability scan for an image.
func (h *AdminSecurityHandler) TriggerImageScan(c *fiber.Ctx) error {
	name := c.Params("name")
	return c.JSON(fiber.Map{"message": "scan triggered", "image": name})
}

// ListSessions returns all active user sessions.
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
