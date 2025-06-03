package blockchain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/service"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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

func NewBlockchainService(config service.AppConfig) *BlockchainService {

	return &BlockchainService{
		SepoliaRPCURL:         config.SepoliaRPCURL,
		ArbitrumSepoliaRPCURL: config.ArbitrumSepoliaRPCURL,
		BaseSepoliaRPCURL:     config.BaseSepoliaRPCURL,
		OptimismSepoliaRPCURL: config.OptimismSepoliaRPCURL,
		PolygonAmoyRPCURL:     config.PolygonAmoyRPCURL,
	}
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

func (b *BlockchainService) GetExecutionConfig(job *domain.Job) (*domain.ExecutionConfig, error) {
	client, err := b.GetClient(job.ChainId)
	if err != nil {
		return nil, err
	}

	// ABI for executionLog(address,uint256)
	contractABI := `[{"inputs":[{"type":"address"},{"type":"uint256"}],"name":"executionLog","outputs":[{"type":"uint48"},{"type":"uint16"},{"type":"uint16"},{"type":"uint48"},{"type":"bool"},{"type":"uint48"},{"type":"bytes"}],"stateMutability":"view","type":"function"}]`

	parsedABI, _ := abi.JSON(strings.NewReader(contractABI))

	calldata, err := parsedABI.Pack("executionLog", common.HexToAddress(job.AccountAddress), big.NewInt(int64(job.OnChainJobID)))

	if err != nil {
		return nil, err
	}

	// Make the call
	addr := common.HexToAddress(scheduledTransfersAddress)
	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &addr,
		Data: calldata,
	}, nil)
	if err != nil {
		return nil, err
	}

	// Unpack the result
	unpacked, err := parsedABI.Unpack("executionLog", result)
	if err != nil {
		return nil, err
	}

	return &domain.ExecutionConfig{
		ExecuteInterval:             unpacked[0].(*big.Int),
		NumberOfExecutions:          unpacked[1].(uint16),
		NumberOfExecutionsCompleted: unpacked[2].(uint16),
		StartDate:                   unpacked[3].(*big.Int),
		IsEnabled:                   unpacked[4].(bool),
		LastExecutionTime:           unpacked[5].(*big.Int),
		ExecutionData:               unpacked[6].([]byte),
	}, nil
}

// GetExecutionConfigsBatch retrieves execution configs for multiple jobs in batch
// Groups jobs by chain ID and makes batch calls for efficiency
func (b *BlockchainService) GetExecutionConfigsBatch(jobs []*domain.Job) (map[string]*domain.ExecutionConfig, error) {
	if len(jobs) == 0 {
		return make(map[string]*domain.ExecutionConfig), nil
	}

	// Group jobs by chain ID for batch processing
	jobsByChain := make(map[int64][]*domain.Job)
	for _, job := range jobs {
		jobsByChain[job.ChainId] = append(jobsByChain[job.ChainId], job)
	}

	results := make(map[string]*domain.ExecutionConfig)

	// Process each chain separately
	for chainId, chainJobs := range jobsByChain {
		client, err := b.GetClient(chainId)
		if err != nil {
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
				return nil, fmt.Errorf("failed to call contract for job %s: %w", jobKeys[i], err)
			}

			// Unpack the result
			unpacked, err := parsedABI.Unpack("executionLog", result)
			if err != nil {
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
	}

	return results, nil
}
