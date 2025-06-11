package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/ethaccount/backend/src/handler"
	"github.com/ethaccount/backend/src/repository"
	"github.com/ethaccount/backend/src/service"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/shopspring/decimal"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/rs/zerolog"
	postgresDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Application struct {
	config         AppConfig
	database       *gorm.DB
	redis          *redis.Client
	PasskeyService *service.PasskeyService
	JobService     *service.JobService
	Scheduler      *service.JobScheduler
}

func NewApplication(ctx context.Context, config AppConfig) *Application {
	logger := zerolog.Ctx(ctx).With().Str("function", "NewApplication").Logger()

	// Connect to Redis
	redisOpts, err := redis.ParseURL(*config.RedisAddr)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse redis URL")
		return nil
	}

	rdb := redis.NewClient(redisOpts)

	// Test Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error().Err(err).Msg("connection to redis failed")
		return nil
	}
	logger.Info().Msg("Redis connection established")

	// Connect to database
	database, err := gorm.Open(postgresDriver.Open(*config.DSN), &gorm.Config{})
	if err != nil {
		logger.Error().Err(err).Msg("connection to database failed")
		return nil
	}

	// Test database connection
	db, err := database.DB()
	if err != nil {
		logger.Error().Err(err).Msg("failed to get underlying database connection")
		return nil
	}

	if err := db.Ping(); err != nil {
		logger.Error().Err(err).Msg("connection to database failed")
		return nil
	}

	logger.Info().Msg("Database connection established")

	// run migration files
	migrationPath := "file://migrations"

	MigrationUp(*config.DSN, migrationPath)

	passkeyRepo := repository.NewPasskeyRepository(database)

	webAuthnConfig := &webauthn.Config{
		RPDisplayName: "Passkey Demo",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:" + *config.Port},
	}

	passkeyService, err := service.NewPasskeyService(ctx, passkeyRepo, webAuthnConfig, 5*time.Minute)
	if err != nil {
		logger.Error().Err(err).Msg("creation of passkey service failed")
		return nil
	}

	jobRepo := repository.NewJobRepository(database)
	jobService := service.NewJobService(jobRepo)

	blockchainService := service.NewBlockchainService(service.BlockchainConfig{
		SepoliaRPCURL:         *config.SepoliaRPCURL,
		ArbitrumSepoliaRPCURL: *config.ArbitrumSepoliaRPCURL,
		BaseSepoliaRPCURL:     *config.BaseSepoliaRPCURL,
		OptimismSepoliaRPCURL: *config.OptimismSepoliaRPCURL,
		PolygonAmoyRPCURL:     *config.PolygonAmoyRPCURL,
	})

	executionService, err := service.NewExecutionService(blockchainService, *config.PrivateKey)
	if err != nil {
		log.Fatalf("failed to create execution service: %v", err)
	}

	scheduler := service.NewJobScheduler(ctx, rdb, "job_queue", *config.PollingInterval, jobService, executionService, blockchainService)

	return &Application{
		config:         config,
		database:       database,
		redis:          rdb,
		PasskeyService: passkeyService,
		JobService:     jobService,
		Scheduler:      scheduler,
	}
}

func (app *Application) Shutdown(ctx context.Context) {
	logger := zerolog.Ctx(ctx).With().Str("function", "Shutdown").Logger()

	// Close database connection
	if app.database != nil {
		db, err := app.database.DB()
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get underlying database connection")
		} else {
			if err := db.Close(); err != nil {
				logger.Error().Err(err).Msg("Failed to close database connection")
			} else {
				logger.Info().Msg("Database connection closed")
			}
		}
	}

	// Close Redis connection
	if app.redis != nil {
		if err := app.redis.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close redis connection")
		} else {
			logger.Info().Msg("Redis connection closed")
		}
	}
}

func (app *Application) RunHTTPServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := zerolog.Ctx(ctx).With().Str("function", "RunHTTPServer").Logger()

	// Set to release mode to disable Gin logger
	gin.SetMode(gin.ReleaseMode)

	ginRouter := gin.Default()

	// Register routes
	app.registerRoutes(ctx, ginRouter)

	// Build HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", *app.config.Port),
		Handler: ginRouter,
	}

	// Start server in goroutine
	go func() {
		zerolog.Ctx(ctx).Info().Msgf("HTTP server is on http://localhost:%s/api/v1/health", *app.config.Port)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			zerolog.Ctx(ctx).Panic().Err(err).Msg("Failed to start HTTP server")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	logger.Info().Msg("Gracefully shutting down HTTP server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Failed to shutdown HTTP server gracefully")
	} else {
		logger.Info().Msg("HTTP server shutdown complete")
	}
}

func (app *Application) RunPollingWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := zerolog.Ctx(ctx).With().Str("function", "RunPollingWorker").Logger()
	logger.Info().Msg("Starting polling worker")

	app.Scheduler.Start()

	<-ctx.Done()
	logger.Info().Msg("Stopping polling worker...")

	app.Scheduler.Stop()

	logger.Info().Msg("Polling worker stopped")
}

func (app *Application) registerRoutes(ctx context.Context, router *gin.Engine) {

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
			if value, ok := field.Interface().(decimal.Decimal); ok {
				return value.String()
			}
			return nil
		}, decimal.Decimal{})
	}

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = *app.config.AllowOrigins
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	config.AllowCredentials = true

	router.Use(cors.New(config))

	handler.SetMiddlewares(ctx, router)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	passkeyHandler := handler.NewPasskeyHandler(app.PasskeyService)
	jobHandler := handler.NewJobHandler(app.JobService)

	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", handler.HandleHealthCheck)

		v1.POST("/register/begin", passkeyHandler.RegisterBegin())
		// v1.POST("/register/verify", passkeyHandler.RegisterVerify)
		// v1.POST("/login/options", passkeyHandler.LoginOptions)
		// v1.POST("/login/verify", passkeyHandler.LoginVerify)

		// Job management endpoints
		v1.GET("/jobs", jobHandler.GetJobList)
		v1.POST("/jobs", jobHandler.RegisterJob)
	}
}
