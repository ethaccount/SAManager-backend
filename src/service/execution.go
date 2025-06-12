package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethaccount/backend/src/domain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/rs/zerolog"
)

type ExecutionService struct {
	blockchainService *BlockchainService
	privateKey        *ecdsa.PrivateKey
}

func NewExecutionService(blockchainService *BlockchainService, privateKeyHex string) (*ExecutionService, error) {
	// Parse private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &ExecutionService{
		blockchainService: blockchainService,
		privateKey:        privateKey,
	}, nil
}

// logger wraps the execution context with component info
func (s *ExecutionService) logger(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx).With().Str("service", "execution").Logger()
	return &l
}

// personalSignHash creates an Ethereum signed message hash
func personalSignHash(data []byte) common.Hash {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256Hash([]byte(msg))
}

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

// ExecuteJob signs the user operation and sends it to the bundler
func (s *ExecutionService) ExecuteJob(ctx context.Context, job domain.EntityJob) (*common.Hash, error) {
	s.logger(ctx).Info().
		Str("job_id", job.ID.String()).
		Str("account_address", job.AccountAddress.Hex()).
		Int64("chain_id", job.ChainID).
		Int64("on_chain_job_id", job.OnChainJobID).
		Msg("executing job")

	// Get user operation from job - direct access instead of GetUserOperation
	userOp := job.UserOperation

	// Get bundler client
	bundlerClient, err := s.blockchainService.GetBundlerClient(ctx, job.ChainID)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Int64("chain_id", job.ChainID).
			Msg("failed to get bundler client")
		return nil, fmt.Errorf("failed to get bundler client: %w", err)
	}

	// Get bundler URL for RPC client
	bundlerURL, err := s.blockchainService.GetBundlerURL(job.ChainID)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Int64("chain_id", job.ChainID).
			Msg("failed to get bundler URL")
		return nil, fmt.Errorf("failed to get bundler URL: %w", err)
	}

	// Create RPC client for direct blockchain calls
	rpcClient, err := rpc.DialContext(ctx, bundlerURL)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Str("bundler_url", bundlerURL).
			Msg("failed to create RPC client")
		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	}
	defer rpcClient.Close()

	// Extract nonce key and get current nonce from entrypoint
	nonceKey, err := extractNonceKey(userOp.Nonce)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to extract nonce key")
		return nil, fmt.Errorf("failed to extract nonce key: %w", err)
	}

	s.logger(ctx).Debug().
		Str("job_id", job.ID.String()).
		Str("nonce_key", "0x"+hex.EncodeToString(nonceKey.Bytes())).
		Msg("extracted nonce key")

	currentNonce, err := getCurrentNonce(ctx, rpcClient, userOp.Sender, nonceKey)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to get current nonce")
		return nil, fmt.Errorf("failed to get current nonce: %w", err)
	}

	s.logger(ctx).Debug().
		Str("job_id", job.ID.String()).
		Str("current_nonce", "0x"+hex.EncodeToString(currentNonce.Bytes())).
		Msg("current nonce from entrypoint")

	// Update user operation with current nonce
	userOp.Nonce = (*hexutil.Big)(currentNonce)

	// Set paymaster (hardcoded for now, could be configurable)
	paymaster := common.HexToAddress("0xcD1c62f36A99f306948dB76c35Bbc1A639f92ce8")
	userOp.Paymaster = &paymaster

	// Add dummy signature for gas estimation
	dummySignature := "0xfffffffffffffffffffffffffffffff0000000000000000000000000000000007aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1c"
	decodedDummySignature, err := hexutil.Decode(dummySignature)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to decode dummy signature")
		return nil, fmt.Errorf("failed to decode dummy signature: %w", err)
	}

	leadingSignature := userOp.Signature
	userOp.Signature = append(userOp.Signature, decodedDummySignature...)

	// Estimate gas values
	entryPointAddress := job.EntryPointAddress
	estimates, err := bundlerClient.EstimateUserOperationGas(ctx, &userOp, entryPointAddress)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to estimate user operation gas")
		return nil, fmt.Errorf("failed to estimate user operation gas: %w", err)
	}

	// Get gas fees
	maxFeePerGas, maxPriorityFeePerGas, err := getMaxFeePerGas(ctx, rpcClient)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to get gas fees")
		return nil, fmt.Errorf("failed to get gas fees: %w", err)
	}

	s.logger(ctx).Debug().
		Str("job_id", job.ID.String()).
		Str("max_fee_per_gas", maxFeePerGas.String()).
		Str("max_priority_fee_per_gas", maxPriorityFeePerGas.String()).
		Str("pre_verification_gas", estimates.PreVerificationGas.String()).
		Str("verification_gas_limit", estimates.VerificationGasLimit.String()).
		Str("call_gas_limit", estimates.CallGasLimit.String()).
		Msg("gas fees and estimates")

	// Update user operation with gas values
	userOp.PreVerificationGas = (*hexutil.Big)(estimates.PreVerificationGas)
	userOp.VerificationGasLimit = (*hexutil.Big)(estimates.VerificationGasLimit)
	userOp.CallGasLimit = (*hexutil.Big)(estimates.CallGasLimit)
	userOp.PaymasterVerificationGasLimit = (*hexutil.Big)(estimates.PaymasterVerificationGasLimit)
	userOp.MaxFeePerGas = (*hexutil.Big)(maxFeePerGas)
	userOp.MaxPriorityFeePerGas = (*hexutil.Big)(maxPriorityFeePerGas)

	// Calculate user operation hash for signing
	hash, err := userOp.GetUserOpHashV07(big.NewInt(job.ChainID))
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Interface("user_op", userOp).
			Msg("failed to calculate user operation hash")
		return nil, fmt.Errorf("failed to calculate user operation hash: %w", err)
	}

	s.logger(ctx).Debug().
		Str("job_id", job.ID.String()).
		Str("user_op_hash", hash.Hex()).
		Msg("calculated user operation hash")

	// Log signer address
	signerAddress := crypto.PubkeyToAddress(s.privateKey.PublicKey)
	s.logger(ctx).Info().
		Str("job_id", job.ID.String()).
		Str("signer_address", signerAddress.Hex()).
		Msg("signing user operation")

	// Sign the user operation hash
	signature, err := crypto.Sign(personalSignHash(hash.Bytes()).Bytes(), s.privateKey)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Interface("user_op", userOp).
			Msg("failed to sign user operation hash")
		return nil, fmt.Errorf("failed to sign user operation hash: %w", err)
	}

	// Adjust signature format for Ethereum (recovery ID + 27)
	signature[64] += 27

	s.logger(ctx).Debug().
		Str("job_id", job.ID.String()).
		Str("signature", "0x"+hex.EncodeToString(signature)).
		Msg("generated signature")

	// Update the signature in the user operation (preserve any existing signature prefix)
	userOp.Signature = append(leadingSignature, signature...)

	s.logger(ctx).Debug().
		Str("job_id", job.ID.String()).
		Str("final_signature", hex.EncodeToString(userOp.Signature)).
		Msg("user operation signed successfully")

	// Send the user operation
	userOpHash, err := bundlerClient.SendUserOperation(ctx, &userOp, entryPointAddress)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Interface("user_op", userOp).
			Msg("failed to send user operation")
		return nil, fmt.Errorf("failed to send user operation: %w", err)
	}

	s.logger(ctx).Info().
		Str("job_id", job.ID.String()).
		Str("user_op_hash", userOpHash.Hex()).
		Msg("job executed successfully")

	return &userOpHash, nil
}
