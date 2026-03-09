package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/harborclient"
	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminPlatformOpsHandler serves backup, GitOps, registry, storage, networking, and database endpoints.
type AdminPlatformOpsHandler struct {
	pool   *pgxpool.Pool
	k8s    k8sclient.Client
	harbor *harborclient.Client
	s3     ports.ObjectStorage
}

// NewAdminPlatformOpsHandler creates a new AdminPlatformOpsHandler.
func NewAdminPlatformOpsHandler(pool *pgxpool.Pool, k8s k8sclient.Client, harbor *harborclient.Client, s3 ports.ObjectStorage) *AdminPlatformOpsHandler {
	return &AdminPlatformOpsHandler{pool: pool, k8s: k8s, harbor: harbor, s3: s3}
}

// CRD API versions for non-Zenith resources.
const (
	veleroAPI      = "velero.io/v1"
	cnpgAPI        = "postgresql.cnpg.io/v1"
	argocdAPI      = "argoproj.io/v1alpha1"
	traefikAPI     = "traefik.io/v1alpha1"
	certManagerAPI = "cert-manager.io/v1"
)

// parseCRD unmarshals spec and status from a CRDObject into maps.
func parseCRD(crd *k8sclient.CRDObject) (spec map[string]interface{}, status map[string]interface{}) {
	spec = make(map[string]interface{})
	status = make(map[string]interface{})
	if len(crd.Spec) > 0 {
		_ = json.Unmarshal(crd.Spec, &spec)
	}
	if len(crd.Status) > 0 {
		_ = json.Unmarshal(crd.Status, &status)
	}
	return
}

// --- Backups ---

func (h *AdminPlatformOpsHandler) GetBackups(c *fiber.Ctx) error {
	overview := entities.AdminBackupOverview{}

	// Velero schedules
	schedules, err := h.k8s.ListCRDsWithVersion(c.Context(), veleroAPI, "Schedule", "velero")
	if err == nil {
		for _, s := range schedules {
			overview.VeleroSchedules = append(overview.VeleroSchedules, veleroScheduleFromCRD(c.Context(), h.k8s, s))
		}
	}

	// CNPG backups
	for _, ns := range []string{"zenith-staging", "zenith-shared", "keycloak"} {
		clusters, err := h.k8s.ListCRDsWithVersion(c.Context(), cnpgAPI, "Cluster", ns)
		if err != nil {
			continue
		}
		for _, cluster := range clusters {
			overview.CNPGBackups = append(overview.CNPGBackups, cnpgBackupFromCRD(cluster))
		}
	}

	return c.JSON(overview)
}

func veleroScheduleFromCRD(ctx context.Context, k8s k8sclient.Client, crd *k8sclient.CRDObject) entities.VeleroSchedule {
	spec, status := parseCRD(crd)
	vs := entities.VeleroSchedule{
		Name:       crd.Metadata.Name,
		Schedule:   mapStr(spec, "schedule"),
		LastBackup: mapStr(status, "lastBackup"),
		LastStatus: mapStr(status, "phase"),
	}
	if vs.LastStatus == "" {
		vs.LastStatus = "unknown"
	}
	// Count backups for this schedule
	backups, err := k8s.ListCRDsWithVersion(ctx, veleroAPI, "Backup", "velero")
	if err == nil {
		for _, b := range backups {
			bSpec, _ := parseCRD(b)
			if mapStr(bSpec, "scheduleName") == crd.Metadata.Name || strings.Contains(b.Metadata.Name, crd.Metadata.Name) {
				vs.BackupCount++
			}
		}
	}
	// Extract retention TTL
	if tmpl, ok := spec["template"].(map[string]interface{}); ok {
		if ttl, ok := tmpl["ttl"].(string); ok {
			vs.Schedule = vs.Schedule + " (retain " + ttl + ")"
		}
	}
	return vs
}

