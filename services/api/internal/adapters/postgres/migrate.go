package postgres

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies all pending database migrations using the embedded FS.
func RunMigrations(dsn string, migrations fs.FS) error {
	source, err := iofs.New(migrations, ".")
	if err != nil {
		return fmt.Errorf("open migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
