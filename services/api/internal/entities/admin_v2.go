package entities

import "time"

// --- Analytics ---

type RevenueStats struct {
	MRR              float64          `json:"mrr"`
	ARR              float64          `json:"arr"`
	ChurnRate        float64          `json:"churnRate"`
	LTV              float64          `json:"ltv"`
	RevenueByPlan    []PlanRevenue    `json:"revenueByPlan"`
	MonthlyTrend     []MonthlyRevenue `json:"monthlyTrend"`
}

type PlanRevenue struct {
	Plan    string  `json:"plan"`
	Revenue float64 `json:"revenue"`
	Count   int     `json:"count"`
}

type MonthlyRevenue struct {
	Month       string  `json:"month"`
	NewRevenue  float64 `json:"newRevenue"`
	ChurnedRevenue float64 `json:"churnedRevenue"`
	TotalMRR    float64 `json:"totalMrr"`
}

type GrowthStats struct {
	TotalUsers       int              `json:"totalUsers"`
	NewThisMonth     int              `json:"newThisMonth"`
	ChurnedThisMonth int              `json:"churnedThisMonth"`
	MonthlyGrowth    []MonthlyGrowth  `json:"monthlyGrowth"`
	Conversions      ConversionStats  `json:"conversions"`
}

type MonthlyGrowth struct {
	Month   string `json:"month"`
	New     int    `json:"new"`
	Churned int    `json:"churned"`
	Total   int    `json:"total"`
}

type ConversionStats struct {
	FreeToProRate  float64 `json:"freeToProRate"`
	ProToTeamRate  float64 `json:"proToTeamRate"`
	TrialToPayRate float64 `json:"trialToPayRate"`
}

type UsageStats struct {
	TopFeatures    []FeatureUsage `json:"topFeatures"`
	AvgAppsPerUser float64        `json:"avgAppsPerUser"`
	AvgDBsPerUser  float64        `json:"avgDbsPerUser"`
}

type FeatureUsage struct {
	Feature    string `json:"feature"`
	UsageCount int    `json:"usageCount"`
	UserCount  int    `json:"userCount"`
}

type CohortData struct {
	Cohort     string    `json:"cohort"`
	Month      string    `json:"month"`
	Retained   int       `json:"retained"`
	Total      int       `json:"total"`
	Percentage float64   `json:"percentage"`
}

// --- CRM ---

type CRMPipeline struct {
	Stages []PipelineStage `json:"stages"`
}

type PipelineStage struct {
	Name      string            `json:"name"`
	Count     int               `json:"count"`
	Customers []PipelineCustomer `json:"customers"`
}

type PipelineCustomer struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	Plan        string  `json:"plan"`
	HealthScore int     `json:"healthScore"`
	MRR         float64 `json:"mrr"`
	LastLogin   string  `json:"lastLogin,omitempty"`
}

type HealthScore struct {
	UserID      string `json:"userId"`
	Score       int    `json:"score"`
	UsageScore  int    `json:"usageScore"`
	SupportScore int   `json:"supportScore"`
	LoginScore  int    `json:"loginScore"`
	RiskLevel   string `json:"riskLevel"`
}

type CustomerNote struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	AuthorID  string    `json:"authorId"`
	AuthorName string   `json:"authorName,omitempty"`
	Note      string    `json:"note"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// --- Services ---

type ServiceStatus struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	Version    string `json:"version,omitempty"`
	Replicas   int    `json:"replicas"`
	Ready      int    `json:"ready"`
	Restarts   int    `json:"restarts"`
	Uptime     string `json:"uptime,omitempty"`
	CPUUsage   string `json:"cpuUsage,omitempty"`
	MemUsage   string `json:"memUsage,omitempty"`
	LastRestart string `json:"lastRestart,omitempty"`
}

type ServiceDetail struct {
	ServiceStatus
	Pods      []ServicePod   `json:"pods"`
	Events    []ServiceEvent `json:"events"`
}

type ServicePod struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Node      string `json:"node,omitempty"`
	Restarts  int    `json:"restarts"`
	Age       string `json:"age"`
	CPU       string `json:"cpu,omitempty"`
	Memory    string `json:"memory,omitempty"`
}

type ServiceEvent struct {
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
	Time    string `json:"time"`
}

// --- Databases (Admin view) ---

type AdminDatabaseCluster struct {
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	Status            string `json:"status"`
	Instances         int    `json:"instances"`
	ReadyInstances    int    `json:"readyInstances"`
	StorageSize       string `json:"storageSize"`
	WALArchiving      string `json:"walArchiving"`
	LastBackup        string `json:"lastBackup,omitempty"`
	RecoveryWindow    string `json:"recoveryWindow,omitempty"`
	PostgresVersion   string `json:"postgresVersion,omitempty"`
}

// --- Storage (Admin view) ---

type AdminS3Bucket struct {
	Name         string `json:"name"`
	Size         string `json:"size"`
	ObjectCount  int64  `json:"objectCount"`
	LastModified string `json:"lastModified,omitempty"`
}

type AdminVolume struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Size         string `json:"size"`
	Status       string `json:"status"`
	StorageClass string `json:"storageClass"`
}

// --- Networking ---

type AdminDNSRecord struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl"`
}

