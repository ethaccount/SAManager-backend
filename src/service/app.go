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
	JobService     *JobService
}

type AppConfig struct {
	LogLevel *string

	// Database configuration
	DSN *string

	// HTTP configuration
	Port *string

	// RPC URLs
	SepoliaRPCURL         *string
	ArbitrumSepoliaRPCURL *string
	BaseSepoliaRPCURL     *string
	OptimismSepoliaRPCURL *string
	PolygonAmoyRPCURL     *string
}

func NewApplication(ctx context.Context, config AppConfig) *Application {
	database, err := gorm.Open(postgresDriver.Open(*config.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// run migration files
	migrationPath := "file://migrations"

	MigrationUp(*config.DSN, migrationPath)

	// Passkey Service
	passkeyRepo := repository.NewPasskeyRepository(database)

	webAuthnConfig := &webauthn.Config{
		RPDisplayName: "Passkey Demo",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:" + *config.Port},
	}

	passkeyService, err := NewPasskeyService(ctx, passkeyRepo, webAuthnConfig, 5*time.Minute)
	if err != nil {
		log.Fatalf("failed to create passkey service: %v", err)
	}

	// Job Service
	jobRepo := repository.NewJobRepository(database)
	jobService := NewJobService(jobRepo)

	// Blockchain Service
	blockchainService := NewBlockchainService(config)

	// Polling Service
	pollingService := NewPollingService(jobService, blockchainService, PollingConfig{
		PollingInterval: 10 * time.Second,
	})

	go pollingService.Start(ctx)

	return &Application{
		PasskeyService: passkeyService,
		JobService:     jobService,
	}
}

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
