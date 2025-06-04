package service

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const (
	EntryPointV07 = "0x0000000071727De22E5E9d8BAf0edAc6f37da032"
	EntryPointV08 = "0x4337084D9E255Ff0702461CF8895CE9E3b5Ff108"
)

// PackedUserOp represents the packed user operation format used in ERC-4337
type PackedUserOp struct {
	Sender             common.Address
	Nonce              *big.Int
	InitCode           []byte
	CallData           []byte
	AccountGasLimits   [32]byte
	PreVerificationGas *big.Int
	GasFees            [32]byte
	PaymasterAndData   []byte
	Signature          []byte
}

// convertToPackedUserOp converts domain.UserOperation to PackedUserOp
func convertToPackedUserOp(userOp *domain.UserOperation) (*PackedUserOp, error) {
	// Parse sender address
	sender := common.HexToAddress(userOp.Sender)

	// Parse nonce
	nonce, err := parseHexToBigInt(userOp.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nonce: %w", err)
	}

	// Parse initCode (factory + factoryData) - requires BOTH factory AND factoryData
	var initCode []byte
	if userOp.Factory != "" && userOp.Factory != "0x" &&
		userOp.FactoryData != "" && userOp.FactoryData != "0x" {
		factoryAddr := common.HexToAddress(userOp.Factory)
		initCode = append(initCode, factoryAddr.Bytes()...)
		factoryData := common.FromHex(userOp.FactoryData)
		initCode = append(initCode, factoryData...)
	}

	// Parse callData
	callData := common.FromHex(userOp.CallData)

	// Parse gas limits and pack into accountGasLimits
	callGasLimit, err := parseHexToBigInt(userOp.CallGasLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to parse callGasLimit: %w", err)
	}
	verificationGasLimit, err := parseHexToBigInt(userOp.VerificationGasLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to parse verificationGasLimit: %w", err)
	}

	// Pack gas limits: verificationGasLimit (16 bytes) + callGasLimit (16 bytes)
	var accountGasLimits [32]byte
	verificationGasLimitBytes := make([]byte, 16)
	callGasLimitBytes := make([]byte, 16)
	verificationGasLimit.FillBytes(verificationGasLimitBytes)
	callGasLimit.FillBytes(callGasLimitBytes)
	copy(accountGasLimits[:16], verificationGasLimitBytes)
	copy(accountGasLimits[16:], callGasLimitBytes)

	// Parse preVerificationGas
	preVerificationGas, err := parseHexToBigInt(userOp.PreVerificationGas)
	if err != nil {
		return nil, fmt.Errorf("failed to parse preVerificationGas: %w", err)
	}

	// Parse gas fees and pack into gasFees
	maxPriorityFeePerGas, err := parseHexToBigInt(userOp.MaxPriorityFeePerGas)
	if err != nil {
		return nil, fmt.Errorf("failed to parse maxPriorityFeePerGas: %w", err)
	}
	maxFeePerGas, err := parseHexToBigInt(userOp.MaxFeePerGas)
	if err != nil {
		return nil, fmt.Errorf("failed to parse maxFeePerGas: %w", err)
	}

	// Pack gas fees: maxPriorityFeePerGas (16 bytes) + maxFeePerGas (16 bytes)
	var gasFees [32]byte
	maxPriorityFeePerGasBytes := make([]byte, 16)
	maxFeePerGasBytes := make([]byte, 16)
	maxPriorityFeePerGas.FillBytes(maxPriorityFeePerGasBytes)
	maxFeePerGas.FillBytes(maxFeePerGasBytes)
	copy(gasFees[:16], maxPriorityFeePerGasBytes)
	copy(gasFees[16:], maxFeePerGasBytes)

	// Parse paymasterAndData - requires BOTH paymaster AND paymasterData
	var paymasterAndData []byte
	var hasPaymasterData bool

	// Check if paymasterData exists and is not empty
	if userOp.PaymasterData != nil {
		if dataStr, ok := userOp.PaymasterData.(string); ok && dataStr != "" && dataStr != "0x" {
			hasPaymasterData = true
		}
	}

	if userOp.Paymaster != "" && userOp.Paymaster != "0x" && hasPaymasterData {
		paymasterAddr := common.HexToAddress(userOp.Paymaster)
		paymasterAndData = append(paymasterAndData, paymasterAddr.Bytes()...)

		// Add paymaster verification gas limit (16 bytes)
		if userOp.PaymasterVerificationGasLimit != "" {
			paymasterVerificationGasLimit, err := parseHexToBigInt(userOp.PaymasterVerificationGasLimit)
			if err != nil {
				return nil, fmt.Errorf("failed to parse paymasterVerificationGasLimit: %w", err)
			}
			paymasterVerificationGasLimitBytes := make([]byte, 16)
			paymasterVerificationGasLimit.FillBytes(paymasterVerificationGasLimitBytes)
			paymasterAndData = append(paymasterAndData, paymasterVerificationGasLimitBytes...)
		} else {
			paymasterAndData = append(paymasterAndData, make([]byte, 16)...)
		}

		// Add paymaster post op gas limit (16 bytes)
		if userOp.PaymasterPostOpGasLimit != "" {
			paymasterPostOpGasLimit, err := parseHexToBigInt(userOp.PaymasterPostOpGasLimit)
			if err != nil {
				return nil, fmt.Errorf("failed to parse paymasterPostOpGasLimit: %w", err)
			}
			paymasterPostOpGasLimitBytes := make([]byte, 16)
			paymasterPostOpGasLimit.FillBytes(paymasterPostOpGasLimitBytes)
			paymasterAndData = append(paymasterAndData, paymasterPostOpGasLimitBytes...)
		} else {
			paymasterAndData = append(paymasterAndData, make([]byte, 16)...)
		}

		// Add paymaster data
		if dataStr, ok := userOp.PaymasterData.(string); ok && dataStr != "" && dataStr != "0x" {
			paymasterData := common.FromHex(dataStr)
			paymasterAndData = append(paymasterAndData, paymasterData...)
		}
	}

	// Parse signature
	signature := common.FromHex(userOp.Signature)

	return &PackedUserOp{
		Sender:             sender,
		Nonce:              nonce,
		InitCode:           initCode,
		CallData:           callData,
		AccountGasLimits:   accountGasLimits,
		PreVerificationGas: preVerificationGas,
		GasFees:            gasFees,
		PaymasterAndData:   paymasterAndData,
		Signature:          signature,
	}, nil
}

