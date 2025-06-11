package domain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
)

// DBJobStatus represents the status of a job in the database
type DBJobStatus string

const (
	DBJobStatusQueuing   DBJobStatus = "queuing"
	DBJobStatusCompleted DBJobStatus = "completed"
	DBJobStatusFailed    DBJobStatus = "failed"
)

// JobModel represents a job in the database
type JobModel struct {
	ID                uuid.UUID       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	AccountAddress    common.Address  `gorm:"type:varchar(42);not null" json:"accountAddress"`
	ChainID           int64           `gorm:"not null" json:"chainId"`
	OnChainJobID      int64           `gorm:"not null" json:"onChainJobId"`
	UserOperation     json.RawMessage `gorm:"type:jsonb;not null" json:"userOperation"`
	EntryPointAddress common.Address  `gorm:"type:varchar(42);not null" json:"entryPointAddress"`
	Status            DBJobStatus     `gorm:"type:varchar(20);not null;default:queuing;check:status IN ('queuing', 'completed', 'failed')" json:"status"`
	ErrMsg            *string         `gorm:"type:text" json:"errMsg,omitempty"`
	CreatedAt         time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt         time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

// GetUserOperation returns the user operation as a typed struct
func (j *JobModel) GetUserOperation() (*erc4337.UserOperation, error) {
	var userOp erc4337.UserOperation
	if err := json.Unmarshal(j.UserOperation, &userOp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user operation: %w", err)
	}
	return &userOp, nil
}

// ExecutionConfig represents the job configuration from the blockchain
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
