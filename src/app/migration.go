package app

import (
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func MigrationUp(databaseDSN string, migrationPath string) {
	migration, err := migrate.New(
		migrationPath,
		databaseDSN)
	if err != nil {
		log.Fatalf("failed to create migrate: %v", err)
	}

	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run migration up: %v", err)
	}
}

func MigrationDown(databaseDSN string, migrationPath string) {
	migration, err := migrate.New(
		migrationPath,
		databaseDSN)
	if err != nil {
		log.Fatalf("failed to create migrate: %v", err)
	}

	if err := migration.Down(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to run migration down: %v", err)
	}
}
