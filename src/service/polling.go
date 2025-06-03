package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type PollingService struct {
	jobService        *JobService
	blockchainService *BlockchainService
	pollingInterval   time.Duration
}

type PollingConfig struct {
	PollingInterval time.Duration
}

func NewPollingService(
	_ context.Context,
	jobService *JobService,
	blockchainService *BlockchainService,
	config PollingConfig,
) *PollingService {
	return &PollingService{
		jobService:        jobService,
		blockchainService: blockchainService,
		pollingInterval:   config.PollingInterval,
	}
}

// logger wraps the execution context with component info
func (s *PollingService) logger(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx).With().Str("component", "polling-service").Logger()
	return &l
}

// Start begins the polling loop
func (s *PollingService) Start(ctx context.Context) error {
	s.logger(ctx).Info().
		Dur("polling_interval", s.pollingInterval).
		Msg("starting polling service")

	ticker := time.NewTicker(s.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger(ctx).Info().Msg("polling service stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := s.poll(ctx); err != nil {
				s.logger(ctx).Error().Err(err).Msg("polling cycle failed")
			}
		}
	}
}

// poll performs a single polling cycle
func (s *PollingService) poll(ctx context.Context) error {
	s.logger(ctx).Debug().Msg("starting polling cycle")

	// Get all registered jobs
	jobs, err := s.jobService.GetAllActiveJobs(ctx)
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		s.logger(ctx).Debug().Msg("no jobs found")
		return nil
	}

	s.logger(ctx).Debug().Int("job_count", len(jobs)).Msg("retrieved jobs")

	// Get execution configs from service in batch
	executionConfigs, err := s.blockchainService.GetExecutionConfigsBatch(jobs)
	if err != nil {
		return err
	}

	overdueJobs := 0

	// Check each job for overdue executions
	for _, job := range jobs {
		config, exists := executionConfigs[job.ID.String()]
		if !exists {
			s.logger(ctx).Warn().
				Str("job_id", job.ID.String()).
				Msg("execution config not found for job")
			continue
		}

		if config.IsTimeToExecute() {
			overdueJobs++
			s.logger(ctx).Info().
				Str("job_id", job.ID.String()).
				Str("account_address", job.AccountAddress).
				Int64("on_chain_job_id", job.OnChainJobID).
				Int64("chain_id", job.ChainId).
				Msg("job is overdue for execution")

			// TODO: Trigger execution service here
			// For now, just log the overdue job as requested (no logic)
		}
	}

	s.logger(ctx).Debug().
		Int("total_jobs", len(jobs)).
		Int("overdue_jobs", overdueJobs).
		Msg("polling cycle completed")

	return nil
}
