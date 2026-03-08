package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDatabaseRepository is a PostgreSQL-backed DatabaseRepository.
type PostgresDatabaseRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresDatabaseRepository creates a new PostgreSQL DatabaseRepository.
func NewPostgresDatabaseRepository(pool *pgxpool.Pool) *PostgresDatabaseRepository {
	return &PostgresDatabaseRepository{pool: pool}
}

const dbSelectCols = `id, app_id, user_id, project_id, name, engine, db_name, db_user, host, port, size_mb, max_size_mb, status, provisioner, created_at, updated_at`

func scanDatabase(scanner interface{ Scan(dest ...any) error }) (*entities.UserDatabase, error) {
	var d entities.UserDatabase
	err := scanner.Scan(
		&d.ID, &d.AppID, &d.UserID, &d.ProjectID, &d.Name, &d.Engine,
		&d.DBName, &d.DBUser, &d.Host, &d.Port,
		&d.SizeMB, &d.MaxSizeMB, &d.Status, &d.Provisioner,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *PostgresDatabaseRepository) CreateDatabase(ctx context.Context, appID, userID string, input *dto.CreateDatabaseInput) (*entities.UserDatabase, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	engine := input.Engine
	if engine == "" {
		engine = entities.DatabaseEnginePostgres
	}

	name := input.Name
	if name == "" {
		name = string(engine)
	}

	id := uuid.New().String()
	now := time.Now()
	dbName := "db_" + strings.ReplaceAll(id[:8], "-", "")
	dbUser := "u_" + strings.ReplaceAll(id[:8], "-", "")

	maxSizeMB := input.MaxSizeMB
	if maxSizeMB <= 0 {
		maxSizeMB = 500 // default
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_databases (id, app_id, user_id, project_id, name, engine, db_name, db_user, host, port, size_mb, max_size_mb, status, provisioner, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		id, appID, userID, "", name, string(engine), dbName, dbUser, "", 5432, 0, maxSizeMB,
		string(entities.DatabaseStatusProvisioning), string(entities.DBProvisionerShared), now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_user_databases_app_engine") {
			return nil, fmt.Errorf("database with engine %s already exists for this app", engine)
		}
		if strings.Contains(err.Error(), "idx_user_databases_user_name") {
			return nil, fmt.Errorf("database '%s' already exists", name)
		}
		return nil, fmt.Errorf("create database: %w", err)
	}

	return &entities.UserDatabase{
		ID:          id,
		AppID:       appID,
		UserID:      userID,
		Name:        name,
		Engine:      engine,
		DBName:      dbName,
		DBUser:      dbUser,
		Host:        "",
		Port:        5432,
		SizeMB:      0,
		MaxSizeMB:   maxSizeMB,
		Status:      entities.DatabaseStatusProvisioning,
		Provisioner: entities.DBProvisionerShared,
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

func (r *PostgresDatabaseRepository) GetDatabase(ctx context.Context, id string) (*entities.UserDatabase, error) {
	d, err := scanDatabase(r.pool.QueryRow(ctx,
		`SELECT `+dbSelectCols+` FROM user_databases WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("database not found: %s", id)
	}
	return d, nil
}

func (r *PostgresDatabaseRepository) ListDatabasesByApp(ctx context.Context, appID string) ([]entities.UserDatabase, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+dbSelectCols+` FROM user_databases WHERE app_id = $1 ORDER BY created_at DESC`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list databases by app: %w", err)
	}
	defer rows.Close()

	var dbs []entities.UserDatabase
	for rows.Next() {
		d, err := scanDatabase(rows)
		if err != nil {
			return nil, fmt.Errorf("scan database: %w", err)
		}
		dbs = append(dbs, *d)
	}
	return dbs, nil
}

func (r *PostgresDatabaseRepository) ListDatabasesByUser(ctx context.Context, userID string) ([]entities.UserDatabase, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+dbSelectCols+` FROM user_databases WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list databases by user: %w", err)
	}
	defer rows.Close()

	var dbs []entities.UserDatabase
	for rows.Next() {
		d, err := scanDatabase(rows)
		if err != nil {
			return nil, fmt.Errorf("scan database: %w", err)
		}
		dbs = append(dbs, *d)
	}
	return dbs, nil
}

func (r *PostgresDatabaseRepository) ListDatabasesByProject(ctx context.Context, projectID string) ([]entities.UserDatabase, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+dbSelectCols+` FROM user_databases WHERE project_id = $1 ORDER BY created_at DESC`, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list databases by project: %w", err)
	}
	defer rows.Close()

	var dbs []entities.UserDatabase
	for rows.Next() {
		d, err := scanDatabase(rows)
		if err != nil {
			return nil, fmt.Errorf("scan database: %w", err)
		}
		dbs = append(dbs, *d)
	}
	return dbs, nil
}

func (r *PostgresDatabaseRepository) DeleteDatabase(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM user_databases WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete database: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("database not found: %s", id)
	}
	return nil
}

func (r *PostgresDatabaseRepository) UpdateDatabaseStatus(ctx context.Context, id string, status entities.DatabaseStatus) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE user_databases SET status = $1, updated_at = $2 WHERE id = $3`,
		string(status), now, id,
	)
	if err != nil {
		return fmt.Errorf("update database status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("database not found: %s", id)
	}
	return nil
}

func (r *PostgresDatabaseRepository) CountDatabasesByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_databases WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count databases: %w", err)
	}
	return count, nil
}

func (r *PostgresDatabaseRepository) CountDatabases(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_databases`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count databases: %w", err)
	}
	return count, nil
}

// UpdateDatabaseHost sets the host and status after provisioning.
func (r *PostgresDatabaseRepository) UpdateDatabaseHost(ctx context.Context, id, host string, provisioner entities.DatabaseProvisioner) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE user_databases SET host = $1, provisioner = $2, status = $3, updated_at = $4 WHERE id = $5`,
		host, string(provisioner), string(entities.DatabaseStatusReady), now, id,
	)
	if err != nil {
		return fmt.Errorf("update database host: %w", err)
	}
	return nil
}
