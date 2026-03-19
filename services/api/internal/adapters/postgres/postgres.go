package postgres

import (
	"context"
	"fmt"
	"os"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool creates a pgxpool connection pool from a DSN.
// When OTel is enabled (OTEL_EXPORTER_OTLP_ENDPOINT is set), all queries
// are automatically traced with span attributes: db.statement, db.system, etc.
func NewPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	// Add OTel tracing to every database query when tracing is enabled
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		cfg.ConnConfig.Tracer = otelpgx.NewTracer()
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
