package service

import (
	"context"
	"log"
	"time"

	"github.com/ethaccount/backend/src/repository"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	postgresDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Application struct {
	PasskeyService *PasskeyService
}

type ApplicationConfig struct {
	DatabaseDSN    string
	WebAuthnConfig *webauthn.Config
}

func NewApplication(ctx context.Context, config ApplicationConfig) *Application {
	database, err := gorm.Open(postgresDriver.Open(config.DatabaseDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// run migration files
	migrationPath := "file://migrations"

	MigrationUp(config.DatabaseDSN, migrationPath)

	passkeyRepo := repository.NewPasskeyRepository(database)
	passkeyService, err := NewPasskeyService(ctx, passkeyRepo, config.WebAuthnConfig, 5*time.Minute)
	if err != nil {
		log.Fatalf("failed to create passkey service: %v", err)
	}

	// pollingService.start()+0

	return &Application{
		PasskeyService: passkeyService,
		// ExecutionService: executionService,
	}
}

func MigrationUp(databaseDSN string, migrationPath string) {
	migration, err := migrate.New(
		migrationPath,
		databaseDSN)
	if err != nil {
		log.Fatalf("failed to create migrate: %v", err)
	}

	if err := migration.Up(); err != nil {
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

	if err := migration.Down(); err != nil {
		log.Fatalf("failed to run migration down: %v", err)
	}
}
