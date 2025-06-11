package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethaccount/backend/src/testutil"
	"github.com/google/uuid"
)

func getBlockchainService() *BlockchainService {
	sepoliaRpcUrl := testutil.GetEnv("SEPOLIA_RPC_URL")
	arbitrumSepoliaRpcUrl := testutil.GetEnv("ARBITRUM_SEPOLIA_RPC_URL")
	baseSepoliaRpcUrl := testutil.GetEnv("BASE_SEPOLIA_RPC_URL")
	optimismSepoliaRpcUrl := testutil.GetEnv("OPTIMISM_SEPOLIA_RPC_URL")
	polygonAmoyRpcUrl := testutil.GetEnv("POLYGON_AMOY_RPC_URL")

	blockchainService := NewBlockchainService(BlockchainConfig{
		SepoliaRPCURL:         sepoliaRpcUrl,
		ArbitrumSepoliaRPCURL: arbitrumSepoliaRpcUrl,
		BaseSepoliaRPCURL:     baseSepoliaRpcUrl,
		OptimismSepoliaRPCURL: optimismSepoliaRpcUrl,
		PolygonAmoyRPCURL:     polygonAmoyRpcUrl,
	})

	return blockchainService
}

func TestGetExecutionConfig(t *testing.T) {
	blockchainService := getBlockchainService()

	// Create test job with the provided data
	job := &domain.Job{
		ID:                uuid.New(),
		AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
		ChainID:           11155111, // Sepolia testnet
		OnChainJobID:      1,
		UserOperation:     json.RawMessage(`{"sender":"0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1","nonce":"0x1","callData":"0x","callGasLimit":"100000","verificationGasLimit":"50000","preVerificationGas":"21000","maxPriorityFeePerGas":"1000000000","maxFeePerGas":"2000000000","signature":"0x"}`),
		EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
	}

	ctx := context.Background()

	// Call GetExecutionConfig
	config, err := blockchainService.GetExecutionConfig(ctx, job)

	// Verify the call succeeds
	if err != nil {
		t.Fatalf("GetExecutionConfig failed: %v", err)
	}

	// Verify the returned config is not nil
	if config == nil {
		t.Fatal("GetExecutionConfig returned nil config")
	}

	// Verify config structure is properly populated
	if config.ExecuteInterval == nil {
		t.Error("ExecuteInterval should not be nil")
	}

	if config.StartDate == nil {
		t.Error("StartDate should not be nil")
	}

	if config.LastExecutionTime == nil {
		t.Error("LastExecutionTime should not be nil")
	}

	if config.ExecutionData == nil {
		t.Error("ExecutionData should not be nil")
	}

	// Log the execution config for debugging
	t.Logf("ExecutionConfig retrieved:")
	t.Logf("  ExecuteInterval: %s", config.ExecuteInterval.String())
	t.Logf("  NumberOfExecutions: %d", config.NumberOfExecutions)
	t.Logf("  NumberOfExecutionsCompleted: %d", config.NumberOfExecutionsCompleted)
	t.Logf("  StartDate: %s", config.StartDate.String())
	t.Logf("  IsEnabled: %t", config.IsEnabled)
	t.Logf("  LastExecutionTime: %s", config.LastExecutionTime.String())
	t.Logf("  ExecutionData length: %d bytes", len(config.ExecutionData))

	/*
		scheduled_transfers_test.go:53: ExecutionConfig retrieved:
		scheduled_transfers_test.go:54:   ExecuteInterval: 180
		scheduled_transfers_test.go:55:   NumberOfExecutions: 3
		scheduled_transfers_test.go:56:   NumberOfExecutionsCompleted: 2
		scheduled_transfers_test.go:57:   StartDate: 1748275200
		scheduled_transfers_test.go:58:   IsEnabled: true
		scheduled_transfers_test.go:59:   LastExecutionTime: 1748508348
		scheduled_transfers_test.go:60:   ExecutionData length: 96 bytes
	*/
}

