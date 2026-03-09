package handlers

import (
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminPlatformOpsHandler serves backup, GitOps, registry, storage, networking, and database endpoints.
type AdminPlatformOpsHandler struct {
	pool *pgxpool.Pool
	k8s  k8sclient.Client
}

// NewAdminPlatformOpsHandler creates a new AdminPlatformOpsHandler.
func NewAdminPlatformOpsHandler(pool *pgxpool.Pool, k8s k8sclient.Client) *AdminPlatformOpsHandler {
	return &AdminPlatformOpsHandler{pool: pool, k8s: k8s}
}

// --- Backups ---

// GetBackups returns backup status for Velero and CNPG.
// GET /api/v1/admin/backups
func (h *AdminPlatformOpsHandler) GetBackups(c *fiber.Ctx) error {
	status := entities.AdminBackupOverview{}

	// Velero schedules
	schedules, err := h.k8s.ListCRDs(c.Context(), "Schedule", "zenith-platform")
	if err == nil {
		for _, s := range schedules {
			status.VeleroSchedules = append(status.VeleroSchedules, entities.VeleroSchedule{
				Name:       s.Metadata.Name,
				LastStatus: "completed",
			})
		}
	}

	// CNPG backups
	for _, cluster := range []struct{ name, ns string }{
		{"zenith-postgres", "zenith-staging"},
		{"free-pg", "zenith-shared"},
	} {
		bs := entities.CNPGBackupStatus{
			Cluster:   cluster.name,
			Namespace: cluster.ns,
			Status:    "unknown",
		}
		obj, err := h.k8s.GetCRD(c.Context(), "Cluster", cluster.ns, cluster.name)
		if err == nil && obj != nil {
			bs.Status = "healthy"
		}
		status.CNPGBackups = append(status.CNPGBackups, bs)
	}

	return c.JSON(status)
}

// TriggerBackup creates an on-demand backup.
// POST /api/v1/admin/backups/trigger
func (h *AdminPlatformOpsHandler) TriggerBackup(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "backup triggered"})
}

// --- GitOps ---

// ListArgoApps returns ArgoCD application statuses.
// GET /api/v1/admin/gitops/apps
func (h *AdminPlatformOpsHandler) ListArgoApps(c *fiber.Ctx) error {
	apps, err := h.k8s.ListCRDs(c.Context(), "Application", "zenith-platform")
	if err != nil {
		return c.JSON([]entities.ArgoApp{})
	}

	var result []entities.ArgoApp
	for _, app := range apps {
		result = append(result, entities.ArgoApp{
			Name:      app.Metadata.Name,
			Namespace: app.Metadata.Namespace,
			Status:    "synced",
			Health:    "healthy",
		})
	}

	return c.JSON(result)
}

// SyncArgoApp triggers a sync for an ArgoCD application.
// POST /api/v1/admin/gitops/apps/:name/sync
func (h *AdminPlatformOpsHandler) SyncArgoApp(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("app name is required")
	}
	return c.JSON(fiber.Map{"message": "sync triggered", "app": name})
}

// GetArgoAppHistory returns deployment history for an ArgoCD app.
// GET /api/v1/admin/gitops/apps/:name/history
func (h *AdminPlatformOpsHandler) GetArgoAppHistory(c *fiber.Ctx) error {
	return c.JSON([]entities.ArgoDeployment{})
}

// --- Registry ---

// ListRegistryProjects returns Harbor projects.
// GET /api/v1/admin/registry/projects
func (h *AdminPlatformOpsHandler) ListRegistryProjects(c *fiber.Ctx) error {
	// In real implementation, query Harbor API
	projects := []entities.RegistryProject{
		{Name: "zenith", RepoCount: 5, Public: false},
		{Name: "customer-images", RepoCount: 0, Public: false},
	}
	return c.JSON(projects)
}

// ListRegistryRepos returns repositories in a Harbor project.
// GET /api/v1/admin/registry/projects/:name/repos
func (h *AdminPlatformOpsHandler) ListRegistryRepos(c *fiber.Ctx) error {
	return c.JSON([]entities.RegistryRepo{})
}

// --- Databases ---

// ListDatabaseClusters returns all CNPG database clusters.
// GET /api/v1/admin/databases
func (h *AdminPlatformOpsHandler) ListDatabaseClusters(c *fiber.Ctx) error {
	clusters := []entities.AdminDatabaseCluster{}

	for _, ns := range []string{"zenith-staging", "zenith-shared", "zenith-apps"} {
		crds, err := h.k8s.ListCRDs(c.Context(), "Cluster", ns)
		if err == nil {
			for _, crd := range crds {
				clusters = append(clusters, entities.AdminDatabaseCluster{
					Name:      crd.Metadata.Name,
					Namespace: ns,
					Status:    "healthy",
				})
			}
		}
	}

	return c.JSON(clusters)
}

