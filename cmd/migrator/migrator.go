package migrator

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/foreground-eclipse/wallet/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func Migrate(logger *zap.Logger, cfg *config.Config) error {
	const op = "cmd.migrator.Migrate"

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.SSLMode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err := db.Ping(); err != nil {
		fmt.Println(db)
		return fmt.Errorf("%s: %w", op, err)
	}

	defer db.Close()

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working dir path: %w", err)
	}

	dir = fmt.Sprintf("%s/internal/migrations", dir)
	dir = filepath.ToSlash(dir)

	logger.Info("trying to apply migrations")

	m, err := migrate.New(fmt.Sprintf("file://%s", dir), fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("no migrations to apply")
			return fmt.Errorf("%s: %w", op, err)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
	defer m.Close()

	logger.Info("migrations applied")

	return nil
}
