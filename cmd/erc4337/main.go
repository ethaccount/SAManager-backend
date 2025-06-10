package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/joho/godotenv"
)

const userOpJSON = `{
	"sender": "0x47D6a8A65cBa9b61B194daC740AA192A7A1e91e1",
	"nonce": "0x0100000000002b0ecfbd0496ee71e01257da0e37de00000000000000000000",
	"factory": null,
	"factoryData": "0x",
	"callData": "0xe9ae5c53000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000058a8e374779aee60413c974b484d6509c7e4ddb6ba000000000000000000000000000000000000000000000000000000000000000094f6113400000000000000000000000000000000000000000000000000000000000000090000000000000000",
	"callGasLimit": "0x00",
	"verificationGasLimit": "0x00",
	"preVerificationGas": "0x00",
	"maxFeePerGas": "0x00",
	"maxPriorityFeePerGas": "0x00",
	"paymaster": null,
	"paymasterVerificationGasLimit": "0x00",
	"paymasterPostOpGasLimit": "0x00",
	"paymasterData": "0x",
	"signature": "0x00ba06d407c8d9ddaaac3b680421283c1c424cd21e8205173dfef1840705aa9957"
}`

// Block represents a block header with baseFeePerGas
type Block struct {
	BaseFeePerGas string `json:"baseFeePerGas"`
}

// extractNonceKey extracts the nonce key by removing the trailing 8 bytes (64 bits)
func extractNonceKey(nonce *hexutil.Big) (*big.Int, error) {
	if nonce == nil {
		return nil, fmt.Errorf("nonce is nil")
	}

	nonceBytes := nonce.ToInt().Bytes()
	if len(nonceBytes) < 8 {
		return nil, fmt.Errorf("nonce too short to extract key")
	}

	// Pad to 32 bytes if needed
	paddedNonce := make([]byte, 32)
	copy(paddedNonce[32-len(nonceBytes):], nonceBytes)

	// Extract key by taking first 24 bytes (removing trailing 8 bytes)
	keyBytes := paddedNonce[:24]
	key := new(big.Int).SetBytes(keyBytes)

	return key, nil
}

// getCurrentNonce calls the entrypoint contract's getNonce function
func getCurrentNonce(ctx context.Context, rpcClient *rpc.Client, sender common.Address, key *big.Int) (*big.Int, error) {
	// Prepare the call data for getNonce(address,uint192)
	// Function selector: getNonce(address,uint192) = 0x35567e1a
	callData := "0x35567e1a"

	// Encode sender address (32 bytes)
	senderBytes := make([]byte, 32)
	copy(senderBytes[12:], sender.Bytes())
	callData += fmt.Sprintf("%x", senderBytes)

	// Encode key (32 bytes, but only 24 bytes are significant for uint192)
	keyBytes := make([]byte, 32)
	key.FillBytes(keyBytes)
	callData += fmt.Sprintf("%x", keyBytes)

	// Make the eth_call
	var result string
	err := rpcClient.CallContext(ctx, &result, "eth_call", map[string]interface{}{
		"to":   erc4337.EntryPointV07,
		"data": callData,
	}, "latest")

	if err != nil {
		return nil, fmt.Errorf("failed to call getNonce: %w", err)
	}

	// Parse the result
	nonce := new(big.Int)
	if err := nonce.UnmarshalText([]byte(result)); err != nil {
		return nil, fmt.Errorf("failed to parse nonce result: %w", err)
	}

	return nonce, nil
}

