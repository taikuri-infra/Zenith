package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Compile-time interface check.
var _ CustomerRepository = (*PostgresCustomerRepository)(nil)

// PostgresCustomerRepository persists customers and plans in PostgreSQL.
type PostgresCustomerRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresCustomerRepository creates a new PostgresCustomerRepository.
func NewPostgresCustomerRepository(pool *pgxpool.Pool) *PostgresCustomerRepository {
	return &PostgresCustomerRepository{pool: pool}
}

// ---------- Plans ----------

func (r *PostgresCustomerRepository) CreatePlan(ctx context.Context, input *models.CreatePlanInput) (*models.Plan, error) {
	now := time.Now()
	id := uuid.New().String()

	currency := input.Currency
	if currency == "" {
		currency = "EUR"
	}
	billingCycle := input.BillingCycle
	if billingCycle == "" {
		billingCycle = "monthly"
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO plans (id, name, cpu_cores, ram_gb, s3_tb, db_storage_gb, volume_gb, lb_count, price_cents, currency, billing_cycle, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, true, $12, $13)`,
		id, input.Name, input.CPUCores, input.RAMGB, input.S3TB, input.DBStorageGB,
		input.VolumeGB, input.LBCount, input.PriceCents, currency, billingCycle, now, now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("plan name already exists")
		}
		return nil, fmt.Errorf("insert plan: %w", err)
	}

	return &models.Plan{
		ID: id, Name: input.Name, CPUCores: input.CPUCores, RAMGB: input.RAMGB,
		S3TB: input.S3TB, DBStorageGB: input.DBStorageGB, VolumeGB: input.VolumeGB,
		LBCount: input.LBCount, PriceCents: input.PriceCents, Currency: currency,
		BillingCycle: billingCycle, Active: true, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (r *PostgresCustomerRepository) GetPlan(ctx context.Context, id string) (*models.Plan, error) {
	var p models.Plan
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, cpu_cores, ram_gb, s3_tb, db_storage_gb, volume_gb, lb_count, price_cents, currency, billing_cycle, active, created_at, updated_at
		 FROM plans WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.CPUCores, &p.RAMGB, &p.S3TB, &p.DBStorageGB, &p.VolumeGB,
		&p.LBCount, &p.PriceCents, &p.Currency, &p.BillingCycle, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("plan not found")
		}
		return nil, fmt.Errorf("get plan: %w", err)
	}
	return &p, nil
}

func (r *PostgresCustomerRepository) ListPlans(ctx context.Context) ([]models.Plan, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, cpu_cores, ram_gb, s3_tb, db_storage_gb, volume_gb, lb_count, price_cents, currency, billing_cycle, active, created_at, updated_at
		 FROM plans ORDER BY price_cents ASC`)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer rows.Close()

	var plans []models.Plan
	for rows.Next() {
		var p models.Plan
		if err := rows.Scan(&p.ID, &p.Name, &p.CPUCores, &p.RAMGB, &p.S3TB, &p.DBStorageGB, &p.VolumeGB,
			&p.LBCount, &p.PriceCents, &p.Currency, &p.BillingCycle, &p.Active, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}
		plans = append(plans, p)
	}
	if plans == nil {
		plans = []models.Plan{}
	}
	return plans, rows.Err()
}