type AdminRoute struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Service   string `json:"service"`
	TLS       bool   `json:"tls"`
	Source    string `json:"source"`
}

type AdminCertificate struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	DnsNames   []string `json:"dnsNames"`
	Issuer     string `json:"issuer"`
	Status     string `json:"status"`
	ExpiresAt  string `json:"expiresAt,omitempty"`
	RenewAt    string `json:"renewAt,omitempty"`
}

// --- Observability ---

type GrafanaDashboard struct {
	UID   string `json:"uid"`
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

type AlertInfo struct {
	Name      string   `json:"name"`
	State     string   `json:"state"`
	Severity  string   `json:"severity"`
	Summary   string   `json:"summary,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	ActiveAt  string   `json:"activeAt,omitempty"`
	Value     string   `json:"value,omitempty"`
}

type AdminAlertRule struct {
	Name      string `json:"name"`
	Group     string `json:"group"`
	Query     string `json:"query"`
	Duration  string `json:"duration"`
	Severity  string `json:"severity"`
	State     string `json:"state"`
}

type LogQueryResult struct {
	Streams []LogStream `json:"streams"`
}

type LogStream struct {
	Labels map[string]string `json:"labels"`
	Values [][]string        `json:"values"`
}

type TraceInfo struct {
	TraceID    string `json:"traceId"`
	RootService string `json:"rootService"`
	RootName    string `json:"rootName"`
	Duration    string `json:"duration"`
	SpanCount   int    `json:"spanCount"`
	StartTime   string `json:"startTime"`
	Status      string `json:"status,omitempty"`
}

// --- Security ---

type SecurityPosture struct {
	OverallScore      int     `json:"overallScore"`
	MFAAdoption       float64 `json:"mfaAdoption"`
	ImageVulns        VulnSummary `json:"imageVulns"`
	PolicyViolations  int     `json:"policyViolations"`
	FalcoAlerts       int     `json:"falcoAlerts"`
	CertWarnings      int     `json:"certWarnings"`
	FailedLogins24h   int     `json:"failedLogins24h"`
	OpenIssues        int     `json:"openIssues"`
}

type VulnSummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

type PolicyInfo struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Action     string `json:"action"`
	Status     string `json:"status"`
	Violations int    `json:"violations"`
}

type FalcoAlert struct {
	Time     string `json:"time"`
	Priority string `json:"priority"`
	Rule     string `json:"rule"`
	Output   string `json:"output"`
	Source   string `json:"source"`
}

type ImageScanResult struct {
	Repository  string      `json:"repository"`
	Tag         string      `json:"tag"`
	Digest      string      `json:"digest,omitempty"`
	ScanStatus  string      `json:"scanStatus"`
	Vulns       VulnSummary `json:"vulns"`
	LastScanned string      `json:"lastScanned,omitempty"`
}

type AdminSession struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	Email     string `json:"email,omitempty"`
	IPAddress string `json:"ipAddress"`
	UserAgent string `json:"userAgent,omitempty"`
	Device    string `json:"device,omitempty"`
	LastSeen  string `json:"lastSeen"`
	CreatedAt string `json:"createdAt"`
}

// --- Backups ---

type AdminBackupOverview struct {
	VeleroSchedules []VeleroSchedule    `json:"veleroSchedules"`
	CNPGBackups     []CNPGBackupStatus  `json:"cnpgBackups"`
}

type VeleroSchedule struct {
	Name         string `json:"name"`
	Schedule     string `json:"schedule"`
	LastBackup   string `json:"lastBackup,omitempty"`
	LastStatus   string `json:"lastStatus"`
	BackupCount  int    `json:"backupCount"`
}

type CNPGBackupStatus struct {
	Cluster       string `json:"cluster"`
	Namespace     string `json:"namespace"`
	LastBackup    string `json:"lastBackup,omitempty"`
	Status        string `json:"status"`
	WALArchiving  string `json:"walArchiving"`
	RetentionDays int    `json:"retentionDays"`
}

// --- GitOps ---

type ArgoApp struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Status     string `json:"status"`
	Health     string `json:"health"`
	SyncStatus string `json:"syncStatus"`
	Revision   string `json:"revision,omitempty"`
	LastSync   string `json:"lastSync,omitempty"`
	RepoURL    string `json:"repoUrl,omitempty"`
	Path       string `json:"path,omitempty"`
}

type ArgoDeployment struct {
	Revision  string `json:"revision"`
	Status    string `json:"status"`
	StartedAt string `json:"startedAt"`
	Message   string `json:"message,omitempty"`
}

// --- Registry ---

type RegistryProject struct {
	Name         string `json:"name"`
	RepoCount    int    `json:"repoCount"`
	StorageUsed  string `json:"storageUsed,omitempty"`
	StorageQuota string `json:"storageQuota,omitempty"`
	Public       bool   `json:"public"`
}

type RegistryRepo struct {
	Name      string `json:"name"`
	TagCount  int    `json:"tagCount"`
	PullCount int    `json:"pullCount"`
	PushTime  string `json:"pushTime,omitempty"`
}

// --- Admin RBAC ---

type AdminRole struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Email       string    `json:"email,omitempty"`
	Name        string    `json:"name,omitempty"`
	AdminRole   string    `json:"adminRole"`
	Permissions []string  `json:"permissions"`
	GrantedBy   string    `json:"grantedBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// AdminPermissionGroup constants
const (
	AdminPermWarRoom        = "war_room"
	AdminPermAnalytics      = "analytics"
	AdminPermCustomers      = "customers"
	AdminPermCRM            = "crm"
	AdminPermSupport        = "support_tickets"
	AdminPermQuality        = "quality"
	AdminPermServices       = "services"
	AdminPermClusters       = "clusters"
	AdminPermInfrastructure = "infrastructure"
	AdminPermObservability  = "observability"
	AdminPermSecurity       = "security"
	AdminPermModules        = "modules"
	AdminPermBackups        = "backups"
	AdminPermGitOps         = "gitops"
	AdminPermRegistry       = "registry"
	AdminPermAdminSettings  = "admin_settings"
)

// DefaultAdminPermissions returns permissions for each admin role level
func DefaultAdminPermissions(role string) []string {
	all := []string{
		AdminPermWarRoom, AdminPermAnalytics, AdminPermCustomers,
		AdminPermCRM, AdminPermSupport, AdminPermQuality,
		AdminPermServices, AdminPermClusters, AdminPermInfrastructure,
		AdminPermObservability, AdminPermSecurity, AdminPermModules,
		AdminPermBackups, AdminPermGitOps, AdminPermRegistry,
		AdminPermAdminSettings,
	}

	switch role {
	case "owner":
		return all
	case "admin":
		return all[:len(all)-1] // everything except admin_settings
	case "support":
		return []string{
			AdminPermWarRoom, AdminPermAnalytics, AdminPermCustomers,
			AdminPermCRM, AdminPermSupport, AdminPermQuality,
		}
	case "viewer":
		return []string{
			AdminPermWarRoom, AdminPermAnalytics, AdminPermCustomers,
			AdminPermCRM, AdminPermSupport, AdminPermQuality,
			AdminPermServices, AdminPermClusters, AdminPermInfrastructure,
			AdminPermObservability, AdminPermSecurity,
			AdminPermBackups, AdminPermGitOps, AdminPermRegistry,
		}
	default:
		return []string{}
	}
}

// --- Stats aggregation types ---

type DatabaseStats struct {
	TotalClusters   int    `json:"totalClusters"`
	HealthyClusters int    `json:"healthyClusters"`
	TotalStorage    string `json:"totalStorage"`
	LastBackup      string `json:"lastBackup,omitempty"`
}

type StorageStats struct {
	TotalBuckets int    `json:"totalBuckets"`
	S3Used       string `json:"s3Used"`
	TotalVolumes int    `json:"totalVolumes"`
	PVCUsed      string `json:"pvcUsed"`
}

type BackupStats struct {
	VeleroSchedules int    `json:"veleroSchedules"`
	CNPGClusters    int    `json:"cnpgClusters"`
	LastBackup      string `json:"lastBackup,omitempty"`
	TotalSize       string `json:"totalSize"`
}

type GitOpsStats struct {
	TotalApps int `json:"totalApps"`
	Synced    int `json:"synced"`
	OutOfSync int `json:"outOfSync"`
	Degraded  int `json:"degraded"`
}

type RegistryStats struct {
	TotalProjects int    `json:"totalProjects"`
	TotalRepos    int    `json:"totalRepos"`
	TotalTags     int    `json:"totalTags"`
	StorageUsed   string `json:"storageUsed"`
	StorageQuota  string `json:"storageQuota"`
}

type AlertStats struct {
	Firing        int `json:"firing"`
	Pending       int `json:"pending"`
	ResolvedToday int `json:"resolvedToday"`
	TotalRules    int `json:"totalRules"`
}

type WafStats struct {
	TotalPolicies   int `json:"totalPolicies"`
	Enforcing       int `json:"enforcing"`
	Auditing        int `json:"auditing"`
	TotalViolations int `json:"totalViolations"`
}

type ImageScanStats struct {
	TotalImages   int `json:"totalImages"`
	CleanImages   int `json:"cleanImages"`
	CriticalCount int `json:"criticalCount"`
	HighCount     int `json:"highCount"`
}

type QualityTicket struct {
	ID       string `json:"id"`
	Subject  string `json:"subject"`
	Customer string `json:"customer"`
	Priority string `json:"priority"`
	Status   string `json:"status"`
	Age      string `json:"age"`
}

// --- Quality / SLA ---

type QualityMetrics struct {
	AvgResponseTime    string          `json:"avgResponseTime"`
	AvgResolutionTime  string          `json:"avgResolutionTime"`
	OpenTickets        int             `json:"openTickets"`
	ResolvedThisWeek   int             `json:"resolvedThisWeek"`
	CSAT               float64         `json:"csat"`
	SLACompliance      float64         `json:"slaCompliance"`
	TicketsByPriority  map[string]int  `json:"ticketsByPriority"`
	TicketsByCategory  map[string]int  `json:"ticketsByCategory"`
	WeeklyTrend        []WeeklyTickets `json:"weeklyTrend"`
}

type WeeklyTickets struct {
	Week     string `json:"week"`
	Opened   int    `json:"opened"`
	Resolved int    `json:"resolved"`
}

// --- War Room ---

type WarRoomData struct {
	KPIs           WarRoomKPIs      `json:"kpis"`
	ServiceHealth  []ServiceStatus  `json:"serviceHealth"`
	RecentAlerts   []AlertInfo      `json:"recentAlerts"`
	ActiveTickets  []TicketSummary  `json:"activeTickets"`
}

type WarRoomKPIs struct {
	MRR             float64 `json:"mrr"`
	MRRTrend        float64 `json:"mrrTrend"`
	ActiveCustomers int     `json:"activeCustomers"`
	TotalCustomers  int     `json:"totalCustomers"`
	NewSignups      int     `json:"newSignups"`
	ChurnRate       float64 `json:"churnRate"`
	AvgResponseTime string  `json:"avgResponseTime"`
	HealthScore     int     `json:"healthScore"`
}

type TicketSummary struct {
	ID       string `json:"id"`
	Subject  string `json:"subject"`
	Priority string `json:"priority"`
	Status   string `json:"status"`
	Age      string `json:"age"`
}
