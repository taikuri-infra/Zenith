package postgres

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/dto"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// bucketNameRe validates bucket names: lowercase alphanumeric + hyphens, 3-63 chars.
var bucketNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)

// PostgresStorageRepository is a PostgreSQL-backed StorageRepository.
type PostgresStorageRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresStorageRepository creates a new PostgreSQL StorageRepository.
func NewPostgresStorageRepository(pool *pgxpool.Pool) *PostgresStorageRepository {
	return &PostgresStorageRepository{pool: pool}
}

func (r *PostgresStorageRepository) CreateBucket(ctx context.Context, appID, userID string, input *dto.CreateBucketInput) (*entities.UserBucket, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if input.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if !bucketNameRe.MatchString(input.Name) {
		return nil, fmt.Errorf("invalid bucket name: must be 3-63 lowercase alphanumeric characters or hyphens")
	}

	access := input.Access
	if access == "" {
		access = entities.BucketAccessPrivate
	}

	id := uuid.New().String()
	now := time.Now()
	s3Prefix := fmt.Sprintf("u/%s/%s/", userID, input.Name)
	endpoint := fmt.Sprintf("https://%s.s3.zenith.local", input.Name)

	// Generate real S3 bucket name for per-customer isolation
	uid := userID
	if len(uid) > 12 {
		uid = uid[:12]
	}
	s3BucketName := "zenith-" + strings.ToLower(uid) + "-" + input.Name

	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_buckets (id, app_id, user_id, name, s3_prefix, s3_bucket_name, access, region, size_mb, max_size_mb, objects, status, endpoint, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		id, appID, userID, input.Name, s3Prefix, s3BucketName, string(access), "fsn1", 0, 1024, 0, string(entities.BucketStatusActive), endpoint, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_user_buckets_user_name") {
			return nil, fmt.Errorf("bucket '%s' already exists", input.Name)
		}
		return nil, fmt.Errorf("create bucket: %w", err)
	}

	return &entities.UserBucket{
		ID:           id,
		AppID:        appID,
		UserID:       userID,
		Name:         input.Name,
		S3Prefix:     s3Prefix,
		S3BucketName: s3BucketName,
		Access:       access,
		Region:       "fsn1",
		SizeMB:       0,
		MaxSizeMB:    1024,
		Objects:      0,
		Status:       entities.BucketStatusActive,
		Endpoint:     endpoint,
		Timestamps: entities.Timestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}, nil
}

const bucketSelectCols = `id, app_id, user_id, name, s3_prefix, s3_bucket_name, access, region, size_mb, max_size_mb, objects, status, endpoint, created_at, updated_at`

func scanBucket(scanner interface{ Scan(dest ...any) error }) (*entities.UserBucket, error) {
	var b entities.UserBucket
	err := scanner.Scan(&b.ID, &b.AppID, &b.UserID, &b.Name, &b.S3Prefix, &b.S3BucketName, &b.Access, &b.Region, &b.SizeMB, &b.MaxSizeMB, &b.Objects, &b.Status, &b.Endpoint, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *PostgresStorageRepository) GetBucket(ctx context.Context, id string) (*entities.UserBucket, error) {
	b, err := scanBucket(r.pool.QueryRow(ctx,
		`SELECT `+bucketSelectCols+` FROM user_buckets WHERE id = $1`, id,
	))
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %s", id)
	}
	return b, nil
}

func (r *PostgresStorageRepository) GetBucketByName(ctx context.Context, userID, name string) (*entities.UserBucket, error) {
	b, err := scanBucket(r.pool.QueryRow(ctx,
		`SELECT `+bucketSelectCols+` FROM user_buckets WHERE user_id = $1 AND name = $2`, userID, name,
	))
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %s", name)
	}
	return b, nil
}

func (r *PostgresStorageRepository) ListBucketsByApp(ctx context.Context, appID string) ([]entities.UserBucket, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+bucketSelectCols+` FROM user_buckets WHERE app_id = $1 ORDER BY created_at DESC`, appID,
	)
	if err != nil {
		return nil, fmt.Errorf("list buckets by app: %w", err)
	}
	defer rows.Close()

	var buckets []entities.UserBucket
	for rows.Next() {
		b, err := scanBucket(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bucket: %w", err)
		}
		buckets = append(buckets, *b)
	}
	return buckets, nil
}

func (r *PostgresStorageRepository) ListBucketsByUser(ctx context.Context, userID string) ([]entities.UserBucket, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+bucketSelectCols+` FROM user_buckets WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list buckets by user: %w", err)
	}
	defer rows.Close()

	var buckets []entities.UserBucket
	for rows.Next() {
		b, err := scanBucket(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bucket: %w", err)
		}
		buckets = append(buckets, *b)
	}
	return buckets, nil
}

func (r *PostgresStorageRepository) UpdateBucket(ctx context.Context, id string, access entities.BucketAccess) (*entities.UserBucket, error) {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE user_buckets SET access = $1, updated_at = $2 WHERE id = $3`,
		string(access), now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update bucket: %w", err)
	}
	return r.GetBucket(ctx, id)
}

func (r *PostgresStorageRepository) DeleteBucket(ctx context.Context, id string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM user_buckets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete bucket: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("bucket not found: %s", id)
	}
	return nil
}

func (r *PostgresStorageRepository) CountBucketsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM user_buckets WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count buckets: %w", err)
	}
	return count, nil
}
