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
	s.logger(ctx).Info().
		Str("function", "RegisterJob").
		Str("accountAddress", accountAddress).
		Int64("chainId", chainId).
		Int64("onChainJobId", jobID).
		Msg("Registering new job")

	job, err := s.jobRepo.RegisterJob(accountAddress, chainId, jobID, userOperation, entryPoint)
	if err != nil {
		return nil, err
	}

	s.logger(ctx).Info().
		Str("id", job.ID.String()).
		Str("accountAddress", accountAddress).
		Int64("chainId", chainId).
		Int64("onChainJobId", jobID).
		Msg("Successfully registered job")

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