// parseHexToBigInt parses a hex string to big.Int, handling both "0x" prefixed and non-prefixed strings
func parseHexToBigInt(hexStr string) (*big.Int, error) {
	if hexStr == "" || hexStr == "0x" {
		return big.NewInt(0), nil
	}

	// Remove 0x prefix if present
	cleanStr := hexStr
	if strings.HasPrefix(hexStr, "0x") {
		cleanStr = hexStr[2:]
	}

	// Try parsing as hex first
	result := new(big.Int)
	result, ok := result.SetString(cleanStr, 16)
	if ok {
		return result, nil
	}

	// If hex parsing fails and there's no 0x prefix, try decimal
	if !strings.HasPrefix(hexStr, "0x") {
		if val, err := strconv.ParseInt(cleanStr, 10, 64); err == nil {
			return big.NewInt(val), nil
		}
	}

	return nil, fmt.Errorf("invalid hex string: %s", hexStr)
}

// GetUserOpHashV07 implements the v0.7 user operation hashing
func GetUserOpHashV07(userOp *domain.UserOperation, chainId int64) ([]byte, error) {
	packedOp, err := convertToPackedUserOp(userOp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to packed user op: %w", err)
	}

	// Hash individual components
	hashedInitCode := crypto.Keccak256(packedOp.InitCode)
	hashedCallData := crypto.Keccak256(packedOp.CallData)
	hashedPaymasterAndData := crypto.Keccak256(packedOp.PaymasterAndData)

	// Create ABI encoder
	uint256Type, _ := abi.NewType("uint256", "", nil)
	addressType, _ := abi.NewType("address", "", nil)
	bytes32Type, _ := abi.NewType("bytes32", "", nil)

	// First encoding: pack the user operation fields
	arguments := abi.Arguments{
		{Type: addressType}, // sender
		{Type: uint256Type}, // nonce
		{Type: bytes32Type}, // hashedInitCode
		{Type: bytes32Type}, // hashedCallData
		{Type: bytes32Type}, // accountGasLimits
		{Type: uint256Type}, // preVerificationGas
		{Type: bytes32Type}, // gasFees
		{Type: bytes32Type}, // hashedPaymasterAndData
	}

	// Convert byte arrays to [32]byte for ABI packing
	var hashedInitCodeBytes32 [32]byte
	var hashedCallDataBytes32 [32]byte
	var hashedPaymasterAndDataBytes32 [32]byte
	copy(hashedInitCodeBytes32[:], hashedInitCode)
	copy(hashedCallDataBytes32[:], hashedCallData)
	copy(hashedPaymasterAndDataBytes32[:], hashedPaymasterAndData)

	packed, err := arguments.Pack(
		packedOp.Sender,
		packedOp.Nonce,
		hashedInitCodeBytes32,
		hashedCallDataBytes32,
		packedOp.AccountGasLimits,
		packedOp.PreVerificationGas,
		packedOp.GasFees,
		hashedPaymasterAndDataBytes32,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack user operation: %w", err)
	}

	// Hash the packed data
	userOpHash := crypto.Keccak256(packed)

	// Second encoding: pack with entry point and chain ID
	finalArguments := abi.Arguments{
		{Type: bytes32Type}, // userOpHash
		{Type: addressType}, // entryPoint
		{Type: uint256Type}, // chainId
	}

	entryPointAddr := common.HexToAddress(EntryPointV07)
	chainIdBig := big.NewInt(chainId)

	// Convert userOpHash to [32]byte
	var userOpHashBytes32 [32]byte
	copy(userOpHashBytes32[:], userOpHash)

	finalPacked, err := finalArguments.Pack(userOpHashBytes32, entryPointAddr, chainIdBig)
	if err != nil {
		return nil, fmt.Errorf("failed to pack final hash: %w", err)
	}

	return crypto.Keccak256(finalPacked), nil
}

