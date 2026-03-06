package entities

import "time"

// PlanTier represents the pricing tier.
type PlanTier string

const (
	PlanFree       PlanTier = "free"
	PlanPro        PlanTier = "pro"
	PlanTeam       PlanTier = "team"
	PlanEnterprise PlanTier = "enterprise"
)

// PlanLimits defines the resource ceilings for a plan tier.
type PlanLimits struct {
	MaxApps         int `json:"max_apps"`
	MaxDatabases    int `json:"max_databases"`
	MaxDBSizeMB     int `json:"max_db_size_mb"`
	MaxAuthUsers    int `json:"max_auth_users"`
	MaxStorageMB    int `json:"max_storage_mb"`
	MaxBuckets      int `json:"max_buckets"`
	MaxCPUMillis    int `json:"max_cpu_millis"`    // millicores
	MaxRAMMB        int `json:"max_ram_mb"`
	MaxTeamMembers  int `json:"max_team_members"`
	BackupsEnabled  bool `json:"backups_enabled"`
	CustomDomain    bool `json:"custom_domain"`
	AlwaysOn        bool `json:"always_on"` // false = scale-to-zero after idle
	SleepAfterMins  int  `json:"sleep_after_mins"` // 0 = no sleep (always on)
}

// DefaultPlanLimits returns the limits for a given plan tier.
func DefaultPlanLimits(tier PlanTier) PlanLimits {
	switch tier {
	case PlanPro:
		return PlanLimits{
			MaxApps: 5, MaxDatabases: 3, MaxDBSizeMB: 5120,
			MaxAuthUsers: 10000, MaxStorageMB: 10240, MaxBuckets: 5,
			MaxCPUMillis: 2000, MaxRAMMB: 2048, MaxTeamMembers: 3,
			BackupsEnabled: true, CustomDomain: true, AlwaysOn: true, SleepAfterMins: 0,
		}
	case PlanTeam:
		return PlanLimits{
			MaxApps: 20, MaxDatabases: 10, MaxDBSizeMB: 20480,
			MaxAuthUsers: 100000, MaxStorageMB: 102400, MaxBuckets: 20,
			MaxCPUMillis: 4000, MaxRAMMB: 4096, MaxTeamMembers: 10,
			BackupsEnabled: true, CustomDomain: true, AlwaysOn: true, SleepAfterMins: 0,
		}
	case PlanEnterprise:
		return PlanLimits{
			MaxApps: 1000, MaxDatabases: 1000, MaxDBSizeMB: 1048576,
			MaxAuthUsers: 10000000, MaxStorageMB: 10485760, MaxBuckets: 1000,
			MaxCPUMillis: 64000, MaxRAMMB: 65536, MaxTeamMembers: 1000,
			BackupsEnabled: true, CustomDomain: true, AlwaysOn: true, SleepAfterMins: 0,
		}
	default: // Free
		return PlanLimits{
			MaxApps: 1, MaxDatabases: 1, MaxDBSizeMB: 100,
			MaxAuthUsers: 1000, MaxStorageMB: 1024, MaxBuckets: 0,
			MaxCPUMillis: 500, MaxRAMMB: 512, MaxTeamMembers: 1,
			BackupsEnabled: false, CustomDomain: false, AlwaysOn: false, SleepAfterMins: 15,
		}
	}
}

// UserPlan tracks which plan a user is on.
type UserPlan struct {
	UserID               string             `json:"user_id"`
	Tier                 PlanTier           `json:"tier"`
	Limits               PlanLimits         `json:"limits"`
	StripeSubscriptionID string             `json:"stripe_subscription_id,omitempty"`
	BillingStatus        SubscriptionStatus `json:"billing_status,omitempty"`
	CurrentPeriodEnd     *time.Time         `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end,omitempty"`
	Timestamps
}
