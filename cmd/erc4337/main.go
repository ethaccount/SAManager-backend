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
	// Make parallel RPC calls
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

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
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

	// print the public key
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	fmt.Printf("Address: %s\n", address.Hex())

	// Parse the user operation from JSON
	var userOp erc4337.UserOperation
	err = json.Unmarshal([]byte(userOpJSON), &userOp)
	if err != nil {
		log.Fatalf("Failed to parse user operation: %v", err)
	}

	ctx := context.Background()

	c, err := erc4337.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatalf("Failed to connect to bundler: %v", err)
	}

	// Create direct RPC client for gas fee calculations and nonce fetching
	rpcClient, err := rpc.DialContext(ctx, rpcUrl)
	if err != nil {
		log.Fatalf("Failed to create RPC client: %v", err)
	}
	defer rpcClient.Close()

	// Extract nonce key from the original nonce and get current nonce from entrypoint
	nonceKey, err := extractNonceKey(userOp.Nonce)
	if err != nil {
		log.Fatalf("Failed to extract nonce key: %v", err)
	}

	fmt.Printf("Extracted nonce key: 0x%x\n", nonceKey)

	// Get current nonce from entrypoint contract
	currentNonce, err := getCurrentNonce(ctx, rpcClient, userOp.Sender, nonceKey)
	if err != nil {
		log.Fatalf("Failed to get current nonce: %v", err)
	}

	fmt.Printf("Current nonce from entrypoint: 0x%x\n", currentNonce)

	// Update user operation with current nonce
	userOp.Nonce = (*hexutil.Big)(currentNonce)

	// Add paymaster
	paymaster := common.HexToAddress("0xcD1c62f36A99f306948dB76c35Bbc1A639f92ce8")
	userOp.Paymaster = &paymaster

	// Add dummy signature
	var dummySignature = "0xfffffffffffffffffffffffffffffff0000000000000000000000000000000007aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1c"

	decodedDummySignature, err := hexutil.Decode(dummySignature)
	if err != nil {
		log.Fatalf("Failed to decode dummy signature: %v", err)
	}

	leadingSignature := userOp.Signature

	// append dummy signature
	userOp.Signature = append(userOp.Signature, decodedDummySignature...)

	// Estimate gas
	estimates, err := c.EstimateUserOperationGas(ctx, &userOp, erc4337.EntryPointV07)
	if err != nil {
		log.Fatalf("Failed to estimate user operation gas: %v", err)
	}

	// Get max fee per gas and max priority fee per gas
	maxFeePerGas, maxPriorityFeePerGas, err := getMaxFeePerGas(ctx, rpcClient)
	if err != nil {
		log.Fatalf("Failed to get gas fees: %v", err)
	}

	fmt.Printf("Gas Values:\n")
	fmt.Printf("  MaxFeePerGas: %s\n", maxFeePerGas.String())
	fmt.Printf("  MaxPriorityFeePerGas: %s\n", maxPriorityFeePerGas.String())

	fmt.Printf("Gas Estimates:\n")
	fmt.Printf("  PreVerificationGas: %s\n", estimates.PreVerificationGas.String())
	fmt.Printf("  VerificationGasLimit: %s\n", estimates.VerificationGasLimit.String())
	fmt.Printf("  CallGasLimit: %s\n", estimates.CallGasLimit.String())

	if estimates.PaymasterVerificationGasLimit != nil {
		fmt.Printf("  PaymasterVerificationGasLimit: %s\n", estimates.PaymasterVerificationGasLimit.String())
	} else {
		fmt.Printf("  PaymasterVerificationGasLimit: <nil>\n")
	}

	// Add gas values to user operation
	userOp.PreVerificationGas = (*hexutil.Big)(estimates.PreVerificationGas)
	userOp.VerificationGasLimit = (*hexutil.Big)(estimates.VerificationGasLimit)
	userOp.CallGasLimit = (*hexutil.Big)(estimates.CallGasLimit)
	userOp.PaymasterVerificationGasLimit = (*hexutil.Big)(estimates.PaymasterVerificationGasLimit)
	userOp.MaxFeePerGas = (*hexutil.Big)(maxFeePerGas)
	userOp.MaxPriorityFeePerGas = (*hexutil.Big)(maxPriorityFeePerGas)

	// print packed user operation
	packedUserOp := userOp.PackUserOp()
	packedUserOpJSON, err := json.MarshalIndent(packedUserOp, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal packed user operation: %v", err)
	}
	fmt.Printf("Packed User Operation: %s\n", string(packedUserOpJSON))

	// Print the user operation
	userOpJSON, err := json.MarshalIndent(userOp, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal user operation: %v", err)
	}
	fmt.Printf("User Operation: %s\n", string(userOpJSON))

	// userOpHash
	hash, err := userOp.GetUserOpHashV07(big.NewInt(11155111))
	if err != nil {
		log.Fatalf("Failed to calculate user operation hash: %v", err)
	}
	fmt.Printf("User Operation Hash: %s\n", hash.Hex())

	signature, err := crypto.Sign(personalSignHash(hash.Bytes()).Bytes(), privateKey)
	if err != nil {
		log.Fatalf("Failed to sign user operation hash: %v", err)
	}

	// https://stackoverflow.com/questions/69762108/implementing-ethereum-personal-sign-eip-191-from-go-ethereum-gives-different-s
	signature[64] += 27

	fmt.Println("0x" + hex.EncodeToString(signature))

	// Update the signature in the user operation
	userOp.Signature = append(leadingSignature, signature...)

	// Send the user operation
	userOpHash, err := c.SendUserOperation(ctx, &userOp, erc4337.EntryPointV07)
	if err != nil {
		log.Fatalf("Failed to send user operation: %v", err)
	}

	fmt.Printf("User Operation sent successfully!\n")
	fmt.Printf("User Operation Hash: %s\n", userOpHash.Hex())

	// Poll for user operation receipt
	fmt.Printf("Polling for user operation receipt...\n")
	maxAttempts := 60               // Maximum number of polling attempts
	pollInterval := 2 * time.Second // Wait 2 seconds between polls

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		fmt.Printf("Attempt %d/%d: Checking for receipt...\n", attempt, maxAttempts)

		receipt, err := c.GetUserOperationReceipt(ctx, userOpHash)
		if err != nil {
			fmt.Printf("Receipt not yet available (attempt %d): %v\n", attempt, err)
		} else {
			fmt.Printf("User Operation Receipt received!\n")
			fmt.Printf("  UserOpHash: %s\n", receipt.UserOpHash.Hex())
			fmt.Printf("  Sender: %s\n", receipt.Sender.Hex())
			fmt.Printf("  Success: %t\n", receipt.Success)
			fmt.Printf("  ActualGasCost: %s\n", receipt.ActualGasCost)
			fmt.Printf("  ActualGasUsed: %s\n", receipt.ActualGasUsed)
			if receipt.Paymaster != (common.Address{}) {
				fmt.Printf("  Paymaster: %s\n", receipt.Paymaster.Hex())
			}
			fmt.Printf("  Nonce: %s\n", receipt.Nonce)
			if receipt.Receipt != nil {
				fmt.Printf("  Transaction Hash: %s\n", receipt.Receipt.TransactionHash.Hex())
				fmt.Printf("  Block Number: %s\n", receipt.Receipt.BlockNumber)
				fmt.Printf("  Gas Used: %s\n", receipt.Receipt.GasUsed)
			}
			return
		}

		if attempt < maxAttempts {
			fmt.Printf("Waiting %v before next attempt...\n", pollInterval)
			time.Sleep(pollInterval)
		}
	}

	fmt.Printf("Failed to get user operation receipt after %d attempts\n", maxAttempts)
}

func personalSignHash(data []byte) common.Hash {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256Hash([]byte(msg))
}
