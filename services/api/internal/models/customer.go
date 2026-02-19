package models

import "time"

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
	PriceCents   int       `json:"priceCents"`
	Currency     string    `json:"currency"`
	BillingCycle string    `json:"billingCycle"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// CreatePlanInput is the request body for creating a plan.
type CreatePlanInput struct {
	Name         string `json:"name"`
	CPUCores     int    `json:"cpuCores"`
	RAMGB        int    `json:"ramGb"`
	S3TB         int    `json:"s3Tb"`
	DBStorageGB  int    `json:"dbStorageGb"`
	VolumeGB     int    `json:"volumeGb"`
	LBCount      int    `json:"lbCount"`
	PriceCents   int    `json:"priceCents"`
	Currency     string `json:"currency"`
	BillingCycle string `json:"billingCycle"`
}

// UpdatePlanInput is the request body for updating a plan.
type UpdatePlanInput struct {
	Name         *string `json:"name,omitempty"`
	CPUCores     *int    `json:"cpuCores,omitempty"`
	RAMGB        *int    `json:"ramGb,omitempty"`
	S3TB         *int    `json:"s3Tb,omitempty"`
	DBStorageGB  *int    `json:"dbStorageGb,omitempty"`
	VolumeGB     *int    `json:"volumeGb,omitempty"`
	LBCount      *int    `json:"lbCount,omitempty"`
	PriceCents   *int    `json:"priceCents,omitempty"`
	Currency     *string `json:"currency,omitempty"`
	BillingCycle *string `json:"billingCycle,omitempty"`
	Active       *bool   `json:"active,omitempty"`
}

// Customer represents a DoTech customer account.
type Customer struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Domain        string    `json:"domain"`
	PlanID        string    `json:"planId"`
	ContactEmail  string    `json:"contactEmail"`
	ContactName   string    `json:"contactName"`
	Status        string    `json:"status"`
	ClusterStatus string    `json:"clusterStatus"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Plan          *Plan     `json:"plan,omitempty"`
}

// CreateCustomerInput is the request body for creating a customer.
type CreateCustomerInput struct {
	Name         string `json:"name"`
	Domain       string `json:"domain"`
	PlanID       string `json:"planId"`
	ContactEmail string `json:"contactEmail"`
	ContactName  string `json:"contactName"`
}

// UpdateCustomerInput is the request body for updating a customer.
type UpdateCustomerInput struct {
	Name         *string `json:"name,omitempty"`
	Domain       *string `json:"domain,omitempty"`
	PlanID       *string `json:"planId,omitempty"`
	ContactEmail *string `json:"contactEmail,omitempty"`
	ContactName  *string `json:"contactName,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

// CustomerStats represents aggregate customer statistics.
type CustomerStats struct {
	TotalCustomers  int    `json:"totalCustomers"`
	ActiveCustomers int    `json:"activeCustomers"`
	MRR             string `json:"mrr"`
	NewThisMonth    int    `json:"newThisMonth"`
}
