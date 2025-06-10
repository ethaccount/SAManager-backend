package service

import (
	"context"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/repository"
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
func (s *JobService) RegisterJob(ctx context.Context, accountAddress string, chainId int64, jobID int64, userOperation *erc4337.UserOperation, entryPoint string) (*domain.Job, error) {
	s.logger(ctx).Debug().
		Str("function", "RegisterJob").
		Str("account_address", accountAddress).
		Int64("chain_id", chainId).
		Int64("job_id", jobID).
		Str("entry_point", entryPoint).
		Msg("registering new job")

	job, err := s.jobRepo.RegisterJob(accountAddress, chainId, jobID, userOperation, entryPoint)
	if err != nil {
		s.logger(ctx).Error().Err(err).Msg("failed to register job in repository")
		return nil, err
	}

	s.logger(ctx).Info().
		Str("job_uuid", job.ID.String()).
		Str("account_address", accountAddress).
		Int64("chain_id", chainId).
		Int64("job_id", jobID).
		Msg("successfully registered job")

	return job, nil
}

// GetAllActiveJobs retrieves all jobs that are available for polling
// Currently delegates to repository, but provides a place for future business logic
func (s *JobService) GetAllActiveJobs(ctx context.Context) ([]*domain.Job, error) {
	s.logger(ctx).Debug().Msg("retrieving all active jobs")

	jobs, err := s.jobRepo.GetAllJobs()
	if err != nil {
		s.logger(ctx).Error().Err(err).Msg("failed to retrieve jobs from repository")
		return nil, err
	}

	s.logger(ctx).Debug().Int("job_count", len(jobs)).Msg("retrieved jobs from repository")
	return jobs, nil
}
