package service

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethaccount/backend/src/domain"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
)

const (
	scheduledTransfersAddress = "0xA8E374779aeE60413c974b484d6509c7E4DDb6bA"
	scheduledOrdersAddress    = "0x40dc90D670C89F322fa8b9f685770296428DCb6b"
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
	clientPool            map[int64]*ethclient.Client
	mu                    sync.RWMutex
}

func NewBlockchainService(config BlockchainConfig) *BlockchainService {
	return &BlockchainService{
		SepoliaRPCURL:         &config.SepoliaRPCURL,
		ArbitrumSepoliaRPCURL: &config.ArbitrumSepoliaRPCURL,
		BaseSepoliaRPCURL:     &config.BaseSepoliaRPCURL,
		OptimismSepoliaRPCURL: &config.OptimismSepoliaRPCURL,
		PolygonAmoyRPCURL:     &config.PolygonAmoyRPCURL,
		clientPool:            make(map[int64]*ethclient.Client),
	}
}

// logger wraps the execution context with component info
func (b *BlockchainService) logger(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx).With().Str("service", "blockchain").Logger()
	return &l
}

func (b *BlockchainService) GetClient(chainId int64) (*ethclient.Client, error) {
	b.mu.RLock()
	if client, exists := b.clientPool[chainId]; exists {
		b.mu.RUnlock()
		return client, nil
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Double-check pattern
	if client, exists := b.clientPool[chainId]; exists {
		return client, nil
	}

	var rpcUrl string

	switch chainId {
	case 11155111:
		rpcUrl = *b.SepoliaRPCURL
	case 421614:
		rpcUrl = *b.ArbitrumSepoliaRPCURL
	case 84532:
		rpcUrl = *b.BaseSepoliaRPCURL
	case 11155420:
		rpcUrl = *b.OptimismSepoliaRPCURL
	case 80002:
		rpcUrl = *b.PolygonAmoyRPCURL
	default:
		return nil, fmt.Errorf("unsupported chain id: %d", chainId)
	}

	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		return nil, err
	}

	if b.clientPool == nil {
		b.clientPool = make(map[int64]*ethclient.Client)
	}
	b.clientPool[chainId] = client

	return client, nil
}

// Close closes all client connections and cleans up the connection pool
func (b *BlockchainService) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, client := range b.clientPool {
		client.Close()
	}
	b.clientPool = nil
}

// getContractAddress returns the appropriate contract address based on job type
func (b *BlockchainService) getContractAddress(jobType domain.DBJobType) (string, error) {
	switch jobType {
	case domain.DBJobTypeTransfer:
		return scheduledTransfersAddress, nil
	case domain.DBJobTypeSwap:
		return scheduledOrdersAddress, nil
	default:
		return "", fmt.Errorf("unsupported job type: %s", jobType)
	}
}

func (b *BlockchainService) GetExecutionConfig(ctx context.Context, job *domain.EntityJob) (*domain.ExecutionConfig, error) {
	b.logger(ctx).Debug().
		Str("account_address", job.AccountAddress.Hex()).
		Int64("chain_id", job.ChainID).
		Int64("job_id", int64(job.OnChainJobID)).
		Str("job_type", string(job.JobType)).
		Msg("getting execution config for job")

	client, err := b.GetClient(job.ChainID)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Int64("chain_id", job.ChainID).
			Msg("failed to get blockchain client")
		return nil, err
	}

	// Get the appropriate contract address based on job type
	contractAddress, err := b.getContractAddress(job.JobType)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Str("job_type", string(job.JobType)).
			Msg("failed to get contract address for job type")
		return nil, err
	}

	// ABI for executionLog(address,uint256)
	contractABI := `[{"inputs":[{"type":"address"},{"type":"uint256"}],"name":"executionLog","outputs":[{"type":"uint48"},{"type":"uint16"},{"type":"uint16"},{"type":"uint48"},{"type":"bool"},{"type":"uint48"},{"type":"bytes"}],"stateMutability":"view","type":"function"}]`

	parsedABI, _ := abi.JSON(strings.NewReader(contractABI))

	calldata, err := parsedABI.Pack("executionLog", job.AccountAddress, big.NewInt(int64(job.OnChainJobID)))

	if err != nil {
		b.logger(ctx).Error().Err(err).
			Str("account_address", job.AccountAddress.Hex()).
			Int64("job_id", int64(job.OnChainJobID)).
			Msg("failed to pack contract call data")
		return nil, err
	}

	// Make the call
	addr := common.HexToAddress(contractAddress)
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &addr,
		Data: calldata,
	}, nil)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Str("contract_address", contractAddress).
			Str("account_address", job.AccountAddress.Hex()).
			Int64("job_id", int64(job.OnChainJobID)).
			Str("job_type", string(job.JobType)).
			Msg("failed to call contract")
		return nil, err
	}

	// Unpack the result
	unpacked, err := parsedABI.Unpack("executionLog", result)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Str("account_address", job.AccountAddress.Hex()).
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
		Str("account_address", job.AccountAddress.Hex()).
		Int64("job_id", int64(job.OnChainJobID)).
		Str("job_type", string(job.JobType)).
		Bool("is_enabled", config.IsEnabled).
		Uint16("executions_completed", config.NumberOfExecutionsCompleted).
		Uint16("total_executions", config.NumberOfExecutions).
		Msg("successfully retrieved execution config")

	return config, nil
}

