package app

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	LogLevel *string

	// Database configuration
	DSN *string

	// Redis configuration
	RedisAddr *string

	// HTTP configuration
	Port *string

	// Polling configuration
	PollingInterval *int

	// RPC URLs
	SepoliaRPCURL         *string
	ArbitrumSepoliaRPCURL *string
	BaseSepoliaRPCURL     *string
	OptimismSepoliaRPCURL *string
	PolygonAmoyRPCURL     *string

	// Private key for signing user operations
	PrivateKey *string
}

func NewAppConfig() *AppConfig {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatalf("DB_URL not set in .env file")
	}

	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		log.Fatalf("REDIS_URL not set in .env file")
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

	// Private key for signing user operations
	privateKey := os.Getenv("PRIVATE_KEY")
	if privateKey == "" {
		log.Fatalf("PRIVATE_KEY not set in .env file")
	}

	// remove 0x prefix if it exists
	privateKey = strings.TrimPrefix(privateKey, "0x")

	// Add polling interval configuration
	pollingIntervalStr := os.Getenv("POLLING_INTERVAL")
	pollingInterval := 60 // default to 1 minute
	if pollingIntervalStr != "" {
		if parsed, err := strconv.Atoi(pollingIntervalStr); err == nil {
			pollingInterval = parsed
		}
	}

	return &AppConfig{
		LogLevel:              &logLevel,
		DSN:                   &dsn,
		RedisAddr:             &redisAddr,
		Port:                  &port,
		PollingInterval:       &pollingInterval,
		SepoliaRPCURL:         &sepoliaRPCURL,
		ArbitrumSepoliaRPCURL: &arbitrumSepoliaRPCURL,
		BaseSepoliaRPCURL:     &baseSepoliaRPCURL,
		OptimismSepoliaRPCURL: &optimismSepoliaRPCURL,
		PolygonAmoyRPCURL:     &polygonAmoyRPCURL,
		PrivateKey:            &privateKey,
	}
}