// GetUserOpHashV08 implements the v0.8 user operation hashing using EIP-712
func GetUserOpHashV08(userOp *domain.UserOperation, chainId int64) ([]byte, error) {
	packedOp, err := convertToPackedUserOp(userOp)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to packed user op: %w", err)
	}

	// Create EIP-712 domain
	domain := apitypes.TypedDataDomain{
		Name:              "ERC4337",
		Version:           "1",
		ChainId:           (*math.HexOrDecimal256)(big.NewInt(chainId)),
		VerifyingContract: EntryPointV08,
	}

	// Define the PackedUserOperation type
	types := apitypes.Types{
		"EIP712Domain": {
			{Name: "name", Type: "string"},
			{Name: "version", Type: "string"},
			{Name: "chainId", Type: "uint256"},
			{Name: "verifyingContract", Type: "address"},
		},
		"PackedUserOperation": {
			{Name: "sender", Type: "address"},
			{Name: "nonce", Type: "uint256"},
			{Name: "initCode", Type: "bytes"},
			{Name: "callData", Type: "bytes"},
			{Name: "accountGasLimits", Type: "bytes32"},
			{Name: "preVerificationGas", Type: "uint256"},
			{Name: "gasFees", Type: "bytes32"},
			{Name: "paymasterAndData", Type: "bytes"},
		},
	}

	// Create the message data - use string representations for EIP-712
	message := map[string]interface{}{
		"sender":             packedOp.Sender.Hex(),
		"nonce":              packedOp.Nonce.String(),
		"initCode":           hexutil.Encode(packedOp.InitCode),
		"callData":           hexutil.Encode(packedOp.CallData),
		"accountGasLimits":   hexutil.Encode(packedOp.AccountGasLimits[:]),
		"preVerificationGas": packedOp.PreVerificationGas.String(),
		"gasFees":            hexutil.Encode(packedOp.GasFees[:]),
		"paymasterAndData":   hexutil.Encode(packedOp.PaymasterAndData),
	}

	// Create typed data
	typedData := apitypes.TypedData{
		Types:       types,
		PrimaryType: "PackedUserOperation",
		Domain:      domain,
		Message:     message,
	}

	// Hash the typed data
	hash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to hash struct: %w", err)
	}

	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("failed to hash domain: %w", err)
	}

	// Create final EIP-712 hash
	rawData := []byte{0x19, 0x01}
	rawData = append(rawData, domainSeparator...)
	rawData = append(rawData, hash...)

	return crypto.Keccak256(rawData), nil
}

// GetUserOpHash determines the entry point version and calls the appropriate hashing function
func GetUserOpHash(userOp *domain.UserOperation, entryPoint string, chainId int64) ([]byte, error) {
	entryPointLower := strings.ToLower(entryPoint)

	switch entryPointLower {
	case strings.ToLower(EntryPointV07):
		return GetUserOpHashV07(userOp, chainId)
	case strings.ToLower(EntryPointV08):
		return GetUserOpHashV08(userOp, chainId)
	default:
		return nil, fmt.Errorf("unsupported entry point: %s", entryPoint)
	}
}
