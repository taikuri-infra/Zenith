package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ ports.TeamMemberRepository = (*PostgresTeamMemberRepository)(nil)

// PostgresTeamMemberRepository persists team members in PostgreSQL.
type PostgresTeamMemberRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTeamMemberRepository creates a new PostgresTeamMemberRepository.
func NewPostgresTeamMemberRepository(pool *pgxpool.Pool) *PostgresTeamMemberRepository {
	return &PostgresTeamMemberRepository{pool: pool}
}

func (r *PostgresTeamMemberRepository) CreateMember(ctx context.Context, member *entities.TeamMember) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO team_members (id, account_id, user_id, email, role, status, invite_token_hash, invite_expires_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		member.ID, member.AccountID, nilIfEmpty(member.UserID), member.Email, string(member.Role),
		string(member.Status), nilIfEmpty(member.InviteTokenHash), member.InviteExpiresAt,
		member.CreatedAt, member.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("member with email %s already exists for this account", member.Email)
		}
		return fmt.Errorf("insert team member: %w", err)
	}
	return nil
}

func (r *PostgresTeamMemberRepository) GetMember(ctx context.Context, id string) (*entities.TeamMember, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, account_id, user_id, email, role, status, invite_token_hash, invite_expires_at, created_at, updated_at
		 FROM team_members WHERE id = $1`, id)
	return scanTeamMember(row)
}

func (r *PostgresTeamMemberRepository) GetMemberByEmail(ctx context.Context, accountID, email string) (*entities.TeamMember, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, account_id, user_id, email, role, status, invite_token_hash, invite_expires_at, created_at, updated_at
		 FROM team_members WHERE account_id = $1 AND email = $2`, accountID, email)
	return scanTeamMember(row)
}

func (r *PostgresTeamMemberRepository) GetMemberByUserID(ctx context.Context, userID string) (*entities.TeamMember, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, account_id, user_id, email, role, status, invite_token_hash, invite_expires_at, created_at, updated_at
		 FROM team_members WHERE user_id = $1 AND status = 'active'`, userID)
	return scanTeamMember(row)
}

func (r *PostgresTeamMemberRepository) GetMemberByInviteHash(ctx context.Context, hash string) (*entities.TeamMember, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, account_id, user_id, email, role, status, invite_token_hash, invite_expires_at, created_at, updated_at
		 FROM team_members WHERE invite_token_hash = $1`, hash)
	return scanTeamMember(row)
}

func (r *PostgresTeamMemberRepository) ListMembers(ctx context.Context, accountID string) ([]entities.TeamMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, account_id, user_id, email, role, status, invite_token_hash, invite_expires_at, created_at, updated_at
		 FROM team_members WHERE account_id = $1 ORDER BY created_at`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list team members: %w", err)
	}
	defer rows.Close()

	var members []entities.TeamMember
	for rows.Next() {
		m, err := scanTeamMemberRow(rows)
		if err != nil {
			return nil, err
		}
		members = append(members, *m)
	}
	return members, nil
}

func (r *PostgresTeamMemberRepository) UpdateMember(ctx context.Context, member *entities.TeamMember) error {
	member.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE team_members SET user_id = $1, email = $2, role = $3, status = $4,
		 invite_token_hash = $5, invite_expires_at = $6, updated_at = $7
		 WHERE id = $8`,
		nilIfEmpty(member.UserID), member.Email, string(member.Role), string(member.Status),
		nilIfEmpty(member.InviteTokenHash), member.InviteExpiresAt, member.UpdatedAt, member.ID,
	)
	if err != nil {
		return fmt.Errorf("update team member: %w", err)
	}
	return nil
}

func (r *PostgresTeamMemberRepository) DeleteMember(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM team_members WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete team member: %w", err)
	}
	return nil
}

func (r *PostgresTeamMemberRepository) CountMembers(ctx context.Context, accountID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM team_members WHERE account_id = $1`, accountID).Scan(&count)
	return count, err
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func scanTeamMember(row pgx.Row) (*entities.TeamMember, error) {
	var m entities.TeamMember
	var userID, role, status, inviteHash *string
	err := row.Scan(&m.ID, &m.AccountID, &userID, &m.Email, &role, &status,
		&inviteHash, &m.InviteExpiresAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("team member not found")
		}
		return nil, fmt.Errorf("scan team member: %w", err)
	}
	if userID != nil {
		m.UserID = *userID
	}
	if role != nil {
		m.Role = entities.Role(*role)
	}
	if status != nil {
		m.Status = entities.TeamMemberStatus(*status)
	}
	if inviteHash != nil {
		m.InviteTokenHash = *inviteHash
	}
	return &m, nil
}

func scanTeamMemberRow(rows pgx.Rows) (*entities.TeamMember, error) {
	var m entities.TeamMember
	var userID, role, status, inviteHash *string
	err := rows.Scan(&m.ID, &m.AccountID, &userID, &m.Email, &role, &status,
		&inviteHash, &m.InviteExpiresAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan team member: %w", err)
	}
	if userID != nil {
		m.UserID = *userID
	}
	if role != nil {
		m.Role = entities.Role(*role)
	}
	if status != nil {
		m.Status = entities.TeamMemberStatus(*status)
	}
	if inviteHash != nil {
		m.InviteTokenHash = *inviteHash
	}
	return &m, nil
}