// getMaxFeePerGas fetches the latest block and max priority fee, then calculates maxFeePerGas
func getMaxFeePerGas(ctx context.Context, rpcClient *rpc.Client) (*big.Int, *big.Int, error) {
	var blockResult *Block
	var maxPriorityFeeResult string

	batch := []rpc.BatchElem{
		{
			Method: "eth_getBlockByNumber",
			Args:   []interface{}{"latest", false},
			Result: &blockResult,
		},
		{
			Method: "rundler_maxPriorityFeePerGas",
			Args:   []interface{}{},
			Result: &maxPriorityFeeResult,
		},
	}

	if err := rpcClient.BatchCallContext(ctx, batch); err != nil {
		return nil, nil, fmt.Errorf("failed to make batch RPC calls: %w", err)
	}

	// Check for individual call errors
	if batch[0].Error != nil {
		return nil, nil, fmt.Errorf("eth_getBlockByNumber failed: %w", batch[0].Error)
	}
	if batch[1].Error != nil {
		return nil, nil, fmt.Errorf("rundler_maxPriorityFeePerGas failed: %w", batch[1].Error)
	}

	// Parse baseFeePerGas
	baseFeePerGas := new(big.Int)
	if err := baseFeePerGas.UnmarshalText([]byte(blockResult.BaseFeePerGas)); err != nil {
		return nil, nil, fmt.Errorf("failed to parse baseFeePerGas: %w", err)
	}

	// Parse maxPriorityFeePerGas
	maxPriorityFeePerGas := new(big.Int)
	if err := maxPriorityFeePerGas.UnmarshalText([]byte(maxPriorityFeeResult)); err != nil {
		return nil, nil, fmt.Errorf("failed to parse maxPriorityFeePerGas: %w", err)
	}

	// Calculate maxFeePerGas: (baseFeePerGas * 150 / 100) + maxPriorityFeePerGas
	maxFeePerGas := new(big.Int)
	maxFeePerGas.Mul(baseFeePerGas, big.NewInt(150))
	maxFeePerGas.Div(maxFeePerGas, big.NewInt(100))
	maxFeePerGas.Add(maxFeePerGas, maxPriorityFeePerGas)

	return maxFeePerGas, maxPriorityFeePerGas, nil
}

