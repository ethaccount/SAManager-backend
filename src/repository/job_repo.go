package repository

import (
	"encoding/json"

	"github.com/ethaccount/backend/src/domain"
	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) RegisterJob(smartAccount string, jobID int64, userOperation *domain.UserOperation, entryPoint string) (*domain.Job, error) {
	userOpJSON, err := json.Marshal(userOperation)
	if err != nil {
		return nil, err
	}

	job := &domain.Job{
		SmartAccount:  smartAccount,
		JobID:         jobID,
		UserOperation: userOpJSON,
		EntryPoint:    entryPoint,
	}

	if err := r.db.Create(job).Error; err != nil {
		return nil, err
	}

	return job, nil
}
