package entities

import "time"

// DashboardStats represents the admin dashboard overview statistics.
type DashboardStats struct {
	ClusterCount     int    `json:"clusterCount"`
	AllHealthy       bool   `json:"allHealthy"`
	TenantCount      int    `json:"tenantCount"`
	ActiveToday      int    `json:"activeToday"`
	MonthlyCost      string `json:"monthlyCost"`
	CostProvider     string `json:"costProvider"`
	UpdatesAvailable int    `json:"updatesAvailable"`
}

// Cluster represents a CAPI-managed Kubernetes cluster.
type Cluster struct {
	Name             string       `json:"name"`
	K8sVersion       string       `json:"k8sVersion"`
	Nodes            int          `json:"nodes"`
	Region           string       `json:"region"`
	Type             string       `json:"type"`
	Tenant           string       `json:"tenant,omitempty"`
	CPUPercent       int          `json:"cpuPercent"`
	RAMPercent       int          `json:"ramPercent"`
	Pods             ResourcePair `json:"pods"`
	PVCs             ResourcePair `json:"pvcs"`
	Status           string       `json:"status"`
	UpgradeAvailable string       `json:"upgradeAvailable,omitempty"`
}

// ResourcePair represents a used/total pair for pods, PVCs, etc.
type ResourcePair struct {
	Used  int `json:"used"`
	Total int `json:"total"`
}

// Tenant represents a platform tenant (project).
type Tenant struct {
	Name      string `json:"name"`
	Plan      string `json:"plan"`
	Apps      int    `json:"apps"`
	Databases int    `json:"databases"`
	CPUUsed   string `json:"cpuUsed"`
	CPULimit  string `json:"cpuLimit"`
	RAMUsed   string `json:"ramUsed"`
	RAMLimit  string `json:"ramLimit"`
	Status    string `json:"status"`
}

// Module represents an installable platform module (Helm chart / operator).
type Module struct {
	Name        string `json:"name"`
	Installed   string `json:"installed"`
	Latest      string `json:"latest"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

// AuditEntry represents an audit-log entry.
type AuditEntry struct {
	Time    string `json:"time"`
	Actor   string `json:"actor"`
	Action  string `json:"action"`
	Cluster string `json:"cluster,omitempty"`
}

// PlatformUpdate represents available platform version information.
type PlatformUpdate struct {
	Version         string   `json:"version"`
	Current         string   `json:"current"`
	ReleasedAt      string   `json:"releasedAt"`
	Features        []string `json:"features"`
	BreakingChanges bool     `json:"breakingChanges"`
}

// UpdateHistoryEntry represents a past platform update.
type UpdateHistoryEntry struct {
	Version string `json:"version"`
	Date    string `json:"date"`
	Status  string `json:"status"`
}

// InfraNode represents a single infrastructure resource group.
type InfraNode struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Count       int    `json:"count"`
	Cluster     string `json:"cluster"`
	MonthlyCost string `json:"monthlyCost"`
}

// InfraOverview represents a summary of all infrastructure resources.
type InfraOverview struct {
	Servers       int         `json:"servers"`
	Volumes       int         `json:"volumes"`
	VolumeSize    string      `json:"volumeSize"`
	LoadBalancers int         `json:"loadBalancers"`
	LBPublic      int         `json:"lbPublic"`
	LBInternal    int         `json:"lbInternal"`
	MonthlyCost   string      `json:"monthlyCost"`
	Resources     []InfraNode `json:"resources"`
}

// PlatformState represents the overall platform state.
type PlatformState struct {
	PlatformVersion       string `json:"platformVersion"`
	UpdateAvailable       string `json:"updateAvailable,omitempty"`
	InstalledDate         string `json:"installedDate"`
	InstalledDaysAgo      int    `json:"installedDaysAgo"`
	ManagementK8sVersion  string `json:"managementK8sVersion"`
	ManagementK8sUpToDate bool   `json:"managementK8sUpToDate"`
	Domain                string `json:"domain"`
	WildcardTLS           bool   `json:"wildcardTls"`
}

// PlatformSettings represents configurable platform settings.
type PlatformSettings struct {
	PlatformName  string `json:"platformName"`
	BaseDomain    string `json:"baseDomain"`
	Provider      string `json:"provider"`
	DefaultRegion string `json:"defaultRegion"`
	RegionLabel   string `json:"regionLabel"`
	AutoBackups   bool   `json:"autoBackups"`
	RetentionDays int    `json:"retentionDays"`
}

// ClusterTimestamps can be embedded when clusters need timestamps.
type ClusterTimestamps struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}
