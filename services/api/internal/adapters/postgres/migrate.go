package postgres

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
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
			v, dirty, _ := m.Version()
			slog.Info("migrations up to date", "version", v, "dirty", dirty)
		} else {
			return fmt.Errorf("run migrations: %w", err)
		}
	} else {
		v, _, _ := m.Version()
		slog.Info("migrations completed", "version", v)
	}

	return nil
}
