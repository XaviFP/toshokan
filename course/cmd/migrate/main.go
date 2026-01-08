package main

import (
	"errors"
	"log"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	db, err := db.InitDB(config.DBConfig{User: "toshokan", Password: "t.o.s.h.o.k.a.n.", Name: "course", Host: "localhost", Port: "5432"})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf(" %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://cmd/migrate/migrations", "course", driver)
	if err != nil {
		log.Fatalf(" %v", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf(" %v", err)
	}
}
