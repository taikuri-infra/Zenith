package entities

import "time"

// AutoscalerConfig holds the configuration for the Hetzner autoscaler.
type AutoscalerConfig struct {
	MinNodes       int     `json:"min_nodes"`
	MaxNodes       int     `json:"max_nodes"`
	ScaleUpCPU     float64 `json:"scale_up_cpu"`     // threshold % to trigger scale-up
	ScaleUpRAM     float64 `json:"scale_up_ram"`     // threshold % to trigger scale-up
	ScaleDownCPU   float64 `json:"scale_down_cpu"`   // threshold % to trigger scale-down
	ScaleDownRAM   float64 `json:"scale_down_ram"`   // threshold % to trigger scale-down
	CooldownUp     time.Duration `json:"cooldown_up"`     // min time between scale-ups
	CooldownDown   time.Duration `json:"cooldown_down"`   // min time between scale-downs
	BudgetCapEUR   float64 `json:"budget_cap_eur"`   // max monthly spend
	ServerType     string  `json:"server_type"`      // e.g. cpx31
	Location       string  `json:"location"`         // e.g. fsn1
}

// HetznerNode represents a Hetzner Cloud server managed by the autoscaler.
type HetznerNode struct {
	ServerID   int64     `json:"server_id"`
	Name       string    `json:"name"`
	IP         string    `json:"ip"`
	Status     string    `json:"status"` // running, provisioning, draining, deleting
	ServerType string    `json:"server_type"`
	CPUCores   int       `json:"cpu_cores"`
	RAMMB      int       `json:"ram_mb"`
	MonthlyCost float64  `json:"monthly_cost"`
	CreatedAt  time.Time `json:"created_at"`
}

// AutoscaleAction describes a scale event type.
type AutoscaleAction string

const (
	AutoscaleActionScaleUp   AutoscaleAction = "scale_up"
	AutoscaleActionScaleDown AutoscaleAction = "scale_down"
)

// AutoscaleEvent records a single scaling action.
type AutoscaleEvent struct {
	ID         string          `json:"id"`
	Timestamp  time.Time       `json:"timestamp"`
	Action     AutoscaleAction `json:"action"`
	OldCount   int             `json:"old_count"`
	NewCount   int             `json:"new_count"`
	Reason     string          `json:"reason"`
	ServerName string          `json:"server_name"`
}

// NodeMetrics holds per-node resource utilization.
type NodeMetrics struct {
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpu_percent"`
	RAMPercent float64 `json:"ram_percent"`
}

// AutoscalerStatus represents the current autoscaler state.
type AutoscalerStatus struct {
	Enabled       bool      `json:"enabled"`
	NodeCount     int       `json:"node_count"`
	MinNodes      int       `json:"min_nodes"`
	MaxNodes      int       `json:"max_nodes"`
	CPUPercent    float64   `json:"cpu_percent"`
	RAMPercent    float64   `json:"ram_percent"`
	BudgetCapEUR  float64   `json:"budget_cap_eur"`
	BudgetUsedEUR float64   `json:"budget_used_eur"`
	LastScaleUp   time.Time `json:"last_scale_up"`
	LastScaleDown time.Time `json:"last_scale_down"`
	LastCheckAt   time.Time `json:"last_check_at"`
}
