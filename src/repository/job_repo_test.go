package repository

import (
	"encoding/json"
	"testing"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/repository/testutil"
	"github.com/google/uuid"
)

func TestJobRepository_RegisterJob(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)

	// Test data
	smartAccount := "0x1234567890123456789012345678901234567890"
	jobID := int64(12345)
	entryPoint := "0x0000000071727De22E5E9d8BAf0edAc6f37da032"

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
	job, err := repo.RegisterJob(smartAccount, jobID, userOperation, entryPoint)
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

	if job.SmartAccount != smartAccount {
		t.Errorf("Expected SmartAccount %s, got %s", smartAccount, job.SmartAccount)
	}

	if job.JobID != jobID {
		t.Errorf("Expected JobID %d, got %d", jobID, job.JobID)
	}

	if job.EntryPoint != entryPoint {
		t.Errorf("Expected EntryPoint %s, got %s", entryPoint, job.EntryPoint)
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

	if dbJob.SmartAccount != smartAccount {
		t.Errorf("Database job SmartAccount mismatch: expected %s, got %s", smartAccount, dbJob.SmartAccount)
	}

	if dbJob.JobID != jobID {
		t.Errorf("Database job JobID mismatch: expected %d, got %d", jobID, dbJob.JobID)
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

	smartAccount := "0x1234567890123456789012345678901234567890"
	jobID := int64(12345)
	entryPoint := "0x0000000071727De22E5E9d8BAf0edAc6f37da032"

	userOperation := &domain.UserOperation{
		Sender:    "0x1234567890123456789012345678901234567890",
		Nonce:     "0x1",
		CallData:  "0xabcdef",
		Signature: "0x123456789abcdef",
	}

	// Register first job
	_, err := repo.RegisterJob(smartAccount, jobID, userOperation, entryPoint)
	if err != nil {
		t.Fatalf("First RegisterJob failed: %v", err)
	}

	// Try to register duplicate job (same smart_account and job_id)
	_, err = repo.RegisterJob(smartAccount, jobID, userOperation, entryPoint)
	if err == nil {
		t.Error("Expected error when registering duplicate job, but got none")
	}
}

func TestJobRepository_RegisterJob_InvalidUserOperation(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	repo := NewJobRepository(db)

	smartAccount := "0x1234567890123456789012345678901234567890"
	jobID := int64(12345)
	entryPoint := "0x0000000071727De22E5E9d8BAf0edAc6f37da032"

	// Test with nil UserOperation
	_, err := repo.RegisterJob(smartAccount, jobID, nil, entryPoint)
	if err == nil {
		t.Error("Expected error when registering job with nil UserOperation, but got none")
	}
}