// personalSignHash creates an Ethereum signed message hash
func personalSignHash(data []byte) common.Hash {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256Hash([]byte(msg))
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	rpcUrl := os.Getenv("SEPOLIA_RPC_URL")
	if rpcUrl == "" {
		log.Fatalf("SEPOLIA_RPC_URL not set in .env file")
	}

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		log.Fatalf("PRIVATE_KEY not set in .env file")
	}

	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	// Parse private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	// Display signing address
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	log.Printf("Signing with address: %s", address.Hex())

	// Parse the user operation from JSON
	var userOp erc4337.UserOperation
	if err := json.Unmarshal([]byte(userOpJSON), &userOp); err != nil {
		log.Fatalf("Failed to parse user operation: %v", err)
	}

	ctx := context.Background()

	// Connect to bundler
	c, err := erc4337.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to bundler: %v", err)
	}

	// Create RPC client for direct blockchain calls
	rpcClient, err := rpc.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatalf("Failed to create RPC client: %v", err)
	}
	defer rpcClient.Close()

	// Extract nonce key and get current nonce from entrypoint
	nonceKey, err := extractNonceKey(userOp.Nonce)
	if err != nil {
		log.Fatalf("Failed to extract nonce key: %v", err)
	}

	log.Printf("Extracted nonce key: 0x%x", nonceKey)

	currentNonce, err := getCurrentNonce(ctx, rpcClient, userOp.Sender, nonceKey)
	if err != nil {
		log.Fatalf("Failed to get current nonce: %v", err)
	}

	log.Printf("Current nonce from entrypoint: 0x%x", currentNonce)

	// Update user operation with current nonce
	userOp.Nonce = (*hexutil.Big)(currentNonce)

	// Set paymaster
	paymaster := common.HexToAddress("0xcD1c62f36A99f306948dB76c35Bbc1A639f92ce8")
	userOp.Paymaster = &paymaster

	// Add dummy signature for gas estimation
	dummySignature := "0xfffffffffffffffffffffffffffffff0000000000000000000000000000000007aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1c"
	decodedDummySignature, err := hexutil.Decode(dummySignature)
	if err != nil {
		log.Fatalf("Failed to decode dummy signature: %v", err)
	}

	leadingSignature := userOp.Signature
	userOp.Signature = append(userOp.Signature, decodedDummySignature...)

	// Estimate gas values
	estimates, err := c.EstimateUserOperationGas(ctx, &userOp, erc4337.EntryPointV07)
	if err != nil {
		log.Fatalf("Failed to estimate user operation gas: %v", err)
	}

	// Get gas fees
	maxFeePerGas, maxPriorityFeePerGas, err := getMaxFeePerGas(ctx, rpcClient)
	if err != nil {
		log.Fatalf("Failed to get gas fees: %v", err)
	}

	log.Printf("Gas fees - MaxFeePerGas: %s, MaxPriorityFeePerGas: %s", maxFeePerGas.String(), maxPriorityFeePerGas.String())
	log.Printf("Gas estimates - PreVerificationGas: %s, VerificationGasLimit: %s, CallGasLimit: %s",
		estimates.PreVerificationGas.String(), estimates.VerificationGasLimit.String(), estimates.CallGasLimit.String())

	// Update user operation with gas values
	userOp.PreVerificationGas = (*hexutil.Big)(estimates.PreVerificationGas)
	userOp.VerificationGasLimit = (*hexutil.Big)(estimates.VerificationGasLimit)
	userOp.CallGasLimit = (*hexutil.Big)(estimates.CallGasLimit)
	userOp.PaymasterVerificationGasLimit = (*hexutil.Big)(estimates.PaymasterVerificationGasLimit)
	userOp.MaxFeePerGas = (*hexutil.Big)(maxFeePerGas)
	userOp.MaxPriorityFeePerGas = (*hexutil.Big)(maxPriorityFeePerGas)

	// Calculate user operation hash
	hash, err := userOp.GetUserOpHashV07(big.NewInt(11155111))
	if err != nil {
		log.Fatalf("Failed to calculate user operation hash: %v", err)
	}
	log.Printf("User Operation Hash: %s", hash.Hex())

	// Sign the user operation hash
	signature, err := crypto.Sign(personalSignHash(hash.Bytes()).Bytes(), privateKey)
	if err != nil {
		log.Fatalf("Failed to sign user operation hash: %v", err)
	}

	// Adjust signature format for Ethereum (recovery ID + 27)
	signature[64] += 27
	log.Printf("Generated signature: 0x%s", hex.EncodeToString(signature))

	// Update the signature in the user operation
	userOp.Signature = append(leadingSignature, signature...)

	// Send the user operation
	userOpHash, err := c.SendUserOperation(ctx, &userOp, erc4337.EntryPointV07)
	if err != nil {
		log.Fatalf("Failed to send user operation: %v", err)
	}

	log.Printf("User Operation sent successfully! Hash: %s", userOpHash.Hex())

	// Poll for user operation receipt
	log.Printf("Polling for user operation receipt...")
	maxAttempts := 60
	pollInterval := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("Attempt %d/%d: Checking for receipt...", attempt, maxAttempts)

		receipt, err := c.GetUserOperationReceipt(ctx, userOpHash)
		if err != nil {
			log.Printf("Receipt not yet available (attempt %d): %v", attempt, err)
		} else {
			log.Printf("User Operation Receipt received!")
			log.Printf("  UserOpHash: %s", receipt.UserOpHash.Hex())
			log.Printf("  Sender: %s", receipt.Sender.Hex())
			log.Printf("  Success: %t", receipt.Success)
			log.Printf("  ActualGasCost: %s", receipt.ActualGasCost)
			log.Printf("  ActualGasUsed: %s", receipt.ActualGasUsed)
			if receipt.Paymaster != (common.Address{}) {
				log.Printf("  Paymaster: %s", receipt.Paymaster.Hex())
			}
			log.Printf("  Nonce: %s", receipt.Nonce)
			if receipt.Receipt != nil {
				log.Printf("  Transaction Hash: %s", receipt.Receipt.TransactionHash.Hex())
				log.Printf("  Block Number: %s", receipt.Receipt.BlockNumber)
				log.Printf("  Gas Used: %s", receipt.Receipt.GasUsed)
			}
			return
		}

		if attempt < maxAttempts {
			time.Sleep(pollInterval)
		}
	}

	log.Printf("Failed to get user operation receipt after %d attempts", maxAttempts)
}
