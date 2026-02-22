package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
)

func main() {
	dbConfig := config.LoadDBConfig()
	if dbConfig.Name == "" {
		dbConfig.Name = "deck" // default for local development
	}

	db, err := db.InitDB(dbConfig)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		slog.Error("failed to create driver", "error", err)
		os.Exit(1)
	}

	// Use MIGRATIONS_PATH env var if set, otherwise use local path
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "file://cmd/migrate/migrations"
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, dbConfig.Name, driver)
	if err != nil {
		slog.Error("failed to create migrate instance", "error", err)
		os.Exit(1)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations completed successfully", "database", dbConfig.Name)
}
