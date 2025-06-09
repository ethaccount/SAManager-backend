package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethaccount/backend/src/domain"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
)

const (
	scheduledTransfersAddress = "0xA8E374779aeE60413c974b484d6509c7E4DDb6bA"
)

type BlockchainConfig struct {
	SepoliaRPCURL         string
	ArbitrumSepoliaRPCURL string
	BaseSepoliaRPCURL     string
	OptimismSepoliaRPCURL string
	PolygonAmoyRPCURL     string
}

type BlockchainService struct {
	SepoliaRPCURL         *string
	ArbitrumSepoliaRPCURL *string
	BaseSepoliaRPCURL     *string
	OptimismSepoliaRPCURL *string
	PolygonAmoyRPCURL     *string
}

func NewBlockchainService(config BlockchainConfig) *BlockchainService {
	return &BlockchainService{
		SepoliaRPCURL:         &config.SepoliaRPCURL,
		ArbitrumSepoliaRPCURL: &config.ArbitrumSepoliaRPCURL,
		BaseSepoliaRPCURL:     &config.BaseSepoliaRPCURL,
		OptimismSepoliaRPCURL: &config.OptimismSepoliaRPCURL,
		PolygonAmoyRPCURL:     &config.PolygonAmoyRPCURL,
	}
}

// logger wraps the execution context with component info
func (b *BlockchainService) logger(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx).With().Str("service", "blockchain").Logger()
	return &l
}

func (b *BlockchainService) GetClient(chainId int64) (*ethclient.Client, error) {
	var rpcUrl string

	if chainId == 11155111 {
		rpcUrl = *b.SepoliaRPCURL
	} else if chainId == 42161 {
		rpcUrl = *b.ArbitrumSepoliaRPCURL
	} else if chainId == 84532 {
		rpcUrl = *b.BaseSepoliaRPCURL
	} else if chainId == 1101 {
		rpcUrl = *b.OptimismSepoliaRPCURL
	} else if chainId == 137 {
		rpcUrl = *b.PolygonAmoyRPCURL
	} else {
		return nil, fmt.Errorf("unsupported chain id: %d", chainId)
	}

	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (b *BlockchainService) GetExecutionConfig(ctx context.Context, job *domain.Job) (*domain.ExecutionConfig, error) {
	b.logger(ctx).Debug().
		Str("account_address", job.AccountAddress).
		Int64("chain_id", job.ChainID).
		Int64("job_id", int64(job.OnChainJobID)).
		Msg("getting execution config for job")

	client, err := b.GetClient(job.ChainID)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Int64("chain_id", job.ChainID).
			Msg("failed to get blockchain client")
		return nil, err
	}

	// ABI for executionLog(address,uint256)
	contractABI := `[{"inputs":[{"type":"address"},{"type":"uint256"}],"name":"executionLog","outputs":[{"type":"uint48"},{"type":"uint16"},{"type":"uint16"},{"type":"uint48"},{"type":"bool"},{"type":"uint48"},{"type":"bytes"}],"stateMutability":"view","type":"function"}]`

	parsedABI, _ := abi.JSON(strings.NewReader(contractABI))

	calldata, err := parsedABI.Pack("executionLog", common.HexToAddress(job.AccountAddress), big.NewInt(int64(job.OnChainJobID)))

	if err != nil {
		b.logger(ctx).Error().Err(err).
			Str("account_address", job.AccountAddress).
			Int64("job_id", int64(job.OnChainJobID)).
			Msg("failed to pack contract call data")
		return nil, err
	}

	// Make the call
	addr := common.HexToAddress(scheduledTransfersAddress)
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &addr,
		Data: calldata,
	}, nil)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Str("contract_address", scheduledTransfersAddress).
			Str("account_address", job.AccountAddress).
			Int64("job_id", int64(job.OnChainJobID)).
			Msg("failed to call contract")
		return nil, err
	}

	// Unpack the result
	unpacked, err := parsedABI.Unpack("executionLog", result)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Str("account_address", job.AccountAddress).
			Int64("job_id", int64(job.OnChainJobID)).
			Msg("failed to unpack contract result")
		return nil, err
	}

	config := &domain.ExecutionConfig{
		ExecuteInterval:             unpacked[0].(*big.Int),
		NumberOfExecutions:          unpacked[1].(uint16),
		NumberOfExecutionsCompleted: unpacked[2].(uint16),
		StartDate:                   unpacked[3].(*big.Int),
		IsEnabled:                   unpacked[4].(bool),
		LastExecutionTime:           unpacked[5].(*big.Int),
		ExecutionData:               unpacked[6].([]byte),
	}

	b.logger(ctx).Debug().
		Str("account_address", job.AccountAddress).
		Int64("job_id", int64(job.OnChainJobID)).
		Bool("is_enabled", config.IsEnabled).
		Uint16("executions_completed", config.NumberOfExecutionsCompleted).
		Uint16("total_executions", config.NumberOfExecutions).
		Msg("successfully retrieved execution config")

	return config, nil
}

