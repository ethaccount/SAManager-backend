package app

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethaccount/backend/src/utils"
)

type AppConfig struct {
	// =========================== REQUIRED ===========================

	// Environment configuration (required)
	Environment *string
	// Database configuration (required)
	DSN *string
	// Redis configuration (required)
	RedisURL *string
	// Private key for signing user operations (required)
	PrivateKey *string
	// API secret for validating requests from frontend (required)
	APISecret *string
	// CORS configuration (required)
	AllowOrigins *[]string

	// =========================== OPTIONAL ===========================

	// Logging configuration
	LogLevel *string

	// HTTP server configuration
	Port *string
	Host *string

	// Polling configuration
	PollingInterval *int

	// Migration configuration
	MigrationPath *string

	// WebAuthn configuration
	RPDisplayName *string
	RPID          *string
	RPOrigins     *[]string

	// Blockchain RPC URLs (all have defaults)
	SepoliaRPCURL         *string
	ArbitrumSepoliaRPCURL *string
	BaseSepoliaRPCURL     *string
	OptimismSepoliaRPCURL *string
	PolygonAmoyRPCURL     *string
}

func NewAppConfig() *AppConfig {
	config := &AppConfig{}

	// Load required configuration
	loadRequiredConfig(config)

	// Load optional configuration with defaults
	loadOptionalConfig(config)

	return config
}

// loadRequiredConfig loads all required configuration values and fails fast if any are missing
func loadRequiredConfig(config *AppConfig) {
	// Database URL (required)
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatalf("REQUIRED: DB_URL not set in environment")
	}
	config.DSN = &dsn

	// Redis URL (required)
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatalf("REQUIRED: REDIS_URL not set in environment")
	}
	config.RedisURL = &redisURL

	// Private key for signing operations (required)
	privateKey := os.Getenv("PRIVATE_KEY")
	if privateKey == "" {
		log.Fatalf("REQUIRED: PRIVATE_KEY not set in environment")
	}
	// Remove 0x prefix if it exists
	privateKey = strings.TrimPrefix(privateKey, "0x")
	config.PrivateKey = &privateKey

	// API secret for validating requests from frontend (required)
	apiSecret := os.Getenv("API_SECRET")
	if apiSecret == "" {
		log.Fatalf("REQUIRED: API_SECRET not set in environment")
	}
	config.APISecret = &apiSecret

	// CORS origins (required in production, optional in development)
	loadCORSConfig(config)

	// Environment (required)
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		log.Fatalf("REQUIRED: ENVIRONMENT not set in environment")
	}
	// Validate environment value
	if environment != "dev" && environment != "staging" && environment != "prod" {
		log.Fatalf("REQUIRED: ENVIRONMENT must be one of: dev, staging, prod (got: %s)", environment)
	}
	config.Environment = &environment
}

// loadOptionalConfig loads all optional configuration values with sensible defaults
func loadOptionalConfig(config *AppConfig) {
	// HTTP server port (default: 8080)
	port := getEnvWithDefault("PORT", "8080")
	config.Port = &port

	// Host configuration - environment aware
	loadHostConfig(config)

	// Log level (default: debug)
	// Available levels: "trace", "debug", "info", "warn", "error", "fatal", "panic", "disabled"
	logLevel := getEnvWithDefault("LOG_LEVEL", "debug")
	config.LogLevel = &logLevel

	// Polling interval in seconds (default: 60)
	pollingInterval := getPollingInterval()
	config.PollingInterval = &pollingInterval

	// Migration path
	migrationPath := getEnvWithDefault("MIGRATION_PATH", "file://"+filepath.Join(utils.FindProjectRoot(), "migrations"))
	config.MigrationPath = &migrationPath

	// Load WebAuthn configuration
	loadWebAuthnConfig(config)

	// Load blockchain RPC URLs with defaults
	loadRPCConfig(config)
}