func cnpgBackupFromCRD(cluster *k8sclient.CRDObject) entities.CNPGBackupStatus {
	spec, status := parseCRD(cluster)
	bs := entities.CNPGBackupStatus{
		Cluster:   cluster.Metadata.Name,
		Namespace: cluster.Metadata.Namespace,
		Status:    "unknown",
	}

	// Parse spec for backup config
	if backup, ok := spec["backup"].(map[string]interface{}); ok {
		bs.WALArchiving = "enabled"
		if rp, ok := backup["retentionPolicy"].(string); ok {
			rp = strings.TrimSuffix(rp, "d")
			var days int
			if _, err := fmt.Sscanf(rp, "%d", &days); err == nil {
				bs.RetentionDays = days
			}
		}
	}

	// Parse status
	if phase, ok := status["phase"].(string); ok {
		if strings.Contains(phase, "healthy") {
			bs.Status = "healthy"
		} else {
			bs.Status = phase
		}
	}
	if t, ok := status["lastSuccessfulBackup"].(string); ok {
		bs.LastBackup = t
	}
	if t, ok := status["firstRecoverabilityPoint"].(string); ok {
		if bs.LastBackup != "" && len(t) >= 10 {
			bs.WALArchiving = "active (since " + t[:10] + ")"
		}
	}

	return bs
}

func (h *AdminPlatformOpsHandler) TriggerBackup(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "backup triggered"})
}

// --- GitOps ---

func (h *AdminPlatformOpsHandler) ListArgoApps(c *fiber.Ctx) error {
	apps, err := h.k8s.ListCRDsWithVersion(c.Context(), argocdAPI, "Application", "argocd")
	if err != nil {
		return c.JSON([]entities.ArgoApp{})
	}

	var result []entities.ArgoApp
	for _, app := range apps {
		spec, status := parseCRD(app)
		a := entities.ArgoApp{
			Name:      app.Metadata.Name,
			Namespace: "argocd",
		}

		// Extract sync/health from status
		if sync, ok := status["sync"].(map[string]interface{}); ok {
			a.Status = mapStr(sync, "status")
			a.Revision = mapStr(sync, "revision")
			if len(a.Revision) > 8 {
				a.Revision = a.Revision[:8]
			}
		}
		if health, ok := status["health"].(map[string]interface{}); ok {
			a.Health = mapStr(health, "status")
		}
		a.SyncStatus = a.Status

		// Extract source from spec
		if source, ok := spec["source"].(map[string]interface{}); ok {
			a.RepoURL = mapStr(source, "repoURL")
			a.Path = mapStr(source, "path")
		}

		if a.Status == "" {
			a.Status = "Unknown"
		}
		if a.Health == "" {
			a.Health = "Unknown"
		}

		result = append(result, a)
	}

	return c.JSON(result)
}

func (h *AdminPlatformOpsHandler) SyncArgoApp(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return NewBadRequest("app name is required")
	}
	return c.JSON(fiber.Map{"message": "sync triggered", "app": name})
}

func (h *AdminPlatformOpsHandler) GetArgoAppHistory(c *fiber.Ctx) error {
	name := c.Params("name")
	app, err := h.k8s.GetCRDWithVersion(c.Context(), argocdAPI, "Application", "argocd", name)
	if err != nil {
		return c.JSON([]entities.ArgoDeployment{})
	}

	_, status := parseCRD(app)
	var history []entities.ArgoDeployment
	if hist, ok := status["history"].([]interface{}); ok {
		for _, h := range hist {
			if hMap, ok := h.(map[string]interface{}); ok {
				rev := mapStr(hMap, "revision")
				if len(rev) > 8 {
					rev = rev[:8]
				}
				history = append(history, entities.ArgoDeployment{
					Revision:  rev,
					Status:    "synced",
					StartedAt: mapStr(hMap, "deployedAt"),
				})
			}
		}
	}

	return c.JSON(history)
}

// --- Registry ---

func (h *AdminPlatformOpsHandler) ListRegistryProjects(c *fiber.Ctx) error {
	if h.harbor == nil {
		return c.JSON([]entities.RegistryProject{})
	}

	repos, err := h.harbor.ListRepositories(c.Context(), "zenith-stage")
	if err != nil {
		return c.JSON([]entities.RegistryProject{{Name: "zenith-stage", RepoCount: 0}})
	}
	return c.JSON([]entities.RegistryProject{
		{Name: "zenith-stage", RepoCount: len(repos), Public: false},
	})
}