// GetExecutionConfigsBatch retrieves execution configs for multiple jobs in batch
// Groups jobs by chain ID and makes batch calls for efficiency
func (b *BlockchainService) GetExecutionConfigsBatch(ctx context.Context, jobs []*domain.Job) (map[string]*domain.ExecutionConfig, error) {
	b.logger(ctx).Debug().
		Int("job_count", len(jobs)).
		Msg("getting execution configs in batch")

	if len(jobs) == 0 {
		return make(map[string]*domain.ExecutionConfig), nil
	}

	// Group jobs by chain ID for batch processing
	jobsByChain := make(map[int64][]*domain.Job)
	for _, job := range jobs {
		jobsByChain[job.ChainID] = append(jobsByChain[job.ChainID], job)
	}

	b.logger(ctx).Debug().
		Int("chain_count", len(jobsByChain)).
		Msg("grouped jobs by chain for batch processing")

	results := make(map[string]*domain.ExecutionConfig)

	// Process each chain separately
	for chainId, chainJobs := range jobsByChain {
		b.logger(ctx).Debug().
			Int64("chain_id", chainId).
			Int("jobs_for_chain", len(chainJobs)).
			Msg("processing jobs for chain")

		client, err := b.GetClient(chainId)
		if err != nil {
			b.logger(ctx).Error().Err(err).
				Int64("chain_id", chainId).
				Msg("failed to get client for chain")
			// Return error for unsupported chains
			return nil, fmt.Errorf("failed to get client for chain %d: %w", chainId, err)
		}

		// ABI for executionLog(address,uint256)
		contractABI := `[{"inputs":[{"type":"address"},{"type":"uint256"}],"name":"executionLog","outputs":[{"type":"uint48"},{"type":"uint16"},{"type":"uint16"},{"type":"uint48"},{"type":"bool"},{"type":"uint48"},{"type":"bytes"}],"stateMutability":"view","type":"function"}]`
		parsedABI, _ := abi.JSON(strings.NewReader(contractABI))

		// Prepare batch calls
		calls := make([]ethereum.CallMsg, len(chainJobs))
		jobKeys := make([]string, len(chainJobs))

		for i, job := range chainJobs {
			calldata, err := parsedABI.Pack("executionLog", common.HexToAddress(job.AccountAddress), big.NewInt(int64(job.OnChainJobID)))
			if err != nil {
				b.logger(ctx).Error().Err(err).
					Str("job_id", job.ID.String()).
					Str("account_address", job.AccountAddress).
					Msg("failed to pack calldata for job")
				return nil, fmt.Errorf("failed to pack calldata for job %s: %w", job.ID.String(), err)
			}

			addr := common.HexToAddress(scheduledTransfersAddress)
			calls[i] = ethereum.CallMsg{
				To:   &addr,
				Data: calldata,
			}
			jobKeys[i] = job.ID.String()
		}

		// Execute batch calls
		for i, call := range calls {
			result, err := client.CallContract(context.Background(), call, nil)
			if err != nil {
				b.logger(ctx).Error().Err(err).
					Str("job_id", jobKeys[i]).
					Int64("chain_id", chainId).
					Msg("failed to call contract for job")
				return nil, fmt.Errorf("failed to call contract for job %s: %w", jobKeys[i], err)
			}

			// Unpack the result
			unpacked, err := parsedABI.Unpack("executionLog", result)
			if err != nil {
				b.logger(ctx).Error().Err(err).
					Str("job_id", jobKeys[i]).
					Msg("failed to unpack result for job")
				return nil, fmt.Errorf("failed to unpack result for job %s: %w", jobKeys[i], err)
			}

			results[jobKeys[i]] = &domain.ExecutionConfig{
				ExecuteInterval:             unpacked[0].(*big.Int),
				NumberOfExecutions:          unpacked[1].(uint16),
				NumberOfExecutionsCompleted: unpacked[2].(uint16),
				StartDate:                   unpacked[3].(*big.Int),
				IsEnabled:                   unpacked[4].(bool),
				LastExecutionTime:           unpacked[5].(*big.Int),
				ExecutionData:               unpacked[6].([]byte),
			}
		}

		b.logger(ctx).Debug().
			Int64("chain_id", chainId).
			Int("processed_jobs", len(chainJobs)).
			Msg("successfully processed all jobs for chain")
	}

	b.logger(ctx).Info().
		Int("total_jobs", len(jobs)).
		Int("total_configs", len(results)).
		Msg("successfully retrieved execution configs in batch")

	return results, nil
}

