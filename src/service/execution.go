package service

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

// ExecuteJob signs the user operation and sends it to the bundler
func (s *ExecutionService) ExecuteJob(ctx context.Context, job *domain.Job) (string, error) {
	s.logger(ctx).Info().
		Str("job_id", job.ID.String()).
		Str("account_address", job.AccountAddress).
		Int64("chain_id", job.ChainID).
		Int64("on_chain_job_id", job.OnChainJobID).
		Msg("executing job")

	// Get user operation from job
	userOp, err := job.GetUserOperation()
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to get user operation from job")
		return "", fmt.Errorf("failed to get user operation: %w", err)
	}

	// Create user operation hash for signing
	userOpHash, err := s.createUserOperationHash(userOp, job.EntryPointAddress, job.ChainID)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to create user operation hash")
		return "", fmt.Errorf("failed to create user operation hash: %w", err)
	}

	// Sign the user operation hash
	signature, err := crypto.Sign(userOpHash, s.privateKey)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to sign user operation")
		return "", fmt.Errorf("failed to sign user operation: %w", err)
	}

	// Append signature to user operation
	userOp.Signature = "0x" + common.Bytes2Hex(signature)

	s.logger(ctx).Debug().
		Str("job_id", job.ID.String()).
		Str("signature", userOp.Signature).
		Msg("user operation signed successfully")

	// Send user operation to bundler
	userOpHashString, err := s.blockchainService.SendUserOperation(ctx, userOp, job.EntryPointAddress, job.ChainID)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", job.ID.String()).
			Msg("failed to send user operation to bundler")
		return "", fmt.Errorf("failed to send user operation to bundler: %w", err)
	}

	s.logger(ctx).Info().
		Str("job_id", job.ID.String()).
		Str("user_op_hash", userOpHashString).
		Msg("job executed successfully")

	return userOpHashString, nil
}

// createUserOperationHash creates the hash that needs to be signed for the user operation
// This follows the ERC-4337 specification for user operation hashing
func (s *ExecutionService) createUserOperationHash(userOp *domain.UserOperation, entryPoint string, chainId int64) ([]byte, error) {
	return GetUserOpHash(userOp, entryPoint, chainId)
}