func TestGetExecutionConfig_UnsupportedChain(t *testing.T) {
	ctx := context.Background()

	blockchainService := getBlockchainService()
	// Test with unsupported chain ID
	job := &domain.Job{
		ID:                uuid.New(),
		AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
		ChainID:           1, // Mainnet - not supported in the function
		OnChainJobID:      2,
		UserOperation:     json.RawMessage(`{}`),
		EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
	}

	// Call GetExecutionConfig - should fail
	config, err := blockchainService.GetExecutionConfig(ctx, job)

	// Verify it returns an error
	if err == nil {
		t.Fatal("Expected error for unsupported chain ID, got nil")
	}

	if config != nil {
		t.Fatal("Expected nil config for unsupported chain ID")
	}

	// Verify error message
	expectedError := "unsupported chain id: 1"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGetExecutionConfigsBatch_EmptyInput(t *testing.T) {
	ctx := context.Background()
	blockchainService := getBlockchainService()

	// Test with empty job slice
	configs, err := blockchainService.GetExecutionConfigsBatch(ctx, []*domain.Job{})

	// Should succeed with empty result
	if err != nil {
		t.Fatalf("GetExecutionConfigsBatch failed with empty input: %v", err)
	}

	if configs == nil {
		t.Fatal("GetExecutionConfigsBatch returned nil for empty input")
	}

	if len(configs) != 0 {
		t.Errorf("Expected empty result, got %d configs", len(configs))
	}
}

func TestGetExecutionConfigsBatch_SingleJob(t *testing.T) {
	ctx := context.Background()
	blockchainService := getBlockchainService()

	// Create single test job
	job := &domain.Job{
		ID:                uuid.New(),
		AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
		ChainID:           11155111, // Sepolia testnet
		OnChainJobID:      1,
		UserOperation:     json.RawMessage(`{"sender":"0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1","nonce":"0x1","callData":"0x","callGasLimit":"100000","verificationGasLimit":"50000","preVerificationGas":"21000","maxPriorityFeePerGas":"1000000000","maxFeePerGas":"2000000000","signature":"0x"}`),
		EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
	}

	// Call GetExecutionConfigsBatch
	configs, err := blockchainService.GetExecutionConfigsBatch(ctx, []*domain.Job{job})

	// Verify the call succeeds
	if err != nil {
		t.Fatalf("GetExecutionConfigsBatch failed: %v", err)
	}

	// Verify we got exactly one result
	if len(configs) != 1 {
		t.Fatalf("Expected 1 config, got %d", len(configs))
	}

	// Verify the config exists for our job
	config, exists := configs[job.ID.String()]
	if !exists {
		t.Fatal("Config not found for job ID")
	}

	// Verify config structure is properly populated
	if config.ExecuteInterval == nil {
		t.Error("ExecuteInterval should not be nil")
	}

	if config.StartDate == nil {
		t.Error("StartDate should not be nil")
	}

	if config.LastExecutionTime == nil {
		t.Error("LastExecutionTime should not be nil")
	}

	if config.ExecutionData == nil {
		t.Error("ExecutionData should not be nil")
	}

	t.Logf("Batch ExecutionConfig retrieved:")
	t.Logf("  ExecuteInterval: %s", config.ExecuteInterval.String())
	t.Logf("  NumberOfExecutions: %d", config.NumberOfExecutions)
	t.Logf("  NumberOfExecutionsCompleted: %d", config.NumberOfExecutionsCompleted)
	t.Logf("  StartDate: %s", config.StartDate.String())
	t.Logf("  IsEnabled: %t", config.IsEnabled)
	t.Logf("  LastExecutionTime: %s", config.LastExecutionTime.String())
	t.Logf("  ExecutionData length: %d bytes", len(config.ExecutionData))
}

func TestGetExecutionConfigsBatch_MultipleJobsSameChain(t *testing.T) {
	ctx := context.Background()
	blockchainService := getBlockchainService()

	// Create multiple test jobs on the same chain
	jobs := []*domain.Job{
		{
			ID:                uuid.New(),
			AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
			ChainID:           11155111, // Sepolia testnet
			OnChainJobID:      1,
			UserOperation:     json.RawMessage(`{"sender":"0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1","nonce":"0x1","callData":"0x","callGasLimit":"100000","verificationGasLimit":"50000","preVerificationGas":"21000","maxPriorityFeePerGas":"1000000000","maxFeePerGas":"2000000000","signature":"0x"}`),
			EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
		},
		{
			ID:                uuid.New(),
			AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
			ChainID:           11155111, // Sepolia testnet
			OnChainJobID:      2,
			UserOperation:     json.RawMessage(`{"sender":"0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1","nonce":"0x2","callData":"0x","callGasLimit":"100000","verificationGasLimit":"50000","preVerificationGas":"21000","maxPriorityFeePerGas":"1000000000","maxFeePerGas":"2000000000","signature":"0x"}`),
			EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
		},
	}

	// Call GetExecutionConfigsBatch
	configs, err := blockchainService.GetExecutionConfigsBatch(ctx, jobs)

	// Verify the call succeeds
	if err != nil {
		t.Fatalf("GetExecutionConfigsBatch failed: %v", err)
	}

	// Verify we got results for all jobs
	if len(configs) != len(jobs) {
		t.Fatalf("Expected %d configs, got %d", len(jobs), len(configs))
	}

	// Verify each job has a config
	for _, job := range jobs {
		config, exists := configs[job.ID.String()]
		if !exists {
			t.Errorf("Config not found for job ID %s", job.ID.String())
			continue
		}

		// Basic validation
		if config.ExecuteInterval == nil {
			t.Errorf("ExecuteInterval should not be nil for job %s", job.ID.String())
		}

		t.Logf("Job %s - ExecuteInterval: %s, NumberOfExecutions: %d",
			job.ID.String(), config.ExecuteInterval.String(), config.NumberOfExecutions)
	}
}

func TestGetExecutionConfigsBatch_MultipleJobsDifferentChains(t *testing.T) {
	ctx := context.Background()
	blockchainService := getBlockchainService()

	// Create test jobs on different chains
	jobs := []*domain.Job{
		{
			ID:                uuid.New(),
			AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
			ChainID:           11155111, // Sepolia testnet
			OnChainJobID:      1,
			UserOperation:     json.RawMessage(`{"sender":"0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1","nonce":"0x1","callData":"0x","callGasLimit":"100000","verificationGasLimit":"50000","preVerificationGas":"21000","maxPriorityFeePerGas":"1000000000","maxFeePerGas":"2000000000","signature":"0x"}`),
			EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
		},
		{
			ID:                uuid.New(),
			AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
			ChainID:           84532, // Base Sepolia
			OnChainJobID:      1,
			UserOperation:     json.RawMessage(`{"sender":"0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1","nonce":"0x1","callData":"0x","callGasLimit":"100000","verificationGasLimit":"50000","preVerificationGas":"21000","maxPriorityFeePerGas":"1000000000","maxFeePerGas":"2000000000","signature":"0x"}`),
			EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
		},
	}

	// Call GetExecutionConfigsBatch
	configs, err := blockchainService.GetExecutionConfigsBatch(ctx, jobs)

	// Verify the call succeeds
	if err != nil {
		t.Fatalf("GetExecutionConfigsBatch failed: %v", err)
	}

	// Verify we got results for all jobs
	if len(configs) != len(jobs) {
		t.Fatalf("Expected %d configs, got %d", len(jobs), len(configs))
	}

	// Verify each job has a config
	for _, job := range jobs {
		config, exists := configs[job.ID.String()]
		if !exists {
			t.Errorf("Config not found for job ID %s on chain %d", job.ID.String(), job.ChainID)
			continue
		}

		// Basic validation
		if config.ExecuteInterval == nil {
			t.Errorf("ExecuteInterval should not be nil for job %s", job.ID.String())
		}

		t.Logf("Job %s (Chain %d) - ExecuteInterval: %s, NumberOfExecutions: %d",
			job.ID.String(), job.ChainID, config.ExecuteInterval.String(), config.NumberOfExecutions)
	}
}

func TestGetExecutionConfigsBatch_UnsupportedChain(t *testing.T) {
	ctx := context.Background()
	blockchainService := getBlockchainService()

	// Create test job with unsupported chain
	job := &domain.Job{
		ID:                uuid.New(),
		AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
		ChainID:           1, // Mainnet - not supported
		OnChainJobID:      1,
		UserOperation:     json.RawMessage(`{}`),
		EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
	}

	// Call GetExecutionConfigsBatch - should fail
	configs, err := blockchainService.GetExecutionConfigsBatch(ctx, []*domain.Job{job})

	// Verify it returns an error
	if err == nil {
		t.Fatal("Expected error for unsupported chain ID, got nil")
	}

	if configs != nil {
		t.Fatal("Expected nil configs for unsupported chain ID")
	}

	// Verify error message contains chain info
	if !strings.Contains(err.Error(), "chain 1") {
		t.Errorf("Expected error to mention chain 1, got: %s", err.Error())
	}
}

func TestGetExecutionConfigsBatch_MixedValidInvalidChains(t *testing.T) {
	ctx := context.Background()
	blockchainService := getBlockchainService()

	// Create jobs with mixed valid and invalid chains
	jobs := []*domain.Job{
		{
			ID:                uuid.New(),
			AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
			ChainID:           11155111, // Sepolia testnet - valid
			OnChainJobID:      1,
			UserOperation:     json.RawMessage(`{"sender":"0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1","nonce":"0x1","callData":"0x","callGasLimit":"100000","verificationGasLimit":"50000","preVerificationGas":"21000","maxPriorityFeePerGas":"1000000000","maxFeePerGas":"2000000000","signature":"0x"}`),
			EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
		},
		{
			ID:                uuid.New(),
			AccountAddress:    common.HexToAddress("0x47d6a8a65cba9b61b194dac740aa192a7a1e91e1"),
			ChainID:           1, // Mainnet - invalid
			OnChainJobID:      1,
			UserOperation:     json.RawMessage(`{}`),
			EntryPointAddress: common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032"),
		},
	}

	// Call GetExecutionConfigsBatch - should fail due to invalid chain
	configs, err := blockchainService.GetExecutionConfigsBatch(ctx, jobs)

	// Verify it returns an error
	if err == nil {
		t.Fatal("Expected error for mixed valid/invalid chains, got nil")
	}

	if configs != nil {
		t.Fatal("Expected nil configs for mixed valid/invalid chains")
	}

	// Verify error message mentions the invalid chain
	if !strings.Contains(err.Error(), "chain 1") {
		t.Errorf("Expected error to mention invalid chain 1, got: %s", err.Error())
	}
}
