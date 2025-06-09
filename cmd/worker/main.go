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

// WorkerManager manages the lifecycle of individual workers
type WorkerManager struct {
	name    string
	ctx     context.Context
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
	logger  zerolog.Logger
	running bool
	mu      sync.RWMutex
}

func NewWorkerManager(name string, parentCtx context.Context, logger zerolog.Logger) *WorkerManager {
	ctx, cancel := context.WithCancel(parentCtx)
	return &WorkerManager{
		name:   name,
		ctx:    ctx,
		cancel: cancel,
		wg:     &sync.WaitGroup{},
		logger: logger.With().Str("worker", name).Logger(),
	}
}

func (sm *WorkerManager) Start() {
	sm.mu.Lock()
	sm.running = true
	sm.mu.Unlock()
}

func (sm *WorkerManager) Stop() {
	sm.mu.Lock()
	if sm.running {
		sm.logger.Info().Msg("Stopping worker...")
		sm.cancel()
		sm.running = false
	}
	sm.mu.Unlock()
}

func (sm *WorkerManager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.running
}

func (sm *WorkerManager) Wait() {
	sm.wg.Wait()
}

// HTTPWorker represents the HTTP server worker
type HTTPWorker struct {
	*WorkerManager
	server *http.Server
	db     *sql.DB
	port   int
}

func NewHTTPWorker(parentCtx context.Context, logger zerolog.Logger, port int, dbDSN string) (*HTTPWorker, error) {
	sm := NewWorkerManager("http", parentCtx, logger)

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
		WorkerManager: sm,
		server:        server,
		db:            db,
		port:          port,
	}, nil
}

