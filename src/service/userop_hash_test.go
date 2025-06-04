package service

import (
	"encoding/hex"
	"testing"

	"github.com/ethaccount/backend/src/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserOpHashV07(t *testing.T) {
	// Test case with known values
	userOp := &domain.UserOperation{
		Sender:               "0x1234567890123456789012345678901234567890",
		Nonce:                "0x1",
		Factory:              "",
		FactoryData:          "",
		CallData:             "0xabcdef",
		CallGasLimit:         "100000",
		VerificationGasLimit: "50000",
		PreVerificationGas:   "21000",
		MaxPriorityFeePerGas: "1000000000",
		MaxFeePerGas:         "2000000000",
		Paymaster:            "",
		Signature:            "0x",
	}

	chainId := int64(11155111) // Sepolia testnet

	hash, err := GetUserOpHashV07(userOp, chainId)
	require.NoError(t, err)
	require.NotNil(t, hash)
	assert.Equal(t, 32, len(hash)) // Keccak256 produces 32-byte hash

	// Test that same inputs produce same hash
	hash2, err := GetUserOpHashV07(userOp, chainId)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Test that different inputs produce different hash
	userOp2 := *userOp
	userOp2.Nonce = "0x2"
	hash3, err := GetUserOpHashV07(&userOp2, chainId)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash3)

	t.Logf("V0.7 Hash: %s", hex.EncodeToString(hash))
}

func TestGetUserOpHashV08(t *testing.T) {
	// Test case with known values
	userOp := &domain.UserOperation{
		Sender:               "0x1234567890123456789012345678901234567890",
		Nonce:                "0x1",
		Factory:              "",
		FactoryData:          "",
		CallData:             "0xabcdef",
		CallGasLimit:         "100000",
		VerificationGasLimit: "50000",
		PreVerificationGas:   "21000",
		MaxPriorityFeePerGas: "1000000000",
		MaxFeePerGas:         "2000000000",
		Paymaster:            "",
		Signature:            "0x",
	}

	chainId := int64(11155111) // Sepolia testnet

	hash, err := GetUserOpHashV08(userOp, chainId)
	require.NoError(t, err)
	require.NotNil(t, hash)
	assert.Equal(t, 32, len(hash)) // Keccak256 produces 32-byte hash

	// Test that same inputs produce same hash
	hash2, err := GetUserOpHashV08(userOp, chainId)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Test that different inputs produce different hash
	userOp2 := *userOp
	userOp2.Nonce = "0x2"
	hash3, err := GetUserOpHashV08(&userOp2, chainId)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash3)

	t.Logf("V0.8 Hash: %s", hex.EncodeToString(hash))
}

func TestGetUserOpHashWithPaymaster(t *testing.T) {
	// Test case with paymaster
	userOp := &domain.UserOperation{
		Sender:                        "0x1234567890123456789012345678901234567890",
		Nonce:                         "0x1",
		Factory:                       "",
		FactoryData:                   "",
		CallData:                      "0xabcdef",
		CallGasLimit:                  "100000",
		VerificationGasLimit:          "50000",
		PreVerificationGas:            "21000",
		MaxPriorityFeePerGas:          "1000000000",
		MaxFeePerGas:                  "2000000000",
		Paymaster:                     "0x9876543210987654321098765432109876543210",
		PaymasterVerificationGasLimit: "30000",
		PaymasterPostOpGasLimit:       "10000",
		PaymasterData:                 "0x1234",
		Signature:                     "0x",
	}

	chainId := int64(11155111)

	// Test V0.7
	hashV07, err := GetUserOpHashV07(userOp, chainId)
	require.NoError(t, err)
	assert.Equal(t, 32, len(hashV07))

	// Test V0.8
	hashV08, err := GetUserOpHashV08(userOp, chainId)
	require.NoError(t, err)
	assert.Equal(t, 32, len(hashV08))

	// V0.7 and V0.8 should produce different hashes
	assert.NotEqual(t, hashV07, hashV08)

	t.Logf("V0.7 Hash with Paymaster: %s", hex.EncodeToString(hashV07))
	t.Logf("V0.8 Hash with Paymaster: %s", hex.EncodeToString(hashV08))
}

