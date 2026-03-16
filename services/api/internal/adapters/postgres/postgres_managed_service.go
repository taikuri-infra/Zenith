package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresManagedServiceRepository is a PostgreSQL-backed ManagedServiceRepository.
type PostgresManagedServiceRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresManagedServiceRepository creates a new PostgreSQL ManagedServiceRepository.
func NewPostgresManagedServiceRepository(pool *pgxpool.Pool) *PostgresManagedServiceRepository {
	return &PostgresManagedServiceRepository{pool: pool}
}

const managedServiceSelectCols = `id, project_id, user_id, service_type, name, version,
	COALESCE(connection_url, ''), COALESCE(internal_host, ''), COALESCE(port, 0),
	COALESCE(username, ''), COALESCE(password, ''), COALESCE(database_name, ''),
	COALESCE(k8s_namespace, ''), COALESCE(k8s_resource_name, ''),
	status, COALESCE(status_message, ''), COALESCE(storage_gb, 5),
	created_at, updated_at`

func scanManagedService(scanner interface{ Scan(dest ...any) error }) (*entities.ManagedService, error) {
	var ms entities.ManagedService
	var serviceType, status string
	err := scanner.Scan(
		&ms.ID, &ms.ProjectID, &ms.UserID, &serviceType, &ms.Name, &ms.Version,
		&ms.ConnectionURL, &ms.InternalHost, &ms.Port,
		&ms.Username, &ms.Password, &ms.DatabaseName,
		&ms.K8sNamespace, &ms.K8sResourceName,
		&status, &ms.StatusMessage, &ms.StorageGB,
		&ms.CreatedAt, &ms.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	ms.ServiceType = entities.ServiceType(serviceType)
	ms.Status = entities.ManagedServiceStatus(status)
	return &ms, nil
}

func (r *PostgresManagedServiceRepository) CreateManagedService(ctx context.Context, svc *entities.ManagedService) error {
	now := time.Now()
	svc.CreatedAt = now
	svc.UpdatedAt = now

	_, err := r.pool.Exec(ctx,
		`INSERT INTO managed_services (id, project_id, user_id, service_type, name, version,
			connection_url, internal_host, port, username, password, database_name,
			k8s_namespace, k8s_resource_name, status, status_message, storage_gb,
			created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		svc.ID, svc.ProjectID, svc.UserID, string(svc.ServiceType), svc.Name, svc.Version,
		svc.ConnectionURL, svc.InternalHost, svc.Port, svc.Username, svc.Password, svc.DatabaseName,
		svc.K8sNamespace, svc.K8sResourceName, string(svc.Status), svc.StatusMessage, svc.StorageGB,
		now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_managed_services_project_name") {
			return fmt.Errorf("managed service '%s' already exists in this project", svc.Name)
		}
		return fmt.Errorf("create managed service: %w", err)
	}
	return nil
}

func (r *PostgresManagedServiceRepository) GetManagedService(ctx context.Context, id string) (*entities.ManagedService, error) {
	ms, err := scanManagedService(r.pool.QueryRow(ctx,
		`SELECT `+managedServiceSelectCols+` FROM managed_services WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("managed service not found: %s", id)
	}
	return ms, nil
}

func (r *PostgresManagedServiceRepository) ListManagedServicesByProject(ctx context.Context, projectID string) ([]entities.ManagedService, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+managedServiceSelectCols+` FROM managed_services WHERE project_id = $1 ORDER BY created_at ASC`, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list managed services: %w", err)
	}
	defer rows.Close()

	var services []entities.ManagedService
	for rows.Next() {
		ms, err := scanManagedService(rows)
		if err != nil {
			return nil, fmt.Errorf("scan managed service: %w", err)
		}
		services = append(services, *ms)
	}
	return services, nil
}

func (r *PostgresManagedServiceRepository) ListManagedServicesByUser(ctx context.Context, userID string) ([]entities.ManagedService, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+managedServiceSelectCols+` FROM managed_services WHERE user_id = $1 ORDER BY created_at ASC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list managed services: %w", err)
	}
	defer rows.Close()

	var services []entities.ManagedService
	for rows.Next() {
		ms, err := scanManagedService(rows)
		if err != nil {
			return nil, fmt.Errorf("scan managed service: %w", err)
		}
		services = append(services, *ms)
	}
	return services, nil
}

func (r *PostgresManagedServiceRepository) UpdateManagedServiceStatus(ctx context.Context, id string, status entities.ManagedServiceStatus, statusMsg, connURL, host string, port int) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE managed_services SET status = $1, status_message = $2, connection_url = $3, internal_host = $4, port = $5, updated_at = now() WHERE id = $6`,
		string(status), statusMsg, connURL, host, port, id,
	)
	if err != nil {
		return fmt.Errorf("update managed service status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("managed service not found: %s", id)
	}
	return nil
}

func (r *PostgresManagedServiceRepository) DeleteManagedService(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM managed_services WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete managed service: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("managed service not found: %s", id)
	}
	return nil
}

func (r *PostgresManagedServiceRepository) CountManagedServicesByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM managed_services WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count managed services: %w", err)
	}
	return count, nil
}
