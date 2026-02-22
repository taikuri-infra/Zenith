package entities

import "time"

// Cluster status constants.
const (
	ClusterStatusPending      = "pending"
	ClusterStatusProvisioning = "provisioning"
	ClusterStatusInstalling   = "installing"
	ClusterStatusRunning      = "running"
	ClusterStatusError        = "error"
	ClusterStatusDeleting     = "deleting"
)

// Plan represents a billing plan with resource limits.
type Plan struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	CPUCores     int       `json:"cpuCores"`
	RAMGB        int       `json:"ramGb"`
	S3TB         int       `json:"s3Tb"`
	DBStorageGB  int       `json:"dbStorageGb"`
	VolumeGB     int       `json:"volumeGb"`
	LBCount      int       `json:"lbCount"`
	StorageGB    int       `json:"storageGb"`
	S3StorageTB  int       `json:"s3StorageTb"`
	LoadBalancer int       `json:"loadBalancer"`
	PriceCents   int       `json:"priceCents"`
	Currency     string    `json:"currency"`
	BillingCycle string    `json:"billingCycle"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Customer represents a tenant on the platform.
type Customer struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Domain            string    `json:"domain"`
	PlanID            string    `json:"planId"`
	Plan              *Plan     `json:"plan,omitempty"`
	Status            string    `json:"status"`
	ContactEmail      string    `json:"contactEmail"`
	ContactName       string    `json:"contactName"`
	Notes             string    `json:"notes,omitempty"`
	CAPIClusterName   string    `json:"capiClusterName,omitempty"`
	ClusterStatus     string    `json:"clusterStatus,omitempty"`
	ClusterNodes      int       `json:"clusterNodes,omitempty"`
	ClusterRegion     string    `json:"clusterRegion,omitempty"`
	ClusterK8sVersion string    `json:"clusterK8sVersion,omitempty"`
	ClusterEndpoint   string    `json:"clusterEndpoint,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// CustomerStats holds aggregate statistics about all customers.
type CustomerStats struct {
	TotalCustomers  int    `json:"totalCustomers"`
	ActiveCustomers int    `json:"activeCustomers"`
	MRR             string `json:"mrr"`
	NewThisMonth    int    `json:"newThisMonth"`
}

// ResourceUsage represents a single metering snapshot.
type ResourceUsage struct {
	ID          string    `json:"id"`
	CustomerID  string    `json:"customerId"`
	CPUCores    float64   `json:"cpuCores"`
	RAMGB       float64   `json:"ramGb"`
	S3TB        float64   `json:"s3Tb"`
	DBStorageGB float64   `json:"dbStorageGb"`
	VolumeGB    float64   `json:"volumeGb"`
	LBCount     int       `json:"lbCount"`
	RecordedAt  time.Time `json:"recordedAt"`
}
