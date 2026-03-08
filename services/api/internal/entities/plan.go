package entities

import "time"

// PlanTier represents the pricing tier.
type PlanTier string

const (
	PlanFree       PlanTier = "free"
	PlanPro        PlanTier = "pro"
	PlanTeam       PlanTier = "team"
	PlanBusiness   PlanTier = "business"
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
	MaxTeamMembers    int `json:"max_team_members"`
	MaxGateways       int `json:"max_gateways"`
	MaxGatewayRoutes  int `json:"max_gateway_routes"`
	MaxAuthPools      int `json:"max_auth_pools"`
	MaxAuthPoolUsers  int `json:"max_auth_pool_users"`
	MaxRedisInstances    int  `json:"max_redis_instances"`    // 0 = Redis not available
	MaxRabbitMQInstances int  `json:"max_rabbitmq_instances"` // 0 = RabbitMQ not available
	MaxMongoDBInstances  int  `json:"max_mongodb_instances"`  // 0 = MongoDB not available
	MaxKafkaInstances    int  `json:"max_kafka_instances"`    // 0 = Kafka not available
	BackupsEnabled       bool `json:"backups_enabled"`
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
			MaxCPUMillis: 2000, MaxRAMMB: 2048, MaxTeamMembers: 2,
			MaxGateways: 5, MaxGatewayRoutes: 50,
			MaxAuthPools: 3, MaxAuthPoolUsers: 50000,
			MaxRedisInstances: 2, MaxRabbitMQInstances: 1, MaxMongoDBInstances: 0, MaxKafkaInstances: 0,
			BackupsEnabled: true, CustomDomain: true, AlwaysOn: true, SleepAfterMins: 0,
		}
	case PlanTeam:
		return PlanLimits{
			MaxApps: 20, MaxDatabases: 10, MaxDBSizeMB: 20480,
			MaxAuthUsers: 100000, MaxStorageMB: 102400, MaxBuckets: 20,
			MaxCPUMillis: 4000, MaxRAMMB: 4096, MaxTeamMembers: 8,
			MaxGateways: 20, MaxGatewayRoutes: 200,
			MaxAuthPools: 10, MaxAuthPoolUsers: 500000,
			MaxRedisInstances: 5, MaxRabbitMQInstances: 3, MaxMongoDBInstances: 2, MaxKafkaInstances: 0,
			BackupsEnabled: true, CustomDomain: true, AlwaysOn: true, SleepAfterMins: 0,
		}
	case PlanBusiness:
		return PlanLimits{
			MaxApps: 1000, MaxDatabases: 1000, MaxDBSizeMB: 524288,
			MaxAuthUsers: 5000000, MaxStorageMB: 5242880, MaxBuckets: 500,
			MaxCPUMillis: 32000, MaxRAMMB: 32768, MaxTeamMembers: 50,
			MaxGateways: 500, MaxGatewayRoutes: 5000,
			MaxAuthPools: 50, MaxAuthPoolUsers: 5000000,
			MaxRedisInstances: 1000, MaxRabbitMQInstances: 1000, MaxMongoDBInstances: 1000, MaxKafkaInstances: 100,
			BackupsEnabled: true, CustomDomain: true, AlwaysOn: true, SleepAfterMins: 0,
		}
	case PlanEnterprise:
		return PlanLimits{
			MaxApps: 1000, MaxDatabases: 1000, MaxDBSizeMB: 1048576,
			MaxAuthUsers: 10000000, MaxStorageMB: 10485760, MaxBuckets: 1000,
			MaxCPUMillis: 64000, MaxRAMMB: 65536, MaxTeamMembers: 1000,
			MaxGateways: 1000, MaxGatewayRoutes: 10000,
			MaxAuthPools: 100, MaxAuthPoolUsers: 10000000,
			MaxRedisInstances: 1000, MaxRabbitMQInstances: 1000, MaxMongoDBInstances: 1000, MaxKafkaInstances: 1000,
			BackupsEnabled: true, CustomDomain: true, AlwaysOn: true, SleepAfterMins: 0,
		}
	default: // Free
		return PlanLimits{
			MaxApps: 1, MaxDatabases: 1, MaxDBSizeMB: 100,
			MaxAuthUsers: 1000, MaxStorageMB: 1024, MaxBuckets: 0,
			MaxCPUMillis: 500, MaxRAMMB: 512, MaxTeamMembers: 1,
			MaxGateways: 1, MaxGatewayRoutes: 3,
			MaxAuthPools: 1, MaxAuthPoolUsers: 1000,
			MaxRedisInstances: 0, MaxRabbitMQInstances: 0, MaxMongoDBInstances: 0, MaxKafkaInstances: 0,
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
