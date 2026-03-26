package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresProjectRepository is a PostgreSQL-backed ProjectRepository.
type PostgresProjectRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresProjectRepository creates a new PostgreSQL ProjectRepository.
func NewPostgresProjectRepository(pool *pgxpool.Pool) *PostgresProjectRepository {
	return &PostgresProjectRepository{pool: pool}
}

const projectSelectCols = `id, user_id, name, slug, description, COALESCE(status, 'active'), COALESCE(harbor_project_name, ''), COALESCE(harbor_robot_user, ''), COALESCE(harbor_robot_pass, ''), created_at, updated_at`

func scanProject(scanner interface{ Scan(dest ...any) error }) (*entities.Project, error) {
	var p entities.Project
	err := scanner.Scan(
		&p.ID, &p.UserID, &p.Name, &p.Slug, &p.Description,
		&p.Status, &p.HarborProjectName, &p.HarborRobotUser, &p.HarborRobotPass,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PostgresProjectRepository) CreateProject(ctx context.Context, userID, name, slug, description string) (*entities.Project, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := r.pool.Exec(ctx,
		`INSERT INTO projects (id, user_id, name, slug, description, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		id, userID, name, slug, description, entities.ProjectStatusDraft, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_projects_user_slug") {
			return nil, fmt.Errorf("project slug '%s' already exists", slug)
		}
		return nil, fmt.Errorf("create project: %w", err)
	}

	return &entities.Project{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Slug:        slug,
		Description: description,
		Status:      entities.ProjectStatusDraft,
		Timestamps:  entities.Timestamps{CreatedAt: now, UpdatedAt: now},
	}, nil
}

func (r *PostgresProjectRepository) GetProject(ctx context.Context, id string) (*entities.Project, error) {
	p, err := scanProject(r.pool.QueryRow(ctx,
		`SELECT `+projectSelectCols+` FROM projects WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("project not found: %s", id)
	}
	return p, nil
}

func (r *PostgresProjectRepository) ListProjectsByUser(ctx context.Context, userID string) ([]entities.Project, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+projectSelectCols+` FROM projects WHERE user_id = $1 ORDER BY created_at ASC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []entities.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, *p)
	}
	return projects, nil
}

func (r *PostgresProjectRepository) UpdateProject(ctx context.Context, id string, name, description *string) (*entities.Project, error) {
	sets := []string{"updated_at = now()"}
	args := []interface{}{}
	argIdx := 1

	if name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *name)
		argIdx++
	}
	if description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *description)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE projects SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)

	ct, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("project not found: %s", id)
	}

	return r.GetProject(ctx, id)
}

func (r *PostgresProjectRepository) DeleteProject(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	return nil
}

func (r *PostgresProjectRepository) CountProjectsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count projects: %w", err)
	}
	return count, nil
}

func (r *PostgresProjectRepository) GetDefaultProject(ctx context.Context, userID string) (*entities.Project, error) {
	p, err := scanProject(r.pool.QueryRow(ctx,
		`SELECT `+projectSelectCols+` FROM projects WHERE user_id = $1 AND slug = 'default' LIMIT 1`, userID,
	))
	if err != nil {
		// Fallback: return the first project
		p, err = scanProject(r.pool.QueryRow(ctx,
			`SELECT `+projectSelectCols+` FROM projects WHERE user_id = $1 ORDER BY created_at ASC LIMIT 1`, userID,
		))
		if err != nil {
			return nil, fmt.Errorf("no projects found for user: %s", userID)
		}
	}
	return p, nil
}

func (r *PostgresProjectRepository) SetHarborCredentials(ctx context.Context, id, harborProjectName, robotUser, robotPass string) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE projects SET harbor_project_name = $1, harbor_robot_user = $2, harbor_robot_pass = $3, updated_at = now() WHERE id = $4`,
		harborProjectName, robotUser, robotPass, id,
	)
	if err != nil {
		return fmt.Errorf("set harbor credentials: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	return nil
}

func (r *PostgresProjectRepository) UpdateProjectStatus(ctx context.Context, id string, status entities.ProjectStatus) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE projects SET status = $1, updated_at = now() WHERE id = $2`,
		string(status), id,
	)
	if err != nil {
		return fmt.Errorf("update project status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", id)
	}
	return nil
}