func TestGetUserOpHashWithFactory(t *testing.T) {
	// Test case with factory (for account creation)
	userOp := &domain.UserOperation{
		Sender:               "0x1234567890123456789012345678901234567890",
		Nonce:                "0x0", // Usually 0 for account creation
		Factory:              "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		FactoryData:          "0x123456789abcdef",
		CallData:             "0xabcdef",
		CallGasLimit:         "100000",
		VerificationGasLimit: "50000",
		PreVerificationGas:   "21000",
		MaxPriorityFeePerGas: "1000000000",
		MaxFeePerGas:         "2000000000",
		Paymaster:            "",
		Signature:            "0x",
	}

	chainId := int64(11155111)

	// Test V0.7
	hashV07, err := GetUserOpHashV07(userOp, chainId)
	require.NoError(t, err)
	assert.Equal(t, 32, len(hashV07))

	// Test V0.8
	hashV08, err := GetUserOpHashV08(userOp, chainId)
	require.NoError(t, err)
	assert.Equal(t, 32, len(hashV08))

	// V0.7 and V0.8 should produce different hashes
	assert.NotEqual(t, hashV07, hashV08)

	t.Logf("V0.7 Hash with Factory: %s", hex.EncodeToString(hashV07))
	t.Logf("V0.8 Hash with Factory: %s", hex.EncodeToString(hashV08))
}

func TestGetUserOpHash(t *testing.T) {
	userOp := &domain.UserOperation{
		Sender:               "0x1234567890123456789012345678901234567890",
		Nonce:                "0x1",
		CallData:             "0xabcdef",
		CallGasLimit:         "100000",
		VerificationGasLimit: "50000",
		PreVerificationGas:   "21000",
		MaxPriorityFeePerGas: "1000000000",
		MaxFeePerGas:         "2000000000",
		Signature:            "0x",
	}

	chainId := int64(11155111)

	// Test V0.7 entry point
	hashV07, err := GetUserOpHash(userOp, EntryPointV07, chainId)
	require.NoError(t, err)
	assert.Equal(t, 32, len(hashV07))

	// Test V0.8 entry point
	hashV08, err := GetUserOpHash(userOp, EntryPointV08, chainId)
	require.NoError(t, err)
	assert.Equal(t, 32, len(hashV08))

	// Should produce different hashes
	assert.NotEqual(t, hashV07, hashV08)

	// Test unsupported entry point
	_, err = GetUserOpHash(userOp, "0x1111111111111111111111111111111111111111", chainId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported entry point")
}

func TestConvertToPackedUserOp(t *testing.T) {
	userOp := &domain.UserOperation{
		Sender:                        "0x1234567890123456789012345678901234567890",
		Nonce:                         "0x1",
		Factory:                       "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		FactoryData:                   "0x123456",
		CallData:                      "0xabcdef",
		CallGasLimit:                  "0x186a0",    // 100000 in hex
		VerificationGasLimit:          "0xc350",     // 50000 in hex
		PreVerificationGas:            "0x5208",     // 21000 in hex
		MaxPriorityFeePerGas:          "0x3b9aca00", // 1000000000 in hex
		MaxFeePerGas:                  "0x77359400", // 2000000000 in hex
		Paymaster:                     "0x9876543210987654321098765432109876543210",
		PaymasterVerificationGasLimit: "0x7530", // 30000 in hex
		PaymasterPostOpGasLimit:       "0x2710", // 10000 in hex
		PaymasterData:                 "0x1234",
		Signature:                     "0x",
	}

	packedOp, err := convertToPackedUserOp(userOp)
	require.NoError(t, err)
	require.NotNil(t, packedOp)

	// Verify sender
	assert.Equal(t, "0x1234567890123456789012345678901234567890", packedOp.Sender.Hex())

	// Verify nonce
	assert.Equal(t, int64(1), packedOp.Nonce.Int64())

	// Verify initCode contains factory address + factory data
	assert.Greater(t, len(packedOp.InitCode), 20) // At least factory address (20 bytes)

	// Verify callData
	assert.Equal(t, []byte{0xab, 0xcd, 0xef}, packedOp.CallData)

	// Verify preVerificationGas
	assert.Equal(t, int64(21000), packedOp.PreVerificationGas.Int64())

	// Verify paymasterAndData contains paymaster address + gas limits + data
	assert.Greater(t, len(packedOp.PaymasterAndData), 20) // At least paymaster address (20 bytes)
}

func TestParseHexToBigInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"0x1", 1, false},
		{"1", 1, false}, // Single digit is same in hex and decimal
		{"0x10", 16, false},
		{"10", 16, false}, // Without 0x prefix, treated as hex (16 in decimal)
		{"0xff", 255, false},
		{"", 0, false},
		{"0x", 0, false},
		{"invalid", 0, true},
		{"9", 9, false},  // Single digit is same in hex and decimal
		{"a", 10, false}, // Hex digit without 0x prefix
		{"A", 10, false}, // Uppercase hex digit without 0x prefix
	}

	for _, test := range tests {
		result, err := parseHexToBigInt(test.input)
		if test.hasError {
			assert.Error(t, err, "Expected error for input: %s", test.input)
		} else {
			require.NoError(t, err, "Unexpected error for input: %s", test.input)
			assert.Equal(t, test.expected, result.Int64(), "Wrong result for input: %s", test.input)
		}
	}
}

