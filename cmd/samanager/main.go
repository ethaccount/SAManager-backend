package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ethaccount/backend/src/app"
	"github.com/joho/godotenv"

	"github.com/ethaccount/backend/docs/swagger"
)

// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  AGPL-3.0-only

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.basic  BasicAuth

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/

const (
	AppName    = "SAManager Backend"
	AppVersion = "0.1.2"
)

func main() {
	// Load .env file if it exists (optional in production)
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Overload(".env")
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}

	config := app.NewAppConfig()

	// Update swagger info dynamically using constants
	swagger.SwaggerInfo.Title = AppName + " API"
	swagger.SwaggerInfo.Version = AppVersion
	swagger.SwaggerInfo.Description = fmt.Sprintf("%s with automation service and webauthn authentication", AppName)
	if config.Host != nil {
		swagger.SwaggerInfo.Host = *config.Host
	}

	// Create root logger
	logger := app.InitLogger(*config.LogLevel)

	// Create root context
	rootCtx, rootCancel := context.WithCancel(context.Background())
	rootCtx = logger.WithContext(rootCtx)

	logger.Info().
		Str("version", AppVersion).
		Str("environment", *config.Environment).
		Msgf("Launching %s", AppName)

	// Build swagger URL based on environment and host config
	var swaggerURL string
	if *config.Environment == "dev" {
		swaggerURL = "http://" + *config.Host + "/swagger/index.html"
	} else {
		// For staging/prod, assume HTTPS
		swaggerURL = "https://" + *config.Host + "/swagger/index.html"
	}

	logger.Info().
		Str("swagger_link", swaggerURL).
		Msg("Swagger link")

	// ================================
	// Start application
	// ================================

	app, err := app.NewApplication(rootCtx, *config)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize application")
		return
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go app.RunHTTPServer(rootCtx, &wg)

	wg.Add(1)
	go app.RunPollingWorker(rootCtx, &wg)

	// ================================

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Cancel root context to signal all workers to stop
	rootCancel()

	// Wait for all workers to complete with timeout
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		logger.Info().Msg("All workers shut down gracefully")
	case <-time.After(15 * time.Second):
		logger.Error().Msg("Timeout waiting for workers to shut down")
	}

	// Shutdown application
	app.Shutdown(rootCtx)

	logger.Info().Msg("Application shutdown complete")
}
