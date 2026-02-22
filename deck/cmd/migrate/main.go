package main

import (
	"errors"
	"log"
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
		log.Fatalf("failed to connect to database: %v", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("failed to create driver: %v", err)
	}

	// Use MIGRATIONS_PATH env var if set, otherwise use local path
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "file://cmd/migrate/migrations"
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsPath, dbConfig.Name, driver)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migration failed: %v", err)
	}

	log.Printf("migrations completed successfully for database: %s", dbConfig.Name)
}