func (r *PostgresCustomerRepository) UpdatePlan(ctx context.Context, id string, input *models.UpdatePlanInput) (*models.Plan, error) {
	// Read current, merge, write back
	plan, err := r.GetPlan(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		plan.Name = *input.Name
	}
	if input.CPUCores != nil {
		plan.CPUCores = *input.CPUCores
	}
	if input.RAMGB != nil {
		plan.RAMGB = *input.RAMGB
	}
	if input.S3TB != nil {
		plan.S3TB = *input.S3TB
	}
	if input.DBStorageGB != nil {
		plan.DBStorageGB = *input.DBStorageGB
	}
	if input.VolumeGB != nil {
		plan.VolumeGB = *input.VolumeGB
	}
	if input.LBCount != nil {
		plan.LBCount = *input.LBCount
	}
	if input.PriceCents != nil {
		plan.PriceCents = *input.PriceCents
	}
	if input.Currency != nil {
		plan.Currency = *input.Currency
	}
	if input.BillingCycle != nil {
		plan.BillingCycle = *input.BillingCycle
	}
	if input.Active != nil {
		plan.Active = *input.Active
	}
	plan.UpdatedAt = time.Now()

	_, err = r.pool.Exec(ctx,
		`UPDATE plans SET name=$1, cpu_cores=$2, ram_gb=$3, s3_tb=$4, db_storage_gb=$5, volume_gb=$6,
		 lb_count=$7, price_cents=$8, currency=$9, billing_cycle=$10, active=$11, updated_at=$12
		 WHERE id=$13`,
		plan.Name, plan.CPUCores, plan.RAMGB, plan.S3TB, plan.DBStorageGB, plan.VolumeGB,
		plan.LBCount, plan.PriceCents, plan.Currency, plan.BillingCycle, plan.Active, plan.UpdatedAt, id,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("plan name already exists")
		}
		return nil, fmt.Errorf("update plan: %w", err)
	}
	return plan, nil
}

// ---------- Customers ----------

func (r *PostgresCustomerRepository) CreateCustomer(ctx context.Context, input *models.CreateCustomerInput) (*models.Customer, error) {
	now := time.Now()
	id := uuid.New().String()
	clusterName := domainToClusterName(input.Domain)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO customers (id, name, domain, plan_id, contact_email, contact_name, status, cluster_status, capi_cluster_name, cluster_region, cluster_nodes, cluster_k8s_version, notes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 'active', 'pending', $7, 'fsn1', 3, 'v1.31.2', '', $8, $9)`,
		id, input.Name, input.Domain, input.PlanID, input.ContactEmail, input.ContactName, clusterName, now, now,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return nil, fmt.Errorf("domain already in use")
			}
			if pgErr.Code == "23503" {
				return nil, fmt.Errorf("plan not found")
			}
		}
		return nil, fmt.Errorf("insert customer: %w", err)
	}

	return &models.Customer{
		ID: id, Name: input.Name, Domain: input.Domain, PlanID: input.PlanID,
		ContactEmail: input.ContactEmail, ContactName: input.ContactName,
		Status: "active", ClusterStatus: "pending", CAPIClusterName: clusterName,
		ClusterRegion: "fsn1", ClusterNodes: 3, ClusterK8sVersion: "v1.31.2",
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (r *PostgresCustomerRepository) GetCustomer(ctx context.Context, id string) (*models.Customer, error) {
	var c models.Customer
	var p models.Plan
	err := r.pool.QueryRow(ctx,
		`SELECT c.id, c.name, c.domain, c.plan_id, c.contact_email, c.contact_name, c.status, c.cluster_status,
		        c.capi_cluster_name, c.cluster_region, c.cluster_nodes, c.cluster_k8s_version,
		        c.notes, c.created_at, c.updated_at,
		        p.id, p.name, p.cpu_cores, p.ram_gb, p.s3_tb, p.db_storage_gb, p.volume_gb, p.lb_count, p.price_cents, p.currency, p.billing_cycle, p.active, p.created_at, p.updated_at
		 FROM customers c JOIN plans p ON c.plan_id = p.id WHERE c.id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Domain, &c.PlanID, &c.ContactEmail, &c.ContactName, &c.Status, &c.ClusterStatus,
		&c.CAPIClusterName, &c.ClusterRegion, &c.ClusterNodes, &c.ClusterK8sVersion,
		&c.Notes, &c.CreatedAt, &c.UpdatedAt,
		&p.ID, &p.Name, &p.CPUCores, &p.RAMGB, &p.S3TB, &p.DBStorageGB, &p.VolumeGB, &p.LBCount, &p.PriceCents, &p.Currency, &p.BillingCycle, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("get customer: %w", err)
	}
	c.Plan = &p
	return &c, nil
}