// loadCORSConfig handles CORS origins configuration
func loadCORSConfig(config *AppConfig) {
	allowOriginsStr := os.Getenv("ALLOW_ORIGINS")
	if allowOriginsStr == "" {
		log.Fatalf("REQUIRED: ALLOW_ORIGINS not set in environment")
	}

	var allowOrigins []string
	// Parse comma-separated origins
	origins := strings.Split(allowOriginsStr, ",")
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowOrigins = append(allowOrigins, origin)
		}
	}

	config.AllowOrigins = &allowOrigins
}

// loadWebAuthnConfig loads WebAuthn configuration with sensible defaults
func loadWebAuthnConfig(config *AppConfig) {
	// WebAuthn RP Display Name
	rpDisplayName := getEnvWithDefault("WEBAUTHN_RP_DISPLAY_NAME", "SAManager Passkey")
	config.RPDisplayName = &rpDisplayName

	// WebAuthn RP ID
	rpID := getEnvWithDefault("WEBAUTHN_RP_ID", "localhost")
	config.RPID = &rpID

	// WebAuthn RP Origins - build from environment or use port-based default
	loadWebAuthnOrigins(config)
}

// loadWebAuthnOrigins handles WebAuthn origins configuration
func loadWebAuthnOrigins(config *AppConfig) {
	originsStr := os.Getenv("WEBAUTHN_RP_ORIGINS")
	var origins []string

	if originsStr != "" {
		// Parse comma-separated origins
		originList := strings.Split(originsStr, ",")
		for _, origin := range originList {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				origins = append(origins, origin)
			}
		}
	} else {
		// Default to localhost with the configured port
		origins = []string{"http://localhost:" + *config.Port}
	}

	config.RPOrigins = &origins
}

// loadRPCConfig loads blockchain RPC URLs with public node defaults
func loadRPCConfig(config *AppConfig) {
	sepoliaRPCURL := getEnvWithDefault("SEPOLIA_RPC_URL", "https://ethereum-sepolia-rpc.publicnode.com")
	config.SepoliaRPCURL = &sepoliaRPCURL

	arbitrumSepoliaRPCURL := getEnvWithDefault("ARBITRUM_SEPOLIA_RPC_URL", "https://arbitrum-sepolia-rpc.publicnode.com")
	config.ArbitrumSepoliaRPCURL = &arbitrumSepoliaRPCURL

	baseSepoliaRPCURL := getEnvWithDefault("BASE_SEPOLIA_RPC_URL", "https://base-sepolia-rpc.publicnode.com")
	config.BaseSepoliaRPCURL = &baseSepoliaRPCURL

	optimismSepoliaRPCURL := getEnvWithDefault("OPTIMISM_SEPOLIA_RPC_URL", "https://optimism-sepolia-rpc.publicnode.com")
	config.OptimismSepoliaRPCURL = &optimismSepoliaRPCURL

	polygonAmoyRPCURL := getEnvWithDefault("POLYGON_AMOY_RPC_URL", "https://polygon-amoy-rpc.publicnode.com")
	config.PolygonAmoyRPCURL = &polygonAmoyRPCURL
}

// getPollingInterval parses polling interval from environment with default fallback
func getPollingInterval() int {
	pollingIntervalStr := os.Getenv("POLLING_INTERVAL")
	if pollingIntervalStr == "" {
		return 60 // default to 1 minute
	}

	if parsed, err := strconv.Atoi(pollingIntervalStr); err == nil {
		return parsed
	}

	log.Printf("Warning: Invalid POLLING_INTERVAL value '%s', using default 60 seconds", pollingIntervalStr)
	return 60
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadHostConfig handles host configuration with environment-aware defaults
func loadHostConfig(config *AppConfig) {
	hostEnv := os.Getenv("HOST")
	var host string

	if hostEnv != "" {
		host = hostEnv
	} else {
		// Environment-aware defaults
		switch *config.Environment {
		case "dev":
			host = "localhost:" + *config.Port
		case "staging", "prod":
			// For deployed environments, assume HTTPS and proper domain
			// User should set HOST env var for production deployments
			host = "localhost:" + *config.Port // fallback
			log.Printf("Warning: HOST environment variable not set for %s environment. Using fallback '%s'. Consider setting HOST for proper deployment.", *config.Environment, host)
		default:
			host = "localhost:" + *config.Port
		}
	}

	config.Host = &host
}
