package setup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"hackload/internal/config"
)

func Setup(ctx context.Context, conf *config.Config, args *config.Args) error {
	format := "sqlite3://%s?x-no-tx-wrap=true"
	connString := fmt.Sprintf(
		format,
		conf.SQLite3Path,
	)

	if err := RunMigrations(args.MigrationsDir, connString); err != nil {
		return err
	}

	return nil
}

func RunMigrations(migrationsPath, dbConnectionString string) error {
	if migrationsPath == "" {
		return nil
	}

	// If the path is relative, convert it to absolute
	if !filepath.IsAbs(migrationsPath) && !strings.Contains(migrationsPath, "://") {
		absPath, err := filepath.Abs(migrationsPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		migrationsPath = absPath
	}

	if !strings.Contains(migrationsPath, "://") {
		migrationsPath = "file://" + migrationsPath
	}

	m, err := migrate.New(migrationsPath, dbConnectionString)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			slog.Error("Error closing migration source", "err", sourceErr)
		}
		if dbErr != nil {
			slog.Error("Error closing migration database", "err", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("No migrations to apply")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	slog.Info("Migrations applied successfully")
	return nil
}
