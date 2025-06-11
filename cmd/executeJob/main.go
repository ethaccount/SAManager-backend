package main

import (
	"context"
	"log"
	"time"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethaccount/backend/src/app"
	"github.com/ethaccount/backend/src/repository"
	"github.com/ethaccount/backend/src/service"
	"github.com/ethereum/go-ethereum/common"
	"github.com/joho/godotenv"
	postgresDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const JOB_ID = "a02461b1-54b2-4e25-9648-a845618ba33b"

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
		log.Println("Proceeding with environment variables from system...")
	}

	// Load configuration
	config := app.NewAppConfig()

	// Setup logger
	logger := app.InitLogger(*config.LogLevel)

	// Initialize context
	ctx := context.Background()
	ctx = logger.WithContext(ctx)

	// Connect to database
	database, err := gorm.Open(postgresDriver.Open(*config.DSN), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Test database connection
	db, err := database.DB()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get underlying database connection")
	}

	if err := db.Ping(); err != nil {
		logger.Fatal().Err(err).Msg("Database ping failed")
	}

	// Defer closing database connection
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close database connection")
		} else {
			logger.Info().Msg("Database connection closed")
		}
	}()

	logger.Info().Msg("Database connection established")

	// Initialize services
	jobRepo := repository.NewJobRepository(database)
	jobService := service.NewJobService(jobRepo)

	// Initialize blockchain service
	blockchainService := service.NewBlockchainService(service.BlockchainConfig{
		SepoliaRPCURL:         *config.SepoliaRPCURL,
		ArbitrumSepoliaRPCURL: *config.ArbitrumSepoliaRPCURL,
		BaseSepoliaRPCURL:     *config.BaseSepoliaRPCURL,
		OptimismSepoliaRPCURL: *config.OptimismSepoliaRPCURL,
		PolygonAmoyRPCURL:     *config.PolygonAmoyRPCURL,
	})

	// Initialize execution service
	executionService, err := service.NewExecutionService(blockchainService, *config.PrivateKey)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create execution service")
	}

	// Get job by ID
	logger.Info().Str("job_id", JOB_ID).Msg("Retrieving job from database")
	job, err := jobService.GetJobByID(ctx, JOB_ID)
	if err != nil {
		logger.Fatal().Err(err).Str("job_id", JOB_ID).Msg("Failed to retrieve job")
	}

	logger.Info().
		Str("job_id", job.ID.String()).
		Str("account_address", job.AccountAddress.Hex()).
		Int64("chain_id", job.ChainID).
		Int64("on_chain_job_id", job.OnChainJobID).
		Msg("Job retrieved successfully")

	// Execute the job
	logger.Info().Str("job_id", JOB_ID).Msg("Executing job")
	userOpHash, err := executionService.ExecuteJob(ctx, *job)
	if err != nil {
		logger.Fatal().Err(err).Str("job_id", JOB_ID).Msg("Failed to execute job")
	}

	logger.Info().
		Str("job_id", JOB_ID).
		Str("user_op_hash", userOpHash).
		Msg("Job executed successfully")

	// Wait for user operation receipt using blockchain service
	logger.Info().Str("user_op_hash", userOpHash).Msg("Waiting for user operation receipt...")
	maxAttempts := 60
	pollInterval := 2 * time.Second

	// Get bundler client
	bundlerClient, err := blockchainService.GetBundlerClient(ctx, job.ChainID)
	if err != nil {
		logger.Fatal().Err(err).
			Int64("chain_id", job.ChainID).
			Msg("Failed to get bundler client")
	}

	receipt, err := bundlerClient.(*erc4337.BundlerClient).WaitForUserOpReceipt(ctx, userOpHash, maxAttempts, pollInterval)
	if err != nil {
		logger.Fatal().Err(err).
			Str("user_op_hash", userOpHash).
			Int("max_attempts", maxAttempts).
			Msg("Failed to get user operation receipt")
	}

	logger.Info().Msg("User Operation Receipt received!")
	logger.Info().
		Str("user_op_hash", receipt.UserOpHash.Hex()).
		Str("sender", receipt.Sender.Hex()).
		Bool("success", receipt.Success).
		Str("actual_gas_cost", receipt.ActualGasCost).
		Str("actual_gas_used", receipt.ActualGasUsed).
		Str("nonce", receipt.Nonce).
		Msg("Receipt details")

	if receipt.Paymaster != (common.Address{}) {
		logger.Info().Str("paymaster", receipt.Paymaster.Hex()).Msg("Paymaster used")
	}

	if receipt.Receipt != nil {
		logger.Info().
			Str("transaction_hash", receipt.Receipt.TransactionHash.Hex()).
			Str("block_number", receipt.Receipt.BlockNumber).
			Str("gas_used", receipt.Receipt.GasUsed).
			Msg("Transaction details")
	}

	// Check if receipt.Success is true
	if receipt.Success {
		logger.Info().Msg("User operation executed successfully!")
	} else {
		logger.Error().Msg("User operation failed!")
	}
}
