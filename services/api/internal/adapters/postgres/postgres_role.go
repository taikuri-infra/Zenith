package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRoleRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRoleRepository(pool *pgxpool.Pool) *PostgresRoleRepository {
	return &PostgresRoleRepository{pool: pool}
}

func (r *PostgresRoleRepository) CreateRole(ctx context.Context, userID, name, description string, permissions []entities.Permission) (*entities.CustomRole, error) {
	id := uuid.New().String()
	now := time.Now()

	permStrings := make([]string, len(permissions))
	for i, p := range permissions {
		permStrings[i] = string(p)
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO custom_roles (id, user_id, name, description, permissions, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, userID, name, description, permStrings, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}
	return &entities.CustomRole{
		ID:          id,
		UserID:      userID,
		Name:        name,
		Description: description,
		Permissions: permissions,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (r *PostgresRoleRepository) GetRole(ctx context.Context, id string) (*entities.CustomRole, error) {
	var role entities.CustomRole
	var permStrings []string
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, description, permissions, created_at, updated_at
		 FROM custom_roles WHERE id = $1`, id,
	).Scan(&role.ID, &role.UserID, &role.Name, &role.Description, &permStrings, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("role not found: %s", id)
	}
	role.Permissions = make([]entities.Permission, len(permStrings))
	for i, s := range permStrings {
		role.Permissions[i] = entities.Permission(s)
	}
	return &role, nil
}

func (r *PostgresRoleRepository) ListRolesByUser(ctx context.Context, userID string) ([]entities.CustomRole, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, description, permissions, created_at, updated_at
		 FROM custom_roles WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []entities.CustomRole
	for rows.Next() {
		var role entities.CustomRole
		var permStrings []string
		if err := rows.Scan(&role.ID, &role.UserID, &role.Name, &role.Description,
			&permStrings, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		role.Permissions = make([]entities.Permission, len(permStrings))
		for i, s := range permStrings {
			role.Permissions[i] = entities.Permission(s)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *PostgresRoleRepository) UpdateRole(ctx context.Context, id string, name, description *string, permissions []entities.Permission) (*entities.CustomRole, error) {
	now := time.Now()
	sets := []string{"updated_at = $1"}
	args := []interface{}{now}
	argIdx := 2

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
	if permissions != nil {
		permStrings := make([]string, len(permissions))
		for i, p := range permissions {
			permStrings[i] = string(p)
		}
		sets = append(sets, fmt.Sprintf("permissions = $%d", argIdx))
		args = append(args, permStrings)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE custom_roles SET %s WHERE id = $%d",
		joinStrings(sets, ", "), argIdx)

	ct, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil, fmt.Errorf("role not found: %s", id)
	}
	return r.GetRole(ctx, id)
}

func (r *PostgresRoleRepository) DeleteRole(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM custom_roles WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("role not found: %s", id)
	}
	return nil
}

func (r *PostgresRoleRepository) AssignRole(ctx context.Context, roleID, memberID, assignedBy string) (*entities.RoleAssignment, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`INSERT INTO role_assignments (id, role_id, member_id, assigned_by, created_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (role_id, member_id) DO NOTHING`,
		id, roleID, memberID, assignedBy, now,
	)
	if err != nil {
		return nil, fmt.Errorf("assign role: %w", err)
	}
	return &entities.RoleAssignment{
		ID:         id,
		RoleID:     roleID,
		MemberID:   memberID,
		AssignedBy: assignedBy,
		CreatedAt:  now,
	}, nil
}

func (r *PostgresRoleRepository) ListAssignmentsByRole(ctx context.Context, roleID string) ([]entities.RoleAssignment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, role_id, member_id, assigned_by, created_at
		 FROM role_assignments WHERE role_id = $1 ORDER BY created_at DESC`, roleID,
	)
	if err != nil {
		return nil, fmt.Errorf("list assignments: %w", err)
	}
	defer rows.Close()

	var assignments []entities.RoleAssignment
	for rows.Next() {
		var a entities.RoleAssignment
		if err := rows.Scan(&a.ID, &a.RoleID, &a.MemberID, &a.AssignedBy, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	return assignments, nil
}

func (r *PostgresRoleRepository) RemoveAssignment(ctx context.Context, assignmentID string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM role_assignments WHERE id = $1`, assignmentID)
	if err != nil {
		return fmt.Errorf("remove assignment: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("assignment not found: %s", assignmentID)
	}
	return nil
}
