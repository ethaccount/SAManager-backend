package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ethaccount/backend/src/handler"
	"github.com/ethaccount/backend/src/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/ethaccount/backend/docs/swagger"
	_ "github.com/ethaccount/backend/docs/swagger"
	"github.com/joho/godotenv"
)

// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.basic  BasicAuth

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/

const (
	AppName    = "SAManager Backend"
	AppVersion = "0.0.1"
	AppBuild   = "dev"
)

func initAppConfig() service.AppConfig {

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

	// RPC URLs
	sepoliaRPCURL := os.Getenv("SEPOLIA_RPC_URL")
	if sepoliaRPCURL == "" {
		sepoliaRPCURL = "https://ethereum-sepolia-rpc.publicnode.com"
	}

	arbitrumSepoliaRPCURL := os.Getenv("ARBITRUM_SEPOLIA_RPC_URL")
	if arbitrumSepoliaRPCURL == "" {
		arbitrumSepoliaRPCURL = "https://arbitrum-sepolia-rpc.publicnode.com"
	}

	baseSepoliaRPCURL := os.Getenv("BASE_SEPOLIA_RPC_URL")
	if baseSepoliaRPCURL == "" {
		baseSepoliaRPCURL = "https://base-sepolia-rpc.publicnode.com"
	}

	optimismSepoliaRPCURL := os.Getenv("OPTIMISM_SEPOLIA_RPC_URL")
	if optimismSepoliaRPCURL == "" {
		optimismSepoliaRPCURL = "https://optimism-sepolia-rpc.publicnode.com"
	}

	polygonAmoyRPCURL := os.Getenv("POLYGON_AMOY_RPC_URL")
	if polygonAmoyRPCURL == "" {
		polygonAmoyRPCURL = "https://polygon-amoy-rpc.publicnode.com"
	}

	// check if all RPC URLs are set
	if sepoliaRPCURL == "" || arbitrumSepoliaRPCURL == "" || baseSepoliaRPCURL == "" || optimismSepoliaRPCURL == "" || polygonAmoyRPCURL == "" {
		log.Fatalf("One or more RPC URLs are not set in .env file")
	}

	return service.AppConfig{
		LogLevel:              &logLevel,
		DSN:                   &dsn,
		Port:                  &port,
		SepoliaRPCURL:         &sepoliaRPCURL,
		ArbitrumSepoliaRPCURL: &arbitrumSepoliaRPCURL,
		BaseSepoliaRPCURL:     &baseSepoliaRPCURL,
		OptimismSepoliaRPCURL: &optimismSepoliaRPCURL,
		PolygonAmoyRPCURL:     &polygonAmoyRPCURL,
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
		Logger()

	return rootLogger
}

func main() {
	// Setup app configuration
	cfg := initAppConfig()

	// Update swagger info dynamically using constants
	swagger.SwaggerInfo.Title = AppName + " API"
	swagger.SwaggerInfo.Version = AppVersion
	swagger.SwaggerInfo.Description = fmt.Sprintf("%s with automation service and webauthn authentication", AppName)
	if cfg.Port != nil {
		swagger.SwaggerInfo.Host = fmt.Sprintf("localhost:%s", *cfg.Port)
	}

	// Create root logger
	rootLogger := initRootLogger(*cfg.LogLevel)

	// Create root context
	rootCtx := context.Background()
	rootCtx = rootLogger.WithContext(rootCtx)

	rootLogger.Info().
		Str("version", AppVersion).
		Str("build", AppBuild).
		Msgf("Launching %s", AppName)

	rootLogger.Info().
		Str("swagger_link", "http://localhost:"+*cfg.Port+"/swagger/index.html").
		Msg("Swagger link")

	// Create application
	app := service.NewApplication(rootCtx, cfg)

	ginRouter := gin.Default()
	handler.RegisterRoutes(rootCtx, ginRouter, app)
	ginRouter.Run(":" + *cfg.Port)
}
