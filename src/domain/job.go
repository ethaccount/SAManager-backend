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

// DBJob represents a job in the database (persistence layer)
type DBJob struct {
	ID                uuid.UUID       `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	AccountAddress    string          `gorm:"type:varchar(42);not null" json:"accountAddress"`
	ChainID           int64           `gorm:"not null" json:"chainId"`
	OnChainJobID      int64           `gorm:"not null" json:"onChainJobId"`
	UserOperation     json.RawMessage `gorm:"type:jsonb;not null" json:"userOperation"`
	EntryPointAddress string          `gorm:"type:varchar(42);not null" json:"entryPointAddress"`
	Status            DBJobStatus     `gorm:"type:varchar(20);not null;default:queuing;check:status IN ('queuing', 'completed', 'failed')" json:"status"`
	ErrMsg            *string         `gorm:"type:text" json:"errMsg,omitempty"`
	CreatedAt         time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt         time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

func (DBJob) TableName() string {
	return "jobs"
}

// ToEntityJob converts DBJob to EntityJob
func (j *DBJob) ToEntityJob() (*EntityJob, error) {
	var userOp erc4337.UserOperation
	if err := json.Unmarshal(j.UserOperation, &userOp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user operation: %w", err)
	}

	return &EntityJob{
		ID:                j.ID,
		AccountAddress:    common.HexToAddress(j.AccountAddress),
		ChainID:           j.ChainID,
		OnChainJobID:      j.OnChainJobID,
		UserOperation:     userOp,
		EntryPointAddress: common.HexToAddress(j.EntryPointAddress),
		Status:            j.Status,
		ErrMsg:            j.ErrMsg,
		CreatedAt:         j.CreatedAt,
		UpdatedAt:         j.UpdatedAt,
	}, nil
}

// EntityJob represents a job in the database
type EntityJob struct {
	ID                uuid.UUID
	AccountAddress    common.Address
	ChainID           int64
	OnChainJobID      int64
	UserOperation     erc4337.UserOperation
	EntryPointAddress common.Address
	Status            DBJobStatus
	ErrMsg            *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (rj *EntityJob) ToDBJob() (*DBJob, error) {
	userOpJSON, err := json.Marshal(rj.UserOperation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user operation: %w", err)
	}

	return &DBJob{
		ID:                rj.ID,
		AccountAddress:    rj.AccountAddress.Hex(),
		ChainID:           rj.ChainID,
		OnChainJobID:      rj.OnChainJobID,
		UserOperation:     userOpJSON,
		EntryPointAddress: rj.EntryPointAddress.Hex(),
		Status:            rj.Status,
		ErrMsg:            rj.ErrMsg,
		CreatedAt:         rj.CreatedAt,
		UpdatedAt:         rj.UpdatedAt,
	}, nil
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