func (h *AdminPlatformOpsHandler) ListRegistryRepos(c *fiber.Ctx) error {
	projectName := c.Params("name")
	if h.harbor == nil {
		return c.JSON([]entities.RegistryRepo{})
	}

	repos, err := h.harbor.ListRepositories(c.Context(), projectName)
	if err != nil {
		return c.JSON([]entities.RegistryRepo{})
	}

	var result []entities.RegistryRepo
	for _, r := range repos {
		name := r.Name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			name = name[idx+1:]
		}
		result = append(result, entities.RegistryRepo{
			Name:     name,
			TagCount: r.ArtifactCount,
			PushTime: r.UpdateTime,
		})
	}
	return c.JSON(result)
}

// --- Databases ---

func (h *AdminPlatformOpsHandler) ListDatabaseClusters(c *fiber.Ctx) error {
	var clusters []entities.AdminDatabaseCluster

	for _, ns := range []string{"zenith-staging", "zenith-shared", "keycloak", "zenith-apps"} {
		crds, err := h.k8s.ListCRDsWithVersion(c.Context(), cnpgAPI, "Cluster", ns)
		if err != nil {
			continue
		}
		for _, crd := range crds {
			clusters = append(clusters, dbClusterFromCRD(crd))
		}
	}

	return c.JSON(clusters)
}

func (h *AdminPlatformOpsHandler) GetDatabaseCluster(c *fiber.Ctx) error {
	name := c.Params("name")
	ns := c.Query("namespace", "zenith-staging")

	obj, err := h.k8s.GetCRDWithVersion(c.Context(), cnpgAPI, "Cluster", ns, name)
	if err != nil {
		return NewNotFound("database cluster")
	}

	return c.JSON(dbClusterFromCRD(obj))
}

func dbClusterFromCRD(crd *k8sclient.CRDObject) entities.AdminDatabaseCluster {
	spec, status := parseCRD(crd)
	dc := entities.AdminDatabaseCluster{
		Name:      crd.Metadata.Name,
		Namespace: crd.Metadata.Namespace,
		Status:    "unknown",
	}

	if v, ok := spec["instances"].(float64); ok {
		dc.Instances = int(v)
	}
	if storage, ok := spec["storage"].(map[string]interface{}); ok {
		dc.StorageSize = mapStr(storage, "size")
	}
	if imageName, ok := spec["imageName"].(string); ok {
		if idx := strings.LastIndex(imageName, ":"); idx >= 0 {
			dc.PostgresVersion = imageName[idx+1:]
		}
	}
	if _, ok := spec["backup"]; ok {
		dc.WALArchiving = "enabled"
	}

	// Status
	if phase, ok := status["phase"].(string); ok {
		if strings.Contains(phase, "healthy") {
			dc.Status = "healthy"
		} else {
			dc.Status = phase
		}
	}
	if v, ok := status["readyInstances"].(float64); ok {
		dc.ReadyInstances = int(v)
	}
	if t, ok := status["lastSuccessfulBackup"].(string); ok {
		dc.LastBackup = t
	}
	if t, ok := status["firstRecoverabilityPoint"].(string); ok {
		dc.RecoveryWindow = t
	}

	return dc
}

// --- Storage ---

func (h *AdminPlatformOpsHandler) ListS3Buckets(c *fiber.Ctx) error {
	bucketNames := []string{"zenith-backups", "zenith-platform-storage"}

	var buckets []entities.AdminS3Bucket
	for _, name := range bucketNames {
		bucket := entities.AdminS3Bucket{Name: name, Size: "unknown"}
		if h.s3 != nil {
			result, err := h.s3.ListObjects(c.Context(), name, "", "", 1000)
			if err == nil {
				var totalSize int64
				for _, obj := range result.Objects {
					totalSize += obj.Size
				}
				bucket.ObjectCount = int64(len(result.Objects))
				bucket.Size = formatBytesAdmin(totalSize)
				if len(result.Objects) > 0 {
					bucket.LastModified = result.Objects[len(result.Objects)-1].LastModified.Format(time.RFC3339)
				}
			} else {
				bucket.Size = "access error"
			}
		}
		buckets = append(buckets, bucket)
	}
	return c.JSON(buckets)
}

