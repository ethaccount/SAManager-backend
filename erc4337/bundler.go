package erc4337

import (
	"context"
	"math/big"

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

func (b *BundlerClient) ChainId(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := b.client.CallContext(ctx, &result, "eth_chainId", []interface{}{}...)
	if err != nil {
		return nil, err
	}
	return (*big.Int)(&result), nil
}

func (b *BundlerClient) EstimateUserOperationGas(ctx context.Context, op *UserOperation, entryPoint common.Address) (*GasEstimates, error) {
	var estimate GasEstimates
	err := b.client.CallContext(ctx, &estimate, "eth_estimateUserOperationGas", op, entryPoint)
	if err != nil {
		return nil, err
	}
	return &estimate, nil
}

func (b *BundlerClient) SendUserOperation(ctx context.Context, op *UserOperation, entryPoint common.Address) (common.Hash, error) {
	var result common.Hash
	err := b.client.CallContext(ctx, &result, "eth_sendUserOperation", op, entryPoint)
	return result, err
}

func (b *BundlerClient) GetUserOperationReceipt(ctx context.Context, userOpHash common.Hash) (*UserOperationReceipt, error) {
	var receipt UserOperationReceipt
	err := b.client.CallContext(ctx, &receipt, "eth_getUserOperationReceipt", userOpHash)
	if err != nil {
		return nil, err
	}
	return &receipt, nil
}
