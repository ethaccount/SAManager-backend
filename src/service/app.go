package service

import (
	"context"
	"log"
	"time"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/repository"
	"github.com/go-webauthn/webauthn/webauthn"
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

	database.AutoMigrate(&domain.User{}, &domain.Credential{}, &domain.Challenge{})

	passkeyRepo := repository.NewPasskeyRepository(database)
	passkeyService, err := NewPasskeyService(ctx, passkeyRepo, config.WebAuthnConfig, 5*time.Minute)
	if err != nil {
		log.Fatalf("failed to create passkey service: %v", err)
	}

	// pollingService.start()

	return &Application{
		PasskeyService: passkeyService,
		// ExecutionService: executionService,
	}
}