// GetExecutionConfigsBatch retrieves execution configs for multiple jobs in batch
// Groups jobs by chain ID and job type, then makes batch calls for efficiency
func (b *BlockchainService) GetExecutionConfigsBatch(ctx context.Context, jobs []*domain.EntityJob) (map[string]*domain.ExecutionConfig, error) {
	b.logger(ctx).Debug().
		Int("job_count", len(jobs)).
		Msg("getting execution configs in batch")

	if len(jobs) == 0 {
		return make(map[string]*domain.ExecutionConfig), nil
	}

	// Group jobs by chain ID and job type for batch processing
	type chainJobTypeKey struct {
		chainId int64
		jobType domain.DBJobType
	}
	jobsByChainAndType := make(map[chainJobTypeKey][]*domain.EntityJob)
	for _, job := range jobs {
		key := chainJobTypeKey{chainId: job.ChainID, jobType: job.JobType}
		jobsByChainAndType[key] = append(jobsByChainAndType[key], job)
	}

	b.logger(ctx).Debug().
		Int("chain_type_combinations", len(jobsByChainAndType)).
		Msg("grouped jobs by chain and type for batch processing")

	results := make(map[string]*domain.ExecutionConfig)

	// Process each chain-type combination separately
	for key, chainTypeJobs := range jobsByChainAndType {
		b.logger(ctx).Debug().
			Int64("chain_id", key.chainId).
			Str("job_type", string(key.jobType)).
			Int("jobs_for_chain_type", len(chainTypeJobs)).
			Msg("processing jobs for chain and type")

		client, err := b.GetClient(key.chainId)
		if err != nil {
			b.logger(ctx).Error().Err(err).
				Int64("chain_id", key.chainId).
				Msg("failed to get client for chain")
			// Return error for unsupported chains
			return nil, fmt.Errorf("failed to get client for chain %d: %w", key.chainId, err)
		}

		// Get the appropriate contract address based on job type
		contractAddress, err := b.getContractAddress(key.jobType)
		if err != nil {
			b.logger(ctx).Error().Err(err).
				Str("job_type", string(key.jobType)).
				Msg("failed to get contract address for job type")
			return nil, fmt.Errorf("failed to get contract address for job type %s: %w", key.jobType, err)
		}

		// ABI for executionLog(address,uint256)
		contractABI := `[{"inputs":[{"type":"address"},{"type":"uint256"}],"name":"executionLog","outputs":[{"type":"uint48"},{"type":"uint16"},{"type":"uint16"},{"type":"uint48"},{"type":"bool"},{"type":"uint48"},{"type":"bytes"}],"stateMutability":"view","type":"function"}]`
		parsedABI, _ := abi.JSON(strings.NewReader(contractABI))

		// Prepare batch calls
		calls := make([]ethereum.CallMsg, len(chainTypeJobs))
		jobKeys := make([]string, len(chainTypeJobs))

		for i, job := range chainTypeJobs {
			calldata, err := parsedABI.Pack("executionLog", job.AccountAddress, big.NewInt(int64(job.OnChainJobID)))
			if err != nil {
				b.logger(ctx).Error().Err(err).
					Str("job_id", job.ID.String()).
					Str("account_address", job.AccountAddress.Hex()).
					Msg("failed to pack calldata for job")
				return nil, fmt.Errorf("failed to pack calldata for job %s: %w", job.ID.String(), err)
			}

			addr := common.HexToAddress(contractAddress)
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
					Int64("chain_id", key.chainId).
					Str("job_type", string(key.jobType)).
					Str("contract_address", contractAddress).
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
			Int64("chain_id", key.chainId).
			Str("job_type", string(key.jobType)).
			Int("processed_jobs", len(chainTypeJobs)).
			Msg("successfully processed all jobs for chain and type")
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

// GetBundlerClient returns a bundler client for a given chain ID
func (b *BlockchainService) GetBundlerClient(ctx context.Context, chainId int64) (erc4337.Bundler, error) {
	b.logger(ctx).Debug().
		Int64("chain_id", chainId).
		Msg("creating bundler client")

	bundlerURL, err := b.GetBundlerURL(chainId)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Int64("chain_id", chainId).
			Msg("failed to get bundler URL")
		return nil, err
	}

	bundlerClient, err := erc4337.DialContext(ctx, bundlerURL)
	if err != nil {
		b.logger(ctx).Error().Err(err).
			Str("bundler_url", bundlerURL).
			Int64("chain_id", chainId).
			Msg("failed to create bundler client")
		return nil, fmt.Errorf("failed to create bundler client for chain %d: %w", chainId, err)
	}

	b.logger(ctx).Debug().
		Int64("chain_id", chainId).
		Str("bundler_url", bundlerURL).
		Msg("successfully created bundler client")

	return bundlerClient, nil
}
