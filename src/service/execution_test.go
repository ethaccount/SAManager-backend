package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TestPrivateKey = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
)

func getTestExecutionService(t *testing.T) *ExecutionService {
	// Create blockchain service
	sepoliaRpcUrl := testutil.GetEnv("SEPOLIA_RPC_URL")
	arbitrumSepoliaRpcUrl := testutil.GetEnv("ARBITRUM_SEPOLIA_RPC_URL")
	baseSepoliaRpcUrl := testutil.GetEnv("BASE_SEPOLIA_RPC_URL")
	optimismSepoliaRpcUrl := testutil.GetEnv("OPTIMISM_SEPOLIA_RPC_URL")
	polygonAmoyRpcUrl := testutil.GetEnv("POLYGON_AMOY_RPC_URL")

	blockchainService := NewBlockchainService(AppConfig{
		SepoliaRPCURL:         &sepoliaRpcUrl,
		ArbitrumSepoliaRPCURL: &arbitrumSepoliaRpcUrl,
		BaseSepoliaRPCURL:     &baseSepoliaRpcUrl,
		OptimismSepoliaRPCURL: &optimismSepoliaRpcUrl,
		PolygonAmoyRPCURL:     &polygonAmoyRpcUrl,
	})

	// Use a test private key (account 2)
	executionService, err := NewExecutionService(blockchainService, TestPrivateKey)
	require.NoError(t, err, "Failed to create execution service")

	return executionService
}

func TestNewExecutionService(t *testing.T) {
	blockchainService := &BlockchainService{}

	// Test with valid private key
	executionService, err := NewExecutionService(blockchainService, TestPrivateKey)

	assert.NoError(t, err)
	assert.NotNil(t, executionService)
	assert.Equal(t, blockchainService, executionService.blockchainService)
	assert.NotNil(t, executionService.privateKey)

	// Test with invalid private key
	invalidPrivateKey := "invalid_key"
	_, err = NewExecutionService(blockchainService, invalidPrivateKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse private key")
}

func TestExecuteJob(t *testing.T) {
	executionService := getTestExecutionService(t)
	ctx := context.Background()

	// Create a test user operation
	userOperation := &domain.UserOperation{
		Sender:               "0x1234567890123456789012345678901234567890",
		Nonce:                "0x1",
		CallData:             "0xabcdef",
		CallGasLimit:         "100000",
		VerificationGasLimit: "50000",
		PreVerificationGas:   "21000",
		MaxPriorityFeePerGas: "1000000000",
		MaxFeePerGas:         "2000000000",
		Signature:            "0x", // Empty signature initially
	}

	// Create a mock job instead of writing to database
	userOpJSON, err := json.Marshal(userOperation)
	require.NoError(t, err)

	job := &domain.Job{
		ID:                uuid.New(),
		AccountAddress:    "0x1234567890123456789012345678901234567890",
		ChainID:           11155111, // Sepolia testnet
		OnChainJobID:      1,
		UserOperation:     userOpJSON,
		EntryPointAddress: "0x0000000071727De22E5E9d8BAf0edAc6f37da032",
	}

	// Note: This test will fail when actually trying to send to bundler
	// since we're using test data and the bundler URLs are placeholders
	// In a real test environment, you would mock the HTTP calls
	_, err = executionService.ExecuteJob(ctx, job)

	// We expect this to fail at the bundler call stage, but the signing should work
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send user operation to bundler")

	// Verify that the user operation was signed (signature was updated)
	updatedUserOp, err := job.GetUserOperation()
	require.NoError(t, err)
	assert.NotEqual(t, "0x", updatedUserOp.Signature)
	assert.NotEmpty(t, updatedUserOp.Signature)
	assert.True(t, len(updatedUserOp.Signature) > 2) // Should have "0x" prefix plus signature data
}

func TestCreateUserOperationHash(t *testing.T) {
	executionService := getTestExecutionService(t)

	userOperation := &domain.UserOperation{
		Sender:               "0x1234567890123456789012345678901234567890",
		Nonce:                "0x1",
		CallData:             "0xabcdef",
		CallGasLimit:         "100000",
		VerificationGasLimit: "50000",
		PreVerificationGas:   "21000",
		MaxPriorityFeePerGas: "1000000000",
		MaxFeePerGas:         "2000000000",
		Paymaster:            "",
	}

	entryPoint := "0x0000000071727De22E5E9d8BAf0edAc6f37da032"
	chainId := int64(11155111)

	hash, err := executionService.createUserOperationHash(userOperation, entryPoint, chainId)

	assert.NoError(t, err)
	assert.NotNil(t, hash)
	assert.Equal(t, 32, len(hash)) // Keccak256 produces 32-byte hash

	// Test that same inputs produce same hash
	hash2, err := executionService.createUserOperationHash(userOperation, entryPoint, chainId)
	assert.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Test that different inputs produce different hash
	userOperation.Nonce = "0x2"
	hash3, err := executionService.createUserOperationHash(userOperation, entryPoint, chainId)
	assert.NoError(t, err)
	assert.NotEqual(t, hash, hash3)
}

func TestExecuteJob_InvalidUserOperation(t *testing.T) {
	executionService := getTestExecutionService(t)
	ctx := context.Background()

	// Create a job with invalid user operation JSON
	job := &domain.Job{
		ID:                uuid.New(),
		AccountAddress:    "0x1234567890123456789012345678901234567890",
		ChainID:           11155111,
		OnChainJobID:      1,
		UserOperation:     json.RawMessage(`{"invalid": "json"`), // Invalid JSON
		EntryPointAddress: "0x0000000071727De22E5E9d8BAf0edAc6f37da032",
	}

	_, err := executionService.ExecuteJob(ctx, job)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get user operation")
}
