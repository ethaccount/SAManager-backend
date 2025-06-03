package repository

import (
	"encoding/json"
	"testing"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/testutil"
	"github.com/google/uuid"
)

func TestJobRepository_RegisterJob(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)

	// Test data
	accountAddress := "0x1234567890123456789012345678901234567890"
	chainId := int64(1) // Ethereum mainnet
	jobID := int64(12345)
	entryPointAddress := "0x0000000071727De22E5E9d8BAf0edAc6f37da032"

	userOperation := &domain.UserOperation{
		Sender:               "0x1234567890123456789012345678901234567890",
		Nonce:                "0x1",
		CallData:             "0xabcdef",
		CallGasLimit:         "100000",
		VerificationGasLimit: "50000",
		PreVerificationGas:   "21000",
		MaxPriorityFeePerGas: "1000000000",
		MaxFeePerGas:         "2000000000",
		Signature:            "0x123456789abcdef",
	}

	// Test RegisterJob
	job, err := repo.RegisterJob(accountAddress, chainId, jobID, userOperation, entryPointAddress)
	if err != nil {
		t.Fatalf("RegisterJob failed: %v", err)
	}

	// Verify the returned job
	if job == nil {
		t.Fatal("RegisterJob returned nil job")
	}

	if job.ID == uuid.Nil {
		t.Error("Job ID should be generated")
	}

	if job.AccountAddress != accountAddress {
		t.Errorf("Expected accountAddress %s, got %s", accountAddress, job.AccountAddress)
	}

	if job.OnChainJobID != jobID {
		t.Errorf("Expected OnChainJobID %d, got %d", jobID, job.OnChainJobID)
	}

	if job.ChainID != chainId {
		t.Errorf("Expected ChainID %d, got %d", chainId, job.ChainID)
	}

	if job.EntryPointAddress != entryPointAddress {
		t.Errorf("Expected EntryPointAddress %s, got %s", entryPointAddress, job.EntryPointAddress)
	}

	if job.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if job.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}

	// Verify the UserOperation was stored correctly
	var storedUserOp domain.UserOperation
	if err := json.Unmarshal(job.UserOperation, &storedUserOp); err != nil {
		t.Fatalf("Failed to unmarshal stored UserOperation: %v", err)
	}

	if storedUserOp.Sender != userOperation.Sender {
		t.Errorf("Expected Sender %s, got %s", userOperation.Sender, storedUserOp.Sender)
	}

	if storedUserOp.Nonce != userOperation.Nonce {
		t.Errorf("Expected Nonce %s, got %s", userOperation.Nonce, storedUserOp.Nonce)
	}

	if storedUserOp.CallData != userOperation.CallData {
		t.Errorf("Expected CallData %s, got %s", userOperation.CallData, storedUserOp.CallData)
	}

	// Verify the job was actually saved to the database
	var dbJob domain.Job
	if err := db.Where("id = ?", job.ID).First(&dbJob).Error; err != nil {
		t.Fatalf("Failed to find job in database: %v", err)
	}

	if dbJob.AccountAddress != accountAddress {
		t.Errorf("Database job accountAddress mismatch: expected %s, got %s", accountAddress, dbJob.AccountAddress)
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
		t.Errorf("Retrieved UserOperation Sender mismatch: expected %s, got %s", userOperation.Sender, retrievedUserOp.Sender)
	}
}

func TestJobRepository_RegisterJob_DuplicateJobID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)

	accountAddress := "0x1234567890123456789012345678901234567890"
	chainId := int64(1) // Ethereum mainnet
	jobID := int64(12345)
	entryPointAddress := "0x0000000071727De22E5E9d8BAf0edAc6f37da032"

	userOperation := &domain.UserOperation{
		Sender:    "0x1234567890123456789012345678901234567890",
		Nonce:     "0x1",
		CallData:  "0xabcdef",
		Signature: "0x123456789abcdef",
	}

	// Register first job
	_, err := repo.RegisterJob(accountAddress, chainId, jobID, userOperation, entryPointAddress)
	if err != nil {
		t.Fatalf("First RegisterJob failed: %v", err)
	}

	// Try to register duplicate job (same account_address and job_id)
	_, err = repo.RegisterJob(accountAddress, chainId, jobID, userOperation, entryPointAddress)
	if err == nil {
		t.Error("Expected error when registering duplicate job, but got none")
	}
}
