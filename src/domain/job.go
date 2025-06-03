package domain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

// Job represents a job mapping in the scheduling system
type Job struct {
	ID                uuid.UUID       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	AccountAddress    string          `gorm:"type:varchar(42);not null"`
	ChainId           int64           `gorm:"not null"`
	OnChainJobID      int64           `gorm:"not null"`
	UserOperation     json.RawMessage `gorm:"type:jsonb;not null"`
	EntryPointAddress string          `gorm:"type:varchar(42);not null"`
	CreatedAt         time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt         time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// GetUserOperation returns the user operation as a typed struct
func (j *Job) GetUserOperation() (*UserOperation, error) {
	var userOp UserOperation
	if err := json.Unmarshal(j.UserOperation, &userOp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user operation: %w", err)
	}
	return &userOp, nil
}

type ExecutionConfig struct {
	ExecuteInterval             *big.Int
	NumberOfExecutions          uint16
	NumberOfExecutionsCompleted uint16
	StartDate                   *big.Int
	IsEnabled                   bool
	LastExecutionTime           *big.Int
	ExecutionData               []byte
}
