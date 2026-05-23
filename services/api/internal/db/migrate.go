package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(dsn, migrationsPath string) error {
	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("init migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		// Dirty state means a previous migration attempt failed mid-run.
		// Force back to the version before the failed one and retry.
		// Safe only when migration files are idempotent (IF NOT EXISTS / IF EXISTS).
		var dirtyErr migrate.ErrDirty
		if errors.As(err, &dirtyErr) {
			if ferr := m.Force(dirtyErr.Version - 1); ferr != nil {
				return fmt.Errorf("force version after dirty state: %w", ferr)
			}
			if rerr := m.Up(); rerr != nil && !errors.Is(rerr, migrate.ErrNoChange) {
				return fmt.Errorf("run migrations after dirty force: %w", rerr)
			}
			return nil
		}
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}
