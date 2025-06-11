package repository

import (
	"encoding/json"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethaccount/backend/src/domain"
	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) CreateJob(accountAddress common.Address, chainId int64, jobID int64, userOperation *erc4337.UserOperation, entryPoint common.Address) (*domain.JobModel, error) {
	userOpJSON, err := json.Marshal(userOperation)
	if err != nil {
		return nil, err
	}

	job := &domain.JobModel{
		AccountAddress:    accountAddress,
		ChainID:           chainId,
		OnChainJobID:      jobID,
		UserOperation:     userOpJSON,
		EntryPointAddress: entryPoint,
	}

	if err := r.db.Create(job).Error; err != nil {
		return nil, err
	}

	return job, nil
}

// FindJobs retrieves all registered jobs from the database
func (r *JobRepository) FindJobs() ([]*domain.JobModel, error) {
	var jobs []*domain.JobModel
	if err := r.db.Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// FindJobById retrieves a specific job by its ID
func (r *JobRepository) FindJobById(id string) (*domain.JobModel, error) {
	var job domain.JobModel
	if err := r.db.Where("id = ?", id).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// FindActiveJobs retrieves all jobs with "queuing" status from the database
func (r *JobRepository) FindActiveJobs() ([]*domain.JobModel, error) {
	var jobs []*domain.JobModel
	if err := r.db.Where("status = ?", domain.DBJobStatusQueuing).Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// UpdateJobStatus updates the status of a job by its ID
// If status is "failed", errMsg can be provided to set the error message
func (r *JobRepository) UpdateJobStatus(id string, status domain.DBJobStatus, errMsg *string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// If status is failed and errMsg is provided, include it in the update
	if status == domain.DBJobStatusFailed && errMsg != nil {
		updates["err_msg"] = *errMsg
	}

	if err := r.db.Model(&domain.JobModel{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}

	return nil
}