// GetBundlerURL returns the bundler URL for a given chain ID
func (b *BlockchainService) GetBundlerURL(chainId int64) (string, error) {
	switch chainId {
	case 11155111: // Sepolia
		return *b.SepoliaRPCURL, nil
	case 421614: // Arbitrum Sepolia
		return *b.ArbitrumSepoliaRPCURL, nil
	case 84532: // Base Sepolia
		return *b.BaseSepoliaRPCURL, nil
	case 11155420: // Optimism Sepolia
		return *b.OptimismSepoliaRPCURL, nil
	case 80002: // Polygon Amoy
		return *b.PolygonAmoyRPCURL, nil
	default:
		return "", fmt.Errorf("unsupported chain id for bundler: %d", chainId)
	}
}

// SendUserOperation sends a user operation to the bundler
func (b *BlockchainService) SendUserOperation(ctx context.Context, userOp *domain.UserOperation, entryPoint string, chainId int64) (string, error) {
	b.logger(ctx).Debug().
		Str("sender", userOp.Sender).
		Int64("chain_id", chainId).
		Str("entry_point", entryPoint).
		Msg("sending user operation to bundler")

	bundlerURL, err := b.GetBundlerURL(chainId)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Int64("chain_id", chainId).
			Msg("failed to get bundler URL")
		return "", err
	}

	// Prepare JSON-RPC request
	rpcRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_sendUserOperation",
		"params":  []interface{}{userOp, entryPoint},
		"id":      1,
	}

	requestBody, err := json.Marshal(rpcRequest)
	if err != nil {
		b.logger(ctx).Error().Err(err).Msg("failed to marshal RPC request")
		return "", fmt.Errorf("failed to marshal RPC request: %w", err)
	}

	// Send HTTP request to bundler
	resp, err := http.Post(bundlerURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Str("bundler_url", bundlerURL).
			Msg("failed to send request to bundler")
		return "", fmt.Errorf("failed to send request to bundler: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		b.logger(ctx).Error().Err(err).Msg("failed to read bundler response")
		return "", fmt.Errorf("failed to read bundler response: %w", err)
	}

	// Parse JSON-RPC response
	var rpcResponse struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      int         `json:"id"`
		Result  string      `json:"result,omitempty"`
		Error   interface{} `json:"error,omitempty"`
	}

	if err := json.Unmarshal(responseBody, &rpcResponse); err != nil {
		b.logger(ctx).Error().Err(err).
			Str("response_body", string(responseBody)).
			Msg("failed to parse bundler response")
		return "", fmt.Errorf("failed to parse bundler response: %w", err)
	}

	// Check for RPC error
	if rpcResponse.Error != nil {
		b.logger(ctx).Error().
			Interface("rpc_error", rpcResponse.Error).
			Msg("bundler returned error")
		return "", fmt.Errorf("bundler error: %v", rpcResponse.Error)
	}

	b.logger(ctx).Info().
		Str("user_op_hash", rpcResponse.Result).
		Str("sender", userOp.Sender).
		Int64("chain_id", chainId).
		Msg("successfully sent user operation to bundler")

	return rpcResponse.Result, nil
}