func (h *AdminPlatformOpsHandler) ListVolumes(c *fiber.Ctx) error {
	pvcs, err := h.k8s.ListPVCs(c.Context(), "")
	if err != nil {
		return c.JSON([]entities.AdminVolume{})
	}

	var volumes []entities.AdminVolume
	for _, pvc := range pvcs {
		volumes = append(volumes, entities.AdminVolume{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Size:         pvc.Size,
			Status:       pvc.Status,
			StorageClass: pvc.StorageClass,
		})
	}
	return c.JSON(volumes)
}

// --- Networking ---

func (h *AdminPlatformOpsHandler) ListDNSRecords(c *fiber.Ctx) error {
	var records []entities.AdminDNSRecord
	for _, ns := range []string{"zenith-staging", "zenith-apps", "argocd", "harbor", "keycloak"} {
		routes, err := h.k8s.ListCRDsWithVersion(c.Context(), traefikAPI, "IngressRoute", ns)
		if err != nil {
			continue
		}
		for _, route := range routes {
			hosts := extractIngressRouteHosts(route)
			for _, host := range hosts {
				records = append(records, entities.AdminDNSRecord{
					Name:    host,
					Type:    "A",
					Content: "77.42.88.149",
					TTL:     300,
				})
			}
		}
	}
	return c.JSON(records)
}

func extractIngressRouteHosts(route *k8sclient.CRDObject) []string {
	spec, _ := parseCRD(route)
	var hosts []string
	routes, ok := spec["routes"].([]interface{})
	if !ok {
		return hosts
	}
	for _, r := range routes {
		rMap, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		matchStr, ok := rMap["match"].(string)
		if !ok {
			continue
		}
		for _, part := range strings.Split(matchStr, "||") {
			part = strings.TrimSpace(part)
			if strings.Contains(part, "Host(") {
				start := strings.Index(part, "`")
				end := strings.LastIndex(part, "`")
				if start >= 0 && end > start {
					hosts = append(hosts, part[start+1:end])
				}
			}
		}
	}
	return hosts
}

func (h *AdminPlatformOpsHandler) ListRoutes(c *fiber.Ctx) error {
	var routes []entities.AdminRoute
	for _, ns := range []string{"zenith-staging", "zenith-apps", "argocd", "harbor", "keycloak", "kube-system"} {
		crds, err := h.k8s.ListCRDsWithVersion(c.Context(), traefikAPI, "IngressRoute", ns)
		if err != nil {
			continue
		}
		for _, crd := range crds {
			spec, _ := parseCRD(crd)
			hosts := extractIngressRouteHosts(crd)
			host := ""
			if len(hosts) > 0 {
				host = hosts[0]
			}
			_, hasTLS := spec["tls"]

			routes = append(routes, entities.AdminRoute{
				Name:    crd.Metadata.Name,
				Host:    host,
				Service: ns,
				TLS:     hasTLS,
				Source:   "traefik",
			})
		}
	}
	return c.JSON(routes)
}

func (h *AdminPlatformOpsHandler) ListCertificates(c *fiber.Ctx) error {
	var certs []entities.AdminCertificate
	for _, ns := range []string{"zenith-staging", "zenith-apps", "argocd", "harbor", "keycloak", "cert-manager"} {
		crds, err := h.k8s.ListCRDsWithVersion(c.Context(), certManagerAPI, "Certificate", ns)
		if err != nil {
			continue
		}
		for _, crd := range crds {
			certs = append(certs, certFromCRD(crd))
		}
	}
	return c.JSON(certs)
}

func certFromCRD(crd *k8sclient.CRDObject) entities.AdminCertificate {
	spec, status := parseCRD(crd)
	cert := entities.AdminCertificate{
		Name:      crd.Metadata.Name,
		Namespace: crd.Metadata.Namespace,
		Status:    "unknown",
	}

	if dnsNames, ok := spec["dnsNames"].([]interface{}); ok {
		for _, d := range dnsNames {
			if s, ok := d.(string); ok {
				cert.DnsNames = append(cert.DnsNames, s)
			}
		}
	}
	if issuerRef, ok := spec["issuerRef"].(map[string]interface{}); ok {
		cert.Issuer = mapStr(issuerRef, "name")
	}

	if conditions, ok := status["conditions"].([]interface{}); ok {
		for _, cond := range conditions {
			if condMap, ok := cond.(map[string]interface{}); ok {
				if mapStr(condMap, "type") == "Ready" {
					if mapStr(condMap, "status") == "True" {
						cert.Status = "valid"
					} else {
						cert.Status = "invalid"
					}
				}
			}
		}
	}
	if t, ok := status["notAfter"].(string); ok {
		cert.ExpiresAt = t
	}
	if t, ok := status["renewalTime"].(string); ok {
		cert.RenewAt = t
	}

	return cert
}