func (r *PostgresCustomerRepository) ListCustomers(ctx context.Context) ([]models.Customer, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.name, c.domain, c.plan_id, c.contact_email, c.contact_name, c.status, c.cluster_status,
		        c.capi_cluster_name, c.cluster_region, c.cluster_nodes, c.cluster_k8s_version,
		        c.notes, c.created_at, c.updated_at,
		        p.id, p.name, p.cpu_cores, p.ram_gb, p.s3_tb, p.db_storage_gb, p.volume_gb, p.lb_count, p.price_cents, p.currency, p.billing_cycle, p.active, p.created_at, p.updated_at
		 FROM customers c JOIN plans p ON c.plan_id = p.id ORDER BY c.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	var customers []models.Customer
	for rows.Next() {
		var c models.Customer
		var p models.Plan
		if err := rows.Scan(&c.ID, &c.Name, &c.Domain, &c.PlanID, &c.ContactEmail, &c.ContactName, &c.Status, &c.ClusterStatus,
			&c.CAPIClusterName, &c.ClusterRegion, &c.ClusterNodes, &c.ClusterK8sVersion,
			&c.Notes, &c.CreatedAt, &c.UpdatedAt,
			&p.ID, &p.Name, &p.CPUCores, &p.RAMGB, &p.S3TB, &p.DBStorageGB, &p.VolumeGB, &p.LBCount, &p.PriceCents, &p.Currency, &p.BillingCycle, &p.Active, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		c.Plan = &p
		customers = append(customers, c)
	}
	if customers == nil {
		customers = []models.Customer{}
	}
	return customers, rows.Err()
}

func (r *PostgresCustomerRepository) UpdateCustomer(ctx context.Context, id string, input *models.UpdateCustomerInput) (*models.Customer, error) {
	customer, err := r.GetCustomer(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		customer.Name = *input.Name
	}
	if input.Domain != nil {
		customer.Domain = *input.Domain
	}
	if input.PlanID != nil {
		customer.PlanID = *input.PlanID
	}
	if input.ContactEmail != nil {
		customer.ContactEmail = *input.ContactEmail
	}
	if input.ContactName != nil {
		customer.ContactName = *input.ContactName
	}
	if input.Notes != nil {
		customer.Notes = *input.Notes
	}
	customer.UpdatedAt = time.Now()

	_, err = r.pool.Exec(ctx,
		`UPDATE customers SET name=$1, domain=$2, plan_id=$3, contact_email=$4, contact_name=$5, notes=$6, updated_at=$7
		 WHERE id=$8`,
		customer.Name, customer.Domain, customer.PlanID, customer.ContactEmail, customer.ContactName, customer.Notes, customer.UpdatedAt, id,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("domain already in use")
		}
		return nil, fmt.Errorf("update customer: %w", err)
	}

	// Re-fetch to get updated plan join
	return r.GetCustomer(ctx, id)
}

func (r *PostgresCustomerRepository) DeleteCustomer(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}

func (r *PostgresCustomerRepository) SuspendCustomer(ctx context.Context, id string) (*models.Customer, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE customers SET status = 'suspended', updated_at = $1 WHERE id = $2`, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("suspend customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("customer not found")
	}
	return r.GetCustomer(ctx, id)
}

func (r *PostgresCustomerRepository) ActivateCustomer(ctx context.Context, id string) (*models.Customer, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE customers SET status = 'active', updated_at = $1 WHERE id = $2`, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("activate customer: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("customer not found")
	}
	return r.GetCustomer(ctx, id)
}

