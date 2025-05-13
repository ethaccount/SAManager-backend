package main

import (
	"context"
	"log"
	"os"

	"github.com/ethaccount/backend/src/handler"
	"github.com/ethaccount/backend/src/service"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rs/zerolog"

	"github.com/joho/godotenv"
)

const (
	AppName    = "SAManager Backend"
	AppVersion = "0.0.1"
	AppBuild   = "dev"
)

type AppConfig struct {
	LogLevel *string

	// Database configuration
	DSN *string

	// HTTP configuration
	Port *string
}

func initAppConfig() AppConfig {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatalf("DB_URL not set in .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// "error", "warn", "info", "debug", "disabled"
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "debug"
	}

	return AppConfig{
		LogLevel: &logLevel,
		DSN:      &dsn,
		Port:     &port,
	}
}

func initRootLogger(levelStr string) zerolog.Logger {
	// Set global log level
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Add color and formatting
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    false,
		TimeFormat: "2006-01-02 15:04:05",
	}

	rootLogger := zerolog.New(output).With().
		Timestamp().
		Str("service", AppName).
		Logger()

	return rootLogger
}

func main() {
	// Setup app configuration
	cfg := initAppConfig()

	// Create root logger
	rootLogger := initRootLogger(*cfg.LogLevel)

	// Create root context
	rootCtx := context.Background()
	rootCtx = rootLogger.WithContext(rootCtx)

	rootLogger.Info().
		Str("version", AppVersion).
		Str("build", AppBuild).
		Msgf("Launching %s", AppName)

	// Create application
	app := service.NewApplication(rootCtx, service.ApplicationConfig{
		DatabaseDSN: *cfg.DSN,
		WebAuthnConfig: &webauthn.Config{
			RPDisplayName: "Passkey Demo",
			RPID:          "localhost",
			RPOrigins:     []string{"http://localhost:" + *cfg.Port},
		},
	})

	ginRouter := gin.Default()
	handler.RegisterRoutes(rootCtx, ginRouter, app)
	ginRouter.Run(":" + *cfg.Port)
}