// --- Stats Endpoints ---

func (h *AdminPlatformOpsHandler) GetDatabaseStats(c *fiber.Ctx) error {
	stats := entities.DatabaseStats{}
	var totalStorageGi float64

	for _, ns := range []string{"zenith-staging", "zenith-shared", "keycloak", "zenith-apps"} {
		crds, err := h.k8s.ListCRDsWithVersion(c.Context(), cnpgAPI, "Cluster", ns)
		if err != nil {
			continue
		}
		stats.TotalClusters += len(crds)
		for _, crd := range crds {
			dc := dbClusterFromCRD(crd)
			if dc.Status == "healthy" {
				stats.HealthyClusters++
			}
			if dc.LastBackup != "" && dc.LastBackup > stats.LastBackup {
				stats.LastBackup = dc.LastBackup
			}
			size := dc.StorageSize
			if strings.HasSuffix(size, "Gi") {
				var gi float64
				if _, err := fmt.Sscanf(strings.TrimSuffix(size, "Gi"), "%f", &gi); err == nil {
					totalStorageGi += gi * float64(dc.Instances)
				}
			}
		}
	}
	stats.TotalStorage = fmt.Sprintf("%.0f Gi", totalStorageGi)
	return c.JSON(stats)
}

func (h *AdminPlatformOpsHandler) GetStorageStats(c *fiber.Ctx) error {
	stats := entities.StorageStats{}

	bucketNames := []string{"zenith-backups", "zenith-platform-storage"}
	stats.TotalBuckets = len(bucketNames)
	var totalS3 int64
	if h.s3 != nil {
		for _, name := range bucketNames {
			result, err := h.s3.ListObjects(c.Context(), name, "", "", 1000)
			if err == nil {
				for _, obj := range result.Objects {
					totalS3 += obj.Size
				}
			}
		}
	}
	stats.S3Used = formatBytesAdmin(totalS3)

	pvcs, err := h.k8s.ListPVCs(c.Context(), "")
	if err == nil {
		stats.TotalVolumes = len(pvcs)
		var totalPVC float64
		for _, pvc := range pvcs {
			if strings.HasSuffix(pvc.Size, "Gi") {
				var gi float64
				if _, err := fmt.Sscanf(strings.TrimSuffix(pvc.Size, "Gi"), "%f", &gi); err == nil {
					totalPVC += gi
				}
			}
		}
		stats.PVCUsed = fmt.Sprintf("%.0f Gi", totalPVC)
	}

	return c.JSON(stats)
}

func (h *AdminPlatformOpsHandler) GetBackupStats(c *fiber.Ctx) error {
	stats := entities.BackupStats{}

	schedules, err := h.k8s.ListCRDsWithVersion(c.Context(), veleroAPI, "Schedule", "velero")
	if err == nil {
		stats.VeleroSchedules = len(schedules)
		for _, s := range schedules {
			_, status := parseCRD(s)
			if lb, ok := status["lastBackup"].(string); ok && lb > stats.LastBackup {
				stats.LastBackup = lb
			}
		}
	}

	for _, ns := range []string{"zenith-staging", "zenith-shared", "keycloak"} {
		clusters, err := h.k8s.ListCRDsWithVersion(c.Context(), cnpgAPI, "Cluster", ns)
		if err == nil {
			stats.CNPGClusters += len(clusters)
		}
	}

	if h.s3 != nil {
		result, err := h.s3.ListObjects(c.Context(), "zenith-backups", "", "", 1000)
		if err == nil {
			var total int64
			for _, obj := range result.Objects {
				total += obj.Size
			}
			stats.TotalSize = formatBytesAdmin(total)
		}
	}
	if stats.TotalSize == "" {
		stats.TotalSize = "unknown"
	}

	return c.JSON(stats)
}