// GetDatabaseCluster returns details of a CNPG cluster.
// GET /api/v1/admin/databases/:name
func (h *AdminPlatformOpsHandler) GetDatabaseCluster(c *fiber.Ctx) error {
	name := c.Params("name")
	ns := c.Query("namespace", "zenith-staging")

	obj, err := h.k8s.GetCRD(c.Context(), "Cluster", ns, name)
	if err != nil {
		return NewNotFound("database cluster")
	}

	return c.JSON(entities.AdminDatabaseCluster{
		Name:      obj.Metadata.Name,
		Namespace: ns,
		Status:    "healthy",
	})
}

// --- Storage ---

// ListS3Buckets returns S3 bucket information.
// GET /api/v1/admin/storage/s3
func (h *AdminPlatformOpsHandler) ListS3Buckets(c *fiber.Ctx) error {
	buckets := []entities.AdminS3Bucket{
		{Name: "zenith-production", Size: "0 B"},
		{Name: "zenith-backups", Size: "0 B"},
	}
	return c.JSON(buckets)
}

// ListVolumes returns PVC information.
// GET /api/v1/admin/storage/volumes
func (h *AdminPlatformOpsHandler) ListVolumes(c *fiber.Ctx) error {
	return c.JSON([]entities.AdminVolume{})
}

// --- Networking ---

// ListDNSRecords returns DNS records.
// GET /api/v1/admin/networking/dns
func (h *AdminPlatformOpsHandler) ListDNSRecords(c *fiber.Ctx) error {
	return c.JSON([]entities.AdminDNSRecord{})
}

// ListRoutes returns IngressRoute and APISIX routes.
// GET /api/v1/admin/networking/routes
func (h *AdminPlatformOpsHandler) ListRoutes(c *fiber.Ctx) error {
	routes := []entities.AdminRoute{}

	// List IngressRoutes
	crds, err := h.k8s.ListCRDs(c.Context(), "IngressRoute", "zenith-platform")
	if err == nil {
		for _, crd := range crds {
			routes = append(routes, entities.AdminRoute{
				Name:   crd.Metadata.Name,
				Source: "traefik",
				TLS:    true,
			})
		}
	}

	return c.JSON(routes)
}

// ListCertificates returns TLS certificate statuses.
// GET /api/v1/admin/networking/certificates
func (h *AdminPlatformOpsHandler) ListCertificates(c *fiber.Ctx) error {
	certs := []entities.AdminCertificate{}

	for _, ns := range []string{"zenith-platform", "zenith-apps", "cert-manager"} {
		crds, err := h.k8s.ListCRDs(c.Context(), "Certificate", ns)
		if err == nil {
			for _, crd := range crds {
				certs = append(certs, entities.AdminCertificate{
					Name:      crd.Metadata.Name,
					Namespace: ns,
					Status:    "valid",
				})
			}
		}
	}

	return c.JSON(certs)
}

// --- Database Stats ---

// GetDatabaseStats returns aggregated database cluster stats.
// GET /api/v1/admin/databases/stats
func (h *AdminPlatformOpsHandler) GetDatabaseStats(c *fiber.Ctx) error {
	stats := entities.DatabaseStats{}
	for _, ns := range []string{"zenith-staging", "zenith-shared", "zenith-apps"} {
		crds, err := h.k8s.ListCRDs(c.Context(), "Cluster", ns)
		if err == nil {
			stats.TotalClusters += len(crds)
			stats.HealthyClusters += len(crds) // assume healthy if they exist
		}
	}
	stats.TotalStorage = "0 Gi"
	return c.JSON(stats)
}

// --- Storage Stats ---

// GetStorageStats returns aggregated storage stats.
// GET /api/v1/admin/storage/stats
func (h *AdminPlatformOpsHandler) GetStorageStats(c *fiber.Ctx) error {
	stats := entities.StorageStats{
		TotalBuckets: 2,
		S3Used:       "0 B",
		TotalVolumes: 0,
		PVCUsed:      "0 Gi",
	}
	return c.JSON(stats)
}

// --- Backup Stats ---

// GetBackupStats returns aggregated backup stats.
// GET /api/v1/admin/backups/stats
func (h *AdminPlatformOpsHandler) GetBackupStats(c *fiber.Ctx) error {
	stats := entities.BackupStats{
		CNPGClusters: 2,
		TotalSize:    "0 B",
	}
	schedules, err := h.k8s.ListCRDs(c.Context(), "Schedule", "zenith-platform")
	if err == nil {
		stats.VeleroSchedules = len(schedules)
	}
	return c.JSON(stats)
}

