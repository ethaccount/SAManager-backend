package domain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethaccount/backend/erc4337"
	"github.com/google/uuid"
)

// Job represents a job mapping in the scheduling system
type Job struct {
	ID                uuid.UUID       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	AccountAddress    string          `gorm:"type:varchar(42);not null" json:"accountAddress"`
	ChainID           int64           `gorm:"not null" json:"chainId"`
	OnChainJobID      int64           `gorm:"not null" json:"onChainJobId"`
	UserOperation     json.RawMessage `gorm:"type:jsonb;not null" json:"userOperation"`
	EntryPointAddress string          `gorm:"type:varchar(42);not null" json:"entryPointAddress"`
	CreatedAt         time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt         time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// GetUserOperation returns the user operation as a typed struct
func (j *Job) GetUserOperation() (*erc4337.UserOperation, error) {
	var userOp erc4337.UserOperation
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

// IsTimeToExecute checks if enough time has passed since the last execution based on the configured execution interval
func (ec *ExecutionConfig) IsTimeToExecute() bool {
	if !ec.IsEnabled {
		return false
	}

	now := big.NewInt(time.Now().Unix())

	// If this is the first execution, check against start date
	if ec.LastExecutionTime == nil || ec.LastExecutionTime.Cmp(big.NewInt(0)) == 0 {
		if ec.StartDate != nil {
			return now.Cmp(ec.StartDate) >= 0
		}
		return true
	}

	// Calculate next execution time (all times are in seconds)
	nextExecutionTime := new(big.Int).Add(ec.LastExecutionTime, ec.ExecuteInterval)

	// Check if current time is >= next execution time
	return now.Cmp(nextExecutionTime) >= 0
}