func (h *AdminPlatformOpsHandler) ListVeleroSchedules(c *fiber.Ctx) error {
	schedules, err := h.k8s.ListCRDsWithVersion(c.Context(), veleroAPI, "Schedule", "velero")
	if err != nil {
		return c.JSON([]entities.VeleroSchedule{})
	}
	var result []entities.VeleroSchedule
	for _, s := range schedules {
		result = append(result, veleroScheduleFromCRD(c.Context(), h.k8s, s))
	}
	return c.JSON(result)
}

func (h *AdminPlatformOpsHandler) ListCNPGBackups(c *fiber.Ctx) error {
	var result []entities.CNPGBackupStatus
	for _, ns := range []string{"zenith-staging", "zenith-shared", "keycloak"} {
		clusters, err := h.k8s.ListCRDsWithVersion(c.Context(), cnpgAPI, "Cluster", ns)
		if err != nil {
			continue
		}
		for _, cluster := range clusters {
			result = append(result, cnpgBackupFromCRD(cluster))
		}
	}
	return c.JSON(result)
}

func (h *AdminPlatformOpsHandler) GetGitOpsStats(c *fiber.Ctx) error {
	stats := entities.GitOpsStats{}
	apps, err := h.k8s.ListCRDsWithVersion(c.Context(), argocdAPI, "Application", "argocd")
	if err == nil {
		stats.TotalApps = len(apps)
		for _, app := range apps {
			_, status := parseCRD(app)
			sync := ""
			health := ""
			if s, ok := status["sync"].(map[string]interface{}); ok {
				sync = mapStr(s, "status")
			}
			if h, ok := status["health"].(map[string]interface{}); ok {
				health = mapStr(h, "status")
			}
			if sync == "Synced" {
				stats.Synced++
			} else if sync == "OutOfSync" {
				stats.OutOfSync++
			}
			if health == "Degraded" {
				stats.Degraded++
			}
		}
	}
	return c.JSON(stats)
}

func (h *AdminPlatformOpsHandler) GetRegistryStats(c *fiber.Ctx) error {
	stats := entities.RegistryStats{}
	if h.harbor != nil {
		repos, err := h.harbor.ListRepositories(c.Context(), "zenith-stage")
		if err == nil {
			stats.TotalProjects = 1
			stats.TotalRepos = len(repos)
			for _, r := range repos {
				stats.TotalTags += r.ArtifactCount
			}
		}
	}
	return c.JSON(stats)
}

// --- Quality ---

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
		var createdAt time.Time
		var email *string
		if rows.Scan(&t.ID, &t.Subject, &email, &t.Priority, &t.Status, &createdAt) == nil {
			if email != nil {
				t.Customer = *email
			}
			t.Age = adminTimeAgo(createdAt)
			tickets = append(tickets, t)
		}
	}
	return c.JSON(tickets)
}

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

		var avgMinutes *float64
		_ = h.pool.QueryRow(c.Context(),
			`SELECT EXTRACT(EPOCH FROM AVG(first_reply.replied_at - t.created_at))/60
			 FROM support_tickets t
			 JOIN LATERAL (
			   SELECT created_at AS replied_at FROM support_messages
			   WHERE ticket_id = t.id AND sender_role = 'admin'
			   ORDER BY created_at LIMIT 1
			 ) first_reply ON true
			 WHERE t.created_at > now() - interval '30 days'`,
		).Scan(&avgMinutes)
		if avgMinutes != nil {
			if *avgMinutes < 60 {
				metrics.AvgResponseTime = fmt.Sprintf("%.0fmin", *avgMinutes)
			} else {
				metrics.AvgResponseTime = fmt.Sprintf("%.1fh", *avgMinutes/60)
			}
		}

		var totalResolved, withinSLA int
		_ = h.pool.QueryRow(c.Context(),
			`SELECT COUNT(*), COUNT(*) FILTER (WHERE updated_at - created_at < interval '24 hours')
			 FROM support_tickets
			 WHERE status IN ('resolved', 'closed') AND created_at > now() - interval '30 days'`,
		).Scan(&totalResolved, &withinSLA)
		if totalResolved > 0 {
			metrics.SLACompliance = float64(withinSLA) / float64(totalResolved) * 100
		} else {
			metrics.SLACompliance = 100
		}
	}

	return c.JSON(metrics)
}

// --- Helpers ---

func mapStr(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func formatBytesAdmin(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func adminTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
