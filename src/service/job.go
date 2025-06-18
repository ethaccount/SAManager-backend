package service

import (
	"context"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/repository"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog"
)

type JobService struct {
	jobRepo *repository.JobRepository
}

func NewJobService(jobRepo *repository.JobRepository) *JobService {
	return &JobService{
		jobRepo: jobRepo,
	}
}

// logger wraps the execution context with component info
func (s *JobService) logger(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx).With().Str("service", "job").Logger()
	return &l
}

// RegisterJob creates a new job registration
func (s *JobService) RegisterJob(ctx context.Context, accountAddress common.Address, chainId int64, jobID int64, jobType domain.DBJobType, userOperation *erc4337.UserOperation, entryPoint common.Address) (*domain.EntityJob, error) {
	s.logger(ctx).Info().
		Str("function", "RegisterJob").
		Str("accountAddress", accountAddress.Hex()).
		Int64("chainId", chainId).
		Int64("onChainJobId", jobID).
		Str("jobType", string(jobType)).
		Msg("Registering new job")

	job, err := s.jobRepo.CreateJob(accountAddress, chainId, jobID, jobType, userOperation, entryPoint)
	if err != nil {
		return nil, err
	}

	s.logger(ctx).Info().
		Str("id", job.ID.String()).
		Str("accountAddress", accountAddress.Hex()).
		Int64("chainId", chainId).
		Int64("onChainJobId", jobID).
		Str("jobType", string(jobType)).
		Msg("Successfully registered job")

	return job, nil
}

// GetActiveJobs retrieves all jobs that are available for polling
func (s *JobService) GetActiveJobs(ctx context.Context) ([]*domain.EntityJob, error) {
	s.logger(ctx).Debug().Msg("retrieving all active jobs")

	jobs, err := s.jobRepo.FindActiveJobs()
	if err != nil {
		s.logger(ctx).Error().Err(err).Msg("failed to retrieve jobs from repository")
		return nil, err
	}

	s.logger(ctx).Debug().Int("job_count", len(jobs)).Msg("retrieved jobs from repository")
	return jobs, nil
}

// GetJobByID retrieves a specific job by its ID
func (s *JobService) GetJobByID(ctx context.Context, id string) (*domain.EntityJob, error) {
	s.logger(ctx).Debug().
		Str("function", "GetJobByID").
		Str("job_id", id).
		Msg("retrieving job by ID")

	job, err := s.jobRepo.FindJobById(id)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", id).
			Msg("failed to retrieve job from repository")
		return nil, err
	}

	s.logger(ctx).Debug().
		Str("job_id", id).
		Msg("successfully retrieved job")
	return job, nil
}

// UpdateJobStatus updates the status of a job by its ID
func (s *JobService) UpdateJobStatus(ctx context.Context, id string, status domain.DBJobStatus, errMsg *string) error {
	s.logger(ctx).Debug().
		Str("function", "UpdateJobStatus").
		Str("job_id", id).
		Str("status", string(status)).
		Msg("updating job status")

	err := s.jobRepo.UpdateJobStatus(id, status, errMsg)
	if err != nil {
		s.logger(ctx).Error().Err(err).
			Str("job_id", id).
			Str("status", string(status)).
			Msg("failed to update job status in repository")
		return err
	}

	s.logger(ctx).Info().
		Str("job_id", id).
		Str("status", string(status)).
		Msg("successfully updated job status")
	return nil
}
