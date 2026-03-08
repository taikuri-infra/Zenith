package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/lib/pq"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies all pending database migrations using the embedded FS.
func RunMigrations(dsn string, migrations fs.FS) error {
	// Ensure sslmode is set (golang-migrate requires explicit SSL config)
	u, err := url.Parse(dsn)
	if err == nil && u.Query().Get("sslmode") == "" {
		q := u.Query()
		q.Set("sslmode", "disable")
		u.RawQuery = q.Encode()
		dsn = u.String()
	}

	// Check if tables exist — if schema_migrations says up-to-date but tables are missing,
	// drop schema_migrations to force re-migration (disaster recovery scenario)
	if err := checkAndFixMigrationState(dsn); err != nil {
		slog.Warn("migration state check failed", "error", err)
	}

	source, err := iofs.New(migrations, ".")
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			v, _, _ := m.Version()
			slog.Info("migrations up to date", "version", v)
		} else {
			return fmt.Errorf("run migrations: %w", err)
		}
	} else {
		v, _, _ := m.Version()
		slog.Info("migrations completed", "version", v)
	}

	return nil
}

// checkAndFixMigrationState detects when schema_migrations exists but app tables don't
// (e.g., after a DB restore that lost data). Drops schema_migrations to force re-migration.
func checkAndFixMigrationState(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if schema_migrations exists
	var smExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='schema_migrations')").Scan(&smExists)
	if err != nil || !smExists {
		return nil // No schema_migrations = fresh DB, nothing to fix
	}

	// schema_migrations exists — check if core app tables also exist
	var usersExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='users')").Scan(&usersExists)
	if err != nil {
		return err
	}

	if !usersExists {
		slog.Warn("detected stale migration state: schema_migrations exists but users table missing, resetting")
		_, err = db.Exec("DROP TABLE IF EXISTS schema_migrations")
		return err
	}

	return nil
}