func (r *PostgresCustomerRepository) UpdateClusterStatus(ctx context.Context, id, status string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE customers SET cluster_status = $1, updated_at = $2 WHERE id = $3`, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update cluster status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}

func (r *PostgresCustomerRepository) SetCAPIClusterName(ctx context.Context, id, clusterName string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE customers SET capi_cluster_name = $1, updated_at = $2 WHERE id = $3`, clusterName, time.Now(), id)
	if err != nil {
		return fmt.Errorf("set capi cluster name: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}

func (r *PostgresCustomerRepository) UpdateClusterInfo(ctx context.Context, id string, nodes int, k8sVersion string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE customers SET cluster_nodes = $1, cluster_k8s_version = $2, updated_at = $3 WHERE id = $4`,
		nodes, k8sVersion, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update cluster info: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}

func (r *PostgresCustomerRepository) GetCustomerByClusterName(ctx context.Context, clusterName string) (*models.Customer, error) {
	var c models.Customer
	var p models.Plan
	err := r.pool.QueryRow(ctx,
		`SELECT c.id, c.name, c.domain, c.plan_id, c.contact_email, c.contact_name, c.status, c.cluster_status,
		        c.capi_cluster_name, c.cluster_region, c.cluster_nodes, c.cluster_k8s_version,
		        c.notes, c.created_at, c.updated_at,
		        p.id, p.name, p.cpu_cores, p.ram_gb, p.s3_tb, p.db_storage_gb, p.volume_gb, p.lb_count, p.price_cents, p.currency, p.billing_cycle, p.active, p.created_at, p.updated_at
		 FROM customers c JOIN plans p ON c.plan_id = p.id WHERE c.capi_cluster_name = $1`, clusterName,
	).Scan(&c.ID, &c.Name, &c.Domain, &c.PlanID, &c.ContactEmail, &c.ContactName, &c.Status, &c.ClusterStatus,
		&c.CAPIClusterName, &c.ClusterRegion, &c.ClusterNodes, &c.ClusterK8sVersion,
		&c.Notes, &c.CreatedAt, &c.UpdatedAt,
		&p.ID, &p.Name, &p.CPUCores, &p.RAMGB, &p.S3TB, &p.DBStorageGB, &p.VolumeGB, &p.LBCount, &p.PriceCents, &p.Currency, &p.BillingCycle, &p.Active, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("get customer by cluster name: %w", err)
	}
	c.Plan = &p
	return &c, nil
}

func (r *PostgresCustomerRepository) ListProvisioningCustomers(ctx context.Context) ([]models.Customer, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT c.id, c.name, c.domain, c.plan_id, c.contact_email, c.contact_name, c.status, c.cluster_status,
		        c.capi_cluster_name, c.cluster_region, c.cluster_nodes, c.cluster_k8s_version,
		        c.notes, c.created_at, c.updated_at
		 FROM customers c WHERE c.cluster_status IN ('pending', 'provisioning', 'installing')`)
	if err != nil {
		return nil, fmt.Errorf("list provisioning customers: %w", err)
	}
	defer rows.Close()

	var customers []models.Customer
	for rows.Next() {
		var c models.Customer
		if err := rows.Scan(&c.ID, &c.Name, &c.Domain, &c.PlanID, &c.ContactEmail, &c.ContactName, &c.Status, &c.ClusterStatus,
			&c.CAPIClusterName, &c.ClusterRegion, &c.ClusterNodes, &c.ClusterK8sVersion,
			&c.Notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan provisioning customer: %w", err)
		}
		customers = append(customers, c)
	}
	if customers == nil {
		customers = []models.Customer{}
	}
	return customers, rows.Err()
}

func (r *PostgresCustomerRepository) GetCustomerStats(ctx context.Context) (*models.CustomerStats, error) {
	var total, active, newThisMonth int
	var mrrCents int

	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers`).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("count customers: %w", err)
	}

	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM customers WHERE status = 'active'`).Scan(&active)
	if err != nil {
		return nil, fmt.Errorf("count active customers: %w", err)
	}

	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(p.price_cents), 0)
		 FROM customers c JOIN plans p ON c.plan_id = p.id
		 WHERE c.status = 'active'`).Scan(&mrrCents)
	if err != nil {
		return nil, fmt.Errorf("calculate mrr: %w", err)
	}

	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM customers WHERE created_at >= date_trunc('month', now())`).Scan(&newThisMonth)
	if err != nil {
		return nil, fmt.Errorf("count new customers: %w", err)
	}

	return &models.CustomerStats{
		TotalCustomers:  total,
		ActiveCustomers: active,
		MRR:             fmt.Sprintf("€%d", mrrCents/100),
		NewThisMonth:    newThisMonth,
	}, nil
}