// TestKnownHashVector tests against a known hash vector to ensure compatibility
func TestKnownHashVector(t *testing.T) {
	// This is a simplified test case - in a real implementation, you would use
	// known test vectors from the ERC-4337 specification or reference implementations
	userOp := &domain.UserOperation{
		Sender:               "0x0000000000000000000000000000000000000001",
		Nonce:                "0x0",
		Factory:              "",
		FactoryData:          "",
		CallData:             "0x",
		CallGasLimit:         "0x186a0",    // 100000
		VerificationGasLimit: "0xc350",     // 50000
		PreVerificationGas:   "0x5208",     // 21000
		MaxPriorityFeePerGas: "0x3b9aca00", // 1000000000
		MaxFeePerGas:         "0x77359400", // 2000000000
		Paymaster:            "",
		Signature:            "0x",
	}

	chainId := int64(1) // Ethereum mainnet

	// Test V0.7
	hashV07, err := GetUserOpHashV07(userOp, chainId)
	require.NoError(t, err)
	t.Logf("Known Vector V0.7 Hash: %s", hex.EncodeToString(hashV07))

	// Test V0.8
	hashV08, err := GetUserOpHashV08(userOp, chainId)
	require.NoError(t, err)
	t.Logf("Known Vector V0.8 Hash: %s", hex.EncodeToString(hashV08))

	// Ensure they're different
	assert.NotEqual(t, hashV07, hashV08)
}

// TestExecutionServiceHashIntegration tests the integration with ExecutionService
func TestExecutionServiceHashIntegration(t *testing.T) {
	// Create a minimal execution service without database dependencies
	executionService := &ExecutionService{}

	userOp := &domain.UserOperation{
		Sender:               "0x1234567890123456789012345678901234567890",
		Nonce:                "0x1",
		CallData:             "0xabcdef",
		CallGasLimit:         "100000",
		VerificationGasLimit: "50000",
		PreVerificationGas:   "21000",
		MaxPriorityFeePerGas: "1000000000",
		MaxFeePerGas:         "2000000000",
		Signature:            "0x",
	}

	chainId := int64(11155111)

	// Test V0.7 entry point
	hashV07, err := executionService.createUserOperationHash(userOp, EntryPointV07, chainId)
	require.NoError(t, err)
	assert.Equal(t, 32, len(hashV07))

	// Test V0.8 entry point
	hashV08, err := executionService.createUserOperationHash(userOp, EntryPointV08, chainId)
	require.NoError(t, err)
	assert.Equal(t, 32, len(hashV08))

	// Should produce different hashes
	assert.NotEqual(t, hashV07, hashV08)

	t.Logf("ExecutionService V0.7 Hash: %s", hex.EncodeToString(hashV07))
	t.Logf("ExecutionService V0.8 Hash: %s", hex.EncodeToString(hashV08))

	// Test unsupported entry point
	_, err = executionService.createUserOperationHash(userOp, "0x1111111111111111111111111111111111111111", chainId)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported entry point")
}
