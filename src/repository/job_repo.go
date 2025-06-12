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

func (r *JobRepository) CreateJob(accountAddress common.Address, chainId int64, jobID int64, userOperation *erc4337.UserOperation, entryPoint common.Address) (*domain.EntityJob, error) {
	userOpJSON, err := json.Marshal(userOperation)
	if err != nil {
		return nil, err
	}

	dbJob := &domain.DBJob{
		AccountAddress:    accountAddress.Hex(),
		ChainID:           chainId,
		OnChainJobID:      jobID,
		UserOperation:     userOpJSON,
		EntryPointAddress: entryPoint.Hex(),
		Status:            domain.DBJobStatusQueuing,
	}

	if err := r.db.Create(dbJob).Error; err != nil {
		return nil, err
	}

	return dbJob.ToEntityJob()
}

// FindJobs retrieves all registered jobs from the database
func (r *JobRepository) FindJobs() ([]*domain.EntityJob, error) {
	var dbJobs []*domain.DBJob
	if err := r.db.Find(&dbJobs).Error; err != nil {
		return nil, err
	}

	jobs := make([]*domain.EntityJob, len(dbJobs))
	for i, dbJob := range dbJobs {
		registeredJob, err := dbJob.ToEntityJob()
		if err != nil {
			return nil, err
		}
		jobs[i] = registeredJob
	}
	return jobs, nil
}

// FindJobById retrieves a specific job by its ID
func (r *JobRepository) FindJobById(id string) (*domain.EntityJob, error) {
	var dbJob domain.DBJob
	if err := r.db.Where("id = ?", id).First(&dbJob).Error; err != nil {
		return nil, err
	}
	return dbJob.ToEntityJob()
}

// FindActiveJobs retrieves all jobs with "queuing" status from the database
func (r *JobRepository) FindActiveJobs() ([]*domain.EntityJob, error) {
	var dbJobs []*domain.DBJob
	if err := r.db.Where("status = ?", domain.DBJobStatusQueuing).Find(&dbJobs).Error; err != nil {
		return nil, err
	}

	jobs := make([]*domain.EntityJob, len(dbJobs))
	for i, dbJob := range dbJobs {
		registeredJob, err := dbJob.ToEntityJob()
		if err != nil {
			return nil, err
		}
		jobs[i] = registeredJob
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

	if err := r.db.Model(&domain.DBJob{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}

	return nil
}
