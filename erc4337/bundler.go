package erc4337

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

type GasEstimates struct {
	PreVerificationGas            *hexutil.Big `json:"preVerificationGas"`
	VerificationGasLimit          *hexutil.Big `json:"verificationGasLimit"`
	CallGasLimit                  *hexutil.Big `json:"callGasLimit"`
	PaymasterVerificationGasLimit *hexutil.Big `json:"paymasterVerificationGasLimit"`
	MaxFeePerGas                  *hexutil.Big `json:"maxFeePerGas"`
	MaxPriorityFeePerGas          *hexutil.Big `json:"maxPriorityFeePerGas"`
}

type parsedTransaction struct {
	BlockHash         common.Hash    `json:"blockHash"`
	BlockNumber       string         `json:"blockNumber"`
	From              common.Address `json:"from"`
	CumulativeGasUsed string         `json:"cumulativeGasUsed"`
	GasUsed           string         `json:"gasUsed"`
	Logs              []*types.Log   `json:"logs"`
	LogsBloom         types.Bloom    `json:"logsBloom"`
	TransactionHash   common.Hash    `json:"transactionHash"`
	TransactionIndex  string         `json:"transactionIndex"`
	EffectiveGasPrice string         `json:"effectiveGasPrice"`
}

type UserOperationReceipt struct {
	UserOpHash    common.Hash        `json:"userOpHash"`
	Sender        common.Address     `json:"sender"`
	Paymaster     common.Address     `json:"paymaster"`
	Nonce         string             `json:"nonce"`
	Success       bool               `json:"success"`
	ActualGasCost string             `json:"actualGasCost"`
	ActualGasUsed string             `json:"actualGasUsed"`
	From          common.Address     `json:"from"`
	Receipt       *parsedTransaction `json:"receipt"`
	Logs          []*types.Log       `json:"logs"`
}

type Bundler interface {
	ChainId(ctx context.Context) (*big.Int, error)
	EstimateUserOperationGas(ctx context.Context, op *UserOperation, entryPoint common.Address) (*GasEstimates, error)
	SendUserOperation(ctx context.Context, op *UserOperation, entryPoint common.Address) (common.Hash, error)
	GetUserOperationReceipt(ctx context.Context, userOpHash common.Hash) (*UserOperationReceipt, error)
}

type BundlerClient struct {
	client *rpc.Client
}

func DialContext(ctx context.Context, rawurl string) (Bundler, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewBundlerClient(c), nil
}

func NewBundlerClient(c *rpc.Client) Bundler {
	return &BundlerClient{c}
}

// handleRPCError wraps RPC errors with detailed error information
func (b *BundlerClient) handleRPCError(err error, operation string) error {
	if err == nil {
		return nil
	}
	if rpcErr, ok := err.(rpc.DataError); ok {
		if data := rpcErr.ErrorData(); data != nil {
			return fmt.Errorf("bundler RPC error in %s: %s, data: %v", operation, rpcErr.Error(), data)
		}
		return fmt.Errorf("bundler RPC error in %s: %s", operation, rpcErr.Error())
	}
	return fmt.Errorf("bundler call failed in %s: %w", operation, err)
}

func (b *BundlerClient) ChainId(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := b.client.CallContext(ctx, &result, "eth_chainId", []interface{}{}...)
	if err != nil {
		return nil, b.handleRPCError(err, "eth_chainId")
	}
	return (*big.Int)(&result), nil
}

func (b *BundlerClient) EstimateUserOperationGas(ctx context.Context, op *UserOperation, entryPoint common.Address) (*GasEstimates, error) {
	var result GasEstimates
	err := b.client.CallContext(ctx, &result, "eth_estimateUserOperationGas", op, entryPoint)
	if err != nil {
		return nil, b.handleRPCError(err, "eth_estimateUserOperationGas")
	}
	return &result, nil
}

func (b *BundlerClient) SendUserOperation(ctx context.Context, op *UserOperation, entryPoint common.Address) (common.Hash, error) {
	var result common.Hash
	err := b.client.CallContext(ctx, &result, "eth_sendUserOperation", op, entryPoint)
	if err != nil {
		return result, b.handleRPCError(err, "eth_sendUserOperation")
	}
	return result, nil
}

func (b *BundlerClient) GetUserOperationReceipt(ctx context.Context, userOpHash common.Hash) (*UserOperationReceipt, error) {
	var receipt UserOperationReceipt
	err := b.client.CallContext(ctx, &receipt, "eth_getUserOperationReceipt", userOpHash)
	if err != nil {
		return nil, b.handleRPCError(err, "eth_getUserOperationReceipt")
	}

	// If UserOpHash is zero value, receipt is not ready/found
	if receipt.UserOpHash == (common.Hash{}) {
		return nil, nil
	}

	return &receipt, nil
}

// WaitForUserOpReceipt polls for user operation receipt with retry logic
func (b *BundlerClient) WaitForUserOpReceipt(ctx context.Context, userOpHash string, maxAttempts int, pollInterval time.Duration) (*UserOperationReceipt, error) {
	// Convert userOpHash string to common.Hash once
	userOpHashHex := common.HexToHash(userOpHash)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Get user operation receipt from bundler client
		receipt, err := b.GetUserOperationReceipt(ctx, userOpHashHex)
		if err != nil || receipt == nil {
			// Receipt not yet available, continue polling if not last attempt
			if attempt == maxAttempts {
				if err != nil {
					return nil, fmt.Errorf("failed to get user operation receipt for %s after %d attempts: %w", userOpHash, maxAttempts, err)
				}
				return nil, fmt.Errorf("failed to get user operation receipt for %s after %d attempts: receipt is nil", userOpHash, maxAttempts)
			}
		} else {
			return receipt, nil
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(pollInterval):
				// Continue polling
			}
		}
	}

	return nil, fmt.Errorf("failed to get user operation receipt for %s after %d attempts", userOpHash, maxAttempts)
}
