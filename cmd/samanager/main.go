package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"github.com/ethaccount/backend/src/app"
	"github.com/joho/godotenv"

	"github.com/ethaccount/backend/docs/swagger"
	"github.com/rs/zerolog"
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
	AppVersion = "0.1.4"
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

	wg.Add(1)
	go runSystemStatsLogger(rootCtx, &wg, logger)

	if *config.Environment == "dev" {
		wg.Add(1)
		go runPprofServer(rootCtx, &wg, logger)
	}
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

// runPprofServer starts a debug server with pprof endpoints
func runPprofServer(ctx context.Context, wg *sync.WaitGroup, logger zerolog.Logger) {
	defer wg.Done()

	// Use the default mux which has pprof endpoints automatically registered
	server := &http.Server{
		Addr:    ":6060",
		Handler: http.DefaultServeMux,
	}

	// Start server in goroutine
	go func() {
		logger.Info().Msg("pprof server is running on http://localhost:6060/debug/pprof/")
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("Failed to start pprof server")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	logger.Info().Msg("Gracefully shutting down pprof server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// Shutdown server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Failed to shutdown pprof server gracefully")
	} else {
		logger.Info().Msg("pprof server shutdown complete")
	}
}

// runSystemStatsLogger logs system statistics periodically
func runSystemStatsLogger(ctx context.Context, wg *sync.WaitGroup, logger zerolog.Logger) {
	defer wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("System stats logger shutting down")
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			logger.Info().
				Uint64("heap_mb", m.HeapInuse/1024/1024).
				Uint64("sys_mb", m.Sys/1024/1024).
				Int("goroutines", runtime.NumGoroutine()).
				Msg("System stats")

			var gcStats debug.GCStats
			debug.ReadGCStats(&gcStats)
			logger.Info().
				Int64("gc_num", gcStats.NumGC).
				Int64("gc_pause_total", int64(gcStats.PauseTotal)).
				Msg("GC stats")
		}
	}
}