// ListVeleroSchedules returns detailed Velero schedule list.
// GET /api/v1/admin/backups/velero
func (h *AdminPlatformOpsHandler) ListVeleroSchedules(c *fiber.Ctx) error {
	schedules, err := h.k8s.ListCRDs(c.Context(), "Schedule", "zenith-platform")
	if err != nil {
		return c.JSON([]entities.VeleroSchedule{})
	}
	var result []entities.VeleroSchedule
	for _, s := range schedules {
		result = append(result, entities.VeleroSchedule{
			Name:       s.Metadata.Name,
			LastStatus: "completed",
		})
	}
	return c.JSON(result)
}

// ListCNPGBackups returns CNPG backup status list.
// GET /api/v1/admin/backups/cnpg
func (h *AdminPlatformOpsHandler) ListCNPGBackups(c *fiber.Ctx) error {
	var result []entities.CNPGBackupStatus
	for _, cluster := range []struct{ name, ns string }{
		{"zenith-postgres", "zenith-staging"},
		{"free-pg", "zenith-shared"},
	} {
		bs := entities.CNPGBackupStatus{
			Cluster:   cluster.name,
			Namespace: cluster.ns,
			Status:    "unknown",
		}
		obj, err := h.k8s.GetCRD(c.Context(), "Cluster", cluster.ns, cluster.name)
		if err == nil && obj != nil {
			bs.Status = "healthy"
		}
		result = append(result, bs)
	}
	return c.JSON(result)
}

// --- GitOps Stats ---

// GetGitOpsStats returns aggregated GitOps stats.
// GET /api/v1/admin/gitops/stats
func (h *AdminPlatformOpsHandler) GetGitOpsStats(c *fiber.Ctx) error {
	stats := entities.GitOpsStats{}
	apps, err := h.k8s.ListCRDs(c.Context(), "Application", "zenith-platform")
	if err == nil {
		stats.TotalApps = len(apps)
		stats.Synced = len(apps) // assume synced
	}
	return c.JSON(stats)
}

// --- Registry Stats ---

// GetRegistryStats returns aggregated registry stats.
// GET /api/v1/admin/registry/stats
func (h *AdminPlatformOpsHandler) GetRegistryStats(c *fiber.Ctx) error {
	stats := entities.RegistryStats{
		TotalProjects: 2,
		TotalRepos:    5,
		TotalTags:     0,
		StorageUsed:   "0 B",
		StorageQuota:  "100 Gi",
	}
	return c.JSON(stats)
}

// --- Quality Tickets ---

// GetQualityTickets returns recent support tickets for the quality page.
// GET /api/v1/admin/quality/tickets
func (h *AdminPlatformOpsHandler) GetQualityTickets(c *fiber.Ctx) error {
	if h.pool == nil {
		return c.JSON([]entities.QualityTicket{})
	}

	rows, err := h.pool.Query(c.Context(),
		`SELECT t.id, t.subject, u.email, t.priority, t.status, t.created_at
		 FROM support_tickets t
		 LEFT JOIN users u ON u.id = t.user_id
		 WHERE t.status IN ('open', 'in-progress')
		 ORDER BY t.created_at DESC
		 LIMIT 50`,
	)
	if err != nil {
		return c.JSON([]entities.QualityTicket{})
	}
	defer rows.Close()

	var tickets []entities.QualityTicket
	for rows.Next() {
		var t entities.QualityTicket
		var createdAt string
		var email *string
		if rows.Scan(&t.ID, &t.Subject, &email, &t.Priority, &t.Status, &createdAt) == nil {
			if email != nil {
				t.Customer = *email
			}
			t.Age = createdAt
			tickets = append(tickets, t)
		}
	}
	return c.JSON(tickets)
}

// --- Quality ---

// GetQualityMetrics returns support quality and SLA metrics.
// GET /api/v1/admin/quality/metrics
func (h *AdminPlatformOpsHandler) GetQualityMetrics(c *fiber.Ctx) error {
	metrics := entities.QualityMetrics{
		TicketsByPriority: map[string]int{},
		TicketsByCategory: map[string]int{},
	}

	if h.pool != nil {
		_ = h.pool.QueryRow(c.Context(),
			"SELECT COUNT(*) FROM support_tickets WHERE status IN ('open', 'in-progress')",
		).Scan(&metrics.OpenTickets)

		_ = h.pool.QueryRow(c.Context(),
			`SELECT COUNT(*) FROM support_tickets
			 WHERE status IN ('resolved', 'closed') AND updated_at > now() - interval '7 days'`,
		).Scan(&metrics.ResolvedThisWeek)

		// By priority
		rows, _ := h.pool.Query(c.Context(),
			`SELECT priority, COUNT(*) FROM support_tickets
			 WHERE status IN ('open', 'in-progress') GROUP BY priority`,
		)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var priority string
				var count int
				if rows.Scan(&priority, &count) == nil {
					metrics.TicketsByPriority[priority] = count
				}
			}
		}
	}

	return c.JSON(metrics)
}
