package service

import (
	"context"
	"fmt"
	"math/big"
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

type BlockchainService struct {
	SepoliaRPCURL         *string
	ArbitrumSepoliaRPCURL *string
	BaseSepoliaRPCURL     *string
	OptimismSepoliaRPCURL *string
	PolygonAmoyRPCURL     *string
}

func NewBlockchainService(config AppConfig) *BlockchainService {
	return &BlockchainService{
		SepoliaRPCURL:         config.SepoliaRPCURL,
		ArbitrumSepoliaRPCURL: config.ArbitrumSepoliaRPCURL,
		BaseSepoliaRPCURL:     config.BaseSepoliaRPCURL,
		OptimismSepoliaRPCURL: config.OptimismSepoliaRPCURL,
		PolygonAmoyRPCURL:     config.PolygonAmoyRPCURL,
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
