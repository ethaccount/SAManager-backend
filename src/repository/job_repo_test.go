package repository

import (
	"encoding/json"
	"testing"

	"math/big"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/testutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/google/uuid"
)

func TestJobRepository_RegisterJob(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)

	// Test data
	accountAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
	chainId := int64(1) // Ethereum mainnet
	jobID := int64(12345)
	entryPointAddress := common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032")

	userOperation := &erc4337.UserOperation{
		Sender:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Nonce:                (*hexutil.Big)(big.NewInt(1)),
		CallData:             hexutil.Bytes([]byte{0xab, 0xcd, 0xef}),
		CallGasLimit:         (*hexutil.Big)(big.NewInt(100000)),
		VerificationGasLimit: (*hexutil.Big)(big.NewInt(50000)),
		PreVerificationGas:   (*hexutil.Big)(big.NewInt(21000)),
		MaxPriorityFeePerGas: (*hexutil.Big)(big.NewInt(1000000000)),
		MaxFeePerGas:         (*hexutil.Big)(big.NewInt(2000000000)),
		Signature:            hexutil.Bytes([]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf}),
	}

	// Test CreateJob
	job, err := repo.CreateJob(accountAddress, chainId, jobID, userOperation, entryPointAddress)
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	// Verify the returned job
	if job == nil {
		t.Fatal("CreateJob returned nil job")
	}

	if job.ID == uuid.Nil {
		t.Error("Job ID should be generated")
	}

	if job.AccountAddress != accountAddress {
		t.Errorf("Expected accountAddress %s, got %s", accountAddress.Hex(), job.AccountAddress.Hex())
	}

	if job.OnChainJobID != jobID {
		t.Errorf("Expected OnChainJobID %d, got %d", jobID, job.OnChainJobID)
	}

	if job.ChainID != chainId {
		t.Errorf("Expected ChainID %d, got %d", chainId, job.ChainID)
	}

	if job.EntryPointAddress != entryPointAddress {
		t.Errorf("Expected EntryPointAddress %s, got %s", entryPointAddress.Hex(), job.EntryPointAddress.Hex())
	}

	if job.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if job.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	// Verify the UserOperation was stored correctly
	var storedUserOp erc4337.UserOperation
	if err := json.Unmarshal(job.UserOperation, &storedUserOp); err != nil {
		t.Fatalf("Failed to unmarshal stored UserOperation: %v", err)
	}

	if storedUserOp.Sender != userOperation.Sender {
		t.Errorf("Expected Sender %s, got %s", userOperation.Sender.Hex(), storedUserOp.Sender.Hex())
	}

	if storedUserOp.Nonce != nil && userOperation.Nonce != nil && (*big.Int)(storedUserOp.Nonce).Cmp((*big.Int)(userOperation.Nonce)) != 0 {
		t.Errorf("Expected Nonce %s, got %s", (*big.Int)(userOperation.Nonce).String(), (*big.Int)(storedUserOp.Nonce).String())
	}

	// Verify the job was actually saved to the database
	var dbJob domain.Job
	if err := db.Where("id = ?", job.ID).First(&dbJob).Error; err != nil {
		t.Fatalf("Failed to find job in database: %v", err)
	}

	if dbJob.AccountAddress != accountAddress {
		t.Errorf("Database job accountAddress mismatch: expected %s, got %s", accountAddress.Hex(), dbJob.AccountAddress.Hex())
	}

	if dbJob.OnChainJobID != jobID {
		t.Errorf("Database job OnChainJobID mismatch: expected %d, got %d", jobID, dbJob.OnChainJobID)
	}

	// Test GetUserOperation method
	retrievedUserOp, err := dbJob.GetUserOperation()
	if err != nil {
		t.Fatalf("GetUserOperation failed: %v", err)
	}

	if retrievedUserOp.Sender != userOperation.Sender {
		t.Errorf("Retrieved UserOperation Sender mismatch: expected %s, got %s", userOperation.Sender.Hex(), retrievedUserOp.Sender.Hex())
	}
}

func TestJobRepository_RegisterJob_DuplicateJobID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)

	accountAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")
	chainId := int64(1) // Ethereum mainnet
	jobID := int64(12345)
	entryPointAddress := common.HexToAddress("0x0000000071727De22E5E9d8BAf0edAc6f37da032")

	userOperation := &erc4337.UserOperation{
		Sender:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
		Nonce:     (*hexutil.Big)(big.NewInt(1)),
		CallData:  hexutil.Bytes([]byte{0xab, 0xcd, 0xef}),
		Signature: hexutil.Bytes([]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf}),
	}

	// Register first job
	_, err := repo.CreateJob(accountAddress, chainId, jobID, userOperation, entryPointAddress)
	if err != nil {
		t.Fatalf("First CreateJob failed: %v", err)
	}

	// Try to register duplicate job (same account_address and job_id)
	_, err = repo.CreateJob(accountAddress, chainId, jobID, userOperation, entryPointAddress)
	if err == nil {
		t.Error("Expected error when registering duplicate job, but got none")
	}
}
