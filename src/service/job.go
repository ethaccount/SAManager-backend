package service

import (
	"context"

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
	l := zerolog.Ctx(ctx).With().Str("component", "job-service").Logger()
	return &l
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
