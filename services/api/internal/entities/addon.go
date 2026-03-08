package entities

// AddOnCategory groups add-ons by type.
type AddOnCategory string

const (
	AddOnCategorySupport  AddOnCategory = "support"
	AddOnCategoryCompute  AddOnCategory = "compute"
	AddOnCategoryStorage  AddOnCategory = "storage"
	AddOnCategorySecurity AddOnCategory = "security"
	AddOnCategoryNetwork  AddOnCategory = "network"
)

// AddOn describes an available marketplace add-on.
type AddOn struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    AddOnCategory `json:"category"`
	PriceCents  int           `json:"price_cents"`  // monthly price in cents
	MinTier     PlanTier      `json:"min_tier"`      // minimum plan tier required
	Features    []string      `json:"features"`
	Popular     bool          `json:"popular"`
}

// AddOnSubscription represents a user's active add-on subscription.
type AddOnSubscription struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	AddOnID   string `json:"addon_id"`
	Status    string `json:"status"` // "active", "cancelled", "pending"
	Timestamps
}

// AvailableAddOns returns the marketplace catalog.
func AvailableAddOns() []AddOn {
	return []AddOn{
		{
			ID: "gold-support", Name: "Gold Support", Category: AddOnCategorySupport,
			Description: "Priority email & chat support with 4-hour response SLA",
			PriceCents: 4900, MinTier: PlanPro, Popular: true,
			Features: []string{"4-hour response SLA", "Priority email support", "Dedicated chat channel", "Monthly health check"},
		},
		{
			ID: "platinum-support", Name: "Platinum Support", Category: AddOnCategorySupport,
			Description: "24/7 phone & video support with 1-hour response SLA and dedicated engineer",
			PriceCents: 19900, MinTier: PlanTeam,
			Features: []string{"1-hour response SLA", "24/7 phone support", "Dedicated support engineer", "Weekly review calls", "Custom runbooks"},
		},
		{
			ID: "extra-compute-small", Name: "Extra Compute (Small)", Category: AddOnCategoryCompute,
			Description: "Add 2 vCPU and 4GB RAM capacity across your apps",
			PriceCents: 2900, MinTier: PlanPro,
			Features: []string{"+2 vCPU capacity", "+4 GB RAM", "Shared across all apps", "Instant activation"},
		},
		{
			ID: "extra-compute-large", Name: "Extra Compute (Large)", Category: AddOnCategoryCompute,
			Description: "Add 8 vCPU and 16GB RAM capacity across your apps",
			PriceCents: 9900, MinTier: PlanTeam, Popular: true,
			Features: []string{"+8 vCPU capacity", "+16 GB RAM", "Shared across all apps", "Instant activation"},
		},
		{
			ID: "extra-storage-50gb", Name: "Extra Storage (50GB)", Category: AddOnCategoryStorage,
			Description: "Add 50GB S3-compatible object storage",
			PriceCents: 990, MinTier: PlanPro,
			Features: []string{"+50 GB S3 storage", "S3-compatible API", "Automatic backups"},
		},
		{
			ID: "extra-storage-500gb", Name: "Extra Storage (500GB)", Category: AddOnCategoryStorage,
			Description: "Add 500GB S3-compatible object storage",
			PriceCents: 4900, MinTier: PlanPro,
			Features: []string{"+500 GB S3 storage", "S3-compatible API", "Automatic backups", "CDN integration"},
		},
		{
			ID: "waf-advanced", Name: "Advanced WAF", Category: AddOnCategorySecurity,
			Description: "Custom WAF rules, rate limiting, and DDoS protection for your apps",
			PriceCents: 4900, MinTier: PlanTeam,
			Features: []string{"Custom WAF rules", "Advanced rate limiting", "DDoS protection", "IP reputation filtering", "Bot detection"},
		},
		{
			ID: "private-networking", Name: "Private Networking", Category: AddOnCategoryNetwork,
			Description: "Dedicated VLAN with private IPs between your services",
			PriceCents: 2900, MinTier: PlanTeam,
			Features: []string{"Private VLAN", "Internal DNS", "No egress charges", "Encrypted links"},
		},
		{
			ID: "managed-ssl", Name: "Managed SSL Certificates", Category: AddOnCategorySecurity,
			Description: "Automated SSL certificate management with custom CA support",
			PriceCents: 990, MinTier: PlanPro,
			Features: []string{"Auto-renewal", "Custom CA support", "Wildcard certificates", "Certificate monitoring"},
		},
	}
}