func (hs *HTTPWorker) Run() {
	hs.wg.Add(1)
	defer hs.wg.Done()

	hs.Start()
	hs.logger.Info().Int("port", hs.port).Msg("Starting HTTP server")

	// Start server in goroutine
	go func() {
		if err := hs.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			hs.logger.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Wait for context cancellation
	<-hs.ctx.Done()

	hs.logger.Info().Msg("Gracefully shutting down HTTP server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown server
	if err := hs.server.Shutdown(shutdownCtx); err != nil {
		hs.logger.Error().Err(err).Msg("Failed to shutdown HTTP server gracefully")
	} else {
		hs.logger.Info().Msg("HTTP server shutdown complete")
	}

	// Close database connection
	if err := hs.db.Close(); err != nil {
		hs.logger.Error().Err(err).Msg("Failed to close database connection")
	} else {
		hs.logger.Info().Msg("Database connection closed")
	}
}

// PollingWorker represents the polling worker
type PollingWorker struct {
	*WorkerManager
	redis  *redis.Client
	ticker *time.Ticker
}

func NewPollingWorker(parentCtx context.Context, logger zerolog.Logger, redisAddr string) (*PollingWorker, error) {
	sm := NewWorkerManager("polling", parentCtx, logger)

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test connection
	if err := rdb.Ping(parentCtx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &PollingWorker{
		WorkerManager: sm,
		redis:         rdb,
		ticker:        time.NewTicker(5 * time.Second),
	}, nil
}

func (ps *PollingWorker) Run() {
	ps.wg.Add(1)
	defer ps.wg.Done()

	ps.Start()
	ps.logger.Info().Msg("Starting polling worker")

	for {
		select {
		case <-ps.ctx.Done():
			ps.logger.Info().Msg("Gracefully shutting down polling worker...")
			ps.ticker.Stop()

			// Close Redis connection
			if err := ps.redis.Close(); err != nil {
				ps.logger.Error().Err(err).Msg("Failed to close redis connection")
			} else {
				ps.logger.Info().Msg("Redis connection closed")
			}
			return

		case <-ps.ticker.C:
			ps.poll()
		}
	}
}

func (ps *PollingWorker) poll() {
	// Simulate polling work
	ctx, cancel := context.WithTimeout(ps.ctx, 3*time.Second)
	defer cancel()

	// Example: Set a value in Redis
	err := ps.redis.Set(ctx, "last_poll", time.Now().Unix(), time.Minute).Err()
	if err != nil {
		ps.logger.Error().Err(err).Msg("Failed to update poll timestamp")
		return
	}

	ps.logger.Debug().Msg("Polling completed successfully")
}

// WorkerController manages multiple workers and handles shutdown commands
type WorkerController struct {
	workers map[string]Worker
	logger  zerolog.Logger
	mu      sync.RWMutex
}

type Worker interface {
	Run()
	Stop()
	IsRunning() bool
	Wait()
}

func NewWorkerController(logger zerolog.Logger) *WorkerController {
	return &WorkerController{
		workers: make(map[string]Worker),
		logger:  logger,
	}
}

func (sc *WorkerController) AddWorker(name string, worker Worker) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.workers[name] = worker
}

func (sc *WorkerController) StartAll() {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for name, worker := range sc.workers {
		go func(name string, svc Worker) {
			sc.logger.Info().Str("worker", name).Msg("Starting worker")
			svc.Run()
			sc.logger.Info().Str("worker", name).Msg("Worker stopped")
		}(name, worker)
	}
}

func (sc *WorkerController) StopWorker(name string) bool {
	sc.mu.RLock()
	worker, exists := sc.workers[name]
	sc.mu.RUnlock()

	if !exists {
		sc.logger.Warn().Str("worker", name).Msg("Worker not found")
		return false
	}

	worker.Stop()
	sc.logger.Info().Str("worker", name).Msg("Stop signal sent to worker")
	return true
}

func (sc *WorkerController) StopAll() {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for name, worker := range sc.workers {
		sc.logger.Info().Str("worker", name).Msg("Stopping worker")
		worker.Stop()
	}
}

func (sc *WorkerController) WaitAll() {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for name, worker := range sc.workers {
		sc.logger.Info().Str("worker", name).Msg("Waiting for worker to stop")
		worker.Wait()
	}
}

func (sc *WorkerController) IsAnyRunning() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	for _, worker := range sc.workers {
		if worker.IsRunning() {
			return true
		}
	}
	return false
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

	// Create root context
	rootCtx, rootCancel := context.WithCancel(context.Background())

	// Initialize worker controller
	controller := NewWorkerController(logger)

	// Create and add HTTP worker
	httpWorker, err := NewHTTPWorker(rootCtx, logger, cfg.Port, cfg.DBConnStr)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create HTTP worker")
	}
	controller.AddWorker("http", httpWorker)

	// Create and add polling worker
	pollingWorker, err := NewPollingWorker(rootCtx, logger, cfg.RedisAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create polling worker")
	}
	controller.AddWorker("polling", pollingWorker)

	// Start all workers
	controller.StartAll()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Setup command channel for individual worker control
	// In a real application, this could be a HTTP endpoint, gRPC worker, or Unix socket
	commandChan := make(chan string, 1)

	// Example: Simulate receiving commands (in real app, this would come from external source)
	go func() {
		// Uncomment and modify these lines to test individual worker stopping:
		// time.Sleep(15 * time.Second)
		// commandChan <- "stop:http"    // Stop only HTTP worker
		// time.Sleep(5 * time.Second)
		// commandChan <- "stop:polling" // Stop only polling worker
	}()

	// Main event loop
	for {
		select {
		case sig := <-sigChan:
			logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
			controller.StopAll()
			rootCancel()

		case cmd := <-commandChan:
			logger.Info().Str("command", cmd).Msg("Received command")
			if cmd == "stop:http" {
				controller.StopWorker("http")
			} else if cmd == "stop:polling" {
				controller.StopWorker("polling")
			} else if cmd == "stop:all" {
				controller.StopAll()
				rootCancel()
			} else {
				logger.Warn().Str("command", cmd).Msg("Unknown command")
			}
		}

		// Check if all workers have stopped
		if !controller.IsAnyRunning() {
			logger.Info().Msg("All workers stopped, shutting down")
			break
		}
	}

	// Wait for all workers to complete with timeout
	waitChan := make(chan struct{})
	go func() {
		controller.WaitAll()
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
