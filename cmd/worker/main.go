package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// HTTPWorker represents the HTTP server worker
type HTTPWorker struct {
	server *http.Server
	db     *sql.DB
	port   int
}

func NewHTTPWorker(ctx context.Context, port int, dbDSN string) (*HTTPWorker, error) {

	// Connect to Postgres
	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("API endpoint working"))
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return &HTTPWorker{
		server: server,
		db:     db,
		port:   port,
	}, nil
}

func (hw *HTTPWorker) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := zerolog.Ctx(ctx).With().Str("worker", "http").Logger()
	logger.Info().Int("port", hw.port).Msg("Starting HTTP worker")

	// Start server in goroutine
	go func() {
		if err := hw.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	logger.Info().Msg("Gracefully shutting down HTTP worker...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown server
	if err := hw.server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Failed to shutdown HTTP server gracefully")
	} else {
		logger.Info().Msg("HTTP server shutdown complete")
	}

	// Close database connection
	if err := hw.db.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close database connection")
	} else {
		logger.Info().Msg("Database connection closed")
	}
}

// PollingWorker represents the polling worker
type PollingWorker struct {
	redis  *redis.Client
	ticker *time.Ticker
}

func NewPollingWorker(ctx context.Context, redisAddr string) (*PollingWorker, error) {
	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &PollingWorker{
		redis:  rdb,
		ticker: time.NewTicker(5 * time.Second),
	}, nil
}

func (pw *PollingWorker) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := zerolog.Ctx(ctx).With().Str("worker", "polling").Logger()
	logger.Info().Msg("Starting polling worker")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Gracefully shutting down polling worker...")
			pw.ticker.Stop()

			// Close Redis connection
			if err := pw.redis.Close(); err != nil {
				logger.Error().Err(err).Msg("Failed to close redis connection")
			} else {
				logger.Info().Msg("Redis connection closed")
			}
			return

		case <-pw.ticker.C:
			pw.poll(ctx)
		}
	}
}

func (pw *PollingWorker) poll(ctx context.Context) {
	logger := zerolog.Ctx(ctx).With().Str("worker", "polling").Logger()

	// Simulate polling work
	pollCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Example: Set a value in Redis
	err := pw.redis.Set(pollCtx, "last_poll", time.Now().Unix(), time.Minute).Err()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update poll timestamp")
		return
	}

	logger.Debug().Msg("Polling completed successfully")
}

// Configuration struct
type Config struct {
	Port      int
	LogLevel  string
	Env       string
	DBConnStr string
	RedisAddr string
}

func initConfig() *Config {
	return &Config{
		Port:      8080,
		LogLevel:  "info",
		Env:       "development",
		DBConnStr: "postgres://user:password@localhost/dbname?sslmode=disable",
		RedisAddr: "localhost:6379",
	}
}

func initLogger(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var logLevel zerolog.Level
	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	return log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(logLevel)
}

func main() {
	// Initialize configuration and logger
	cfg := initConfig()
	logger := initLogger(cfg.LogLevel)

	logger.Info().
		Str("env", cfg.Env).
		Int("port", cfg.Port).
		Msg("Starting application")

	// Create root context with logger
	rootCtx, rootCancel := context.WithCancel(context.Background())
	rootCtx = logger.WithContext(rootCtx)
	wg := sync.WaitGroup{}

	// Create and start HTTP worker
	httpWorker, err := NewHTTPWorker(rootCtx, cfg.Port, cfg.DBConnStr)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create HTTP worker")
	}

	wg.Add(1)
	go httpWorker.Run(rootCtx, &wg)

	// Create and start polling worker
	pollingWorker, err := NewPollingWorker(rootCtx, cfg.RedisAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create polling worker")
	}

	wg.Add(1)
	go pollingWorker.Run(rootCtx, &wg)

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

	logger.Info().Msg("Application shutdown complete")
}
