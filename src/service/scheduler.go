package service

import (
	"context"
	"sync"
	"time"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/repository"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// JobScheduler manages job scheduling and execution
type JobScheduler struct {
	jobCache          *repository.JobCacheRepository
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	pollingInterval   int
	jobService        *JobService
	executionService  *ExecutionService
	blockchainService *BlockchainService
}

// NewJobScheduler creates a new job scheduler instance
func NewJobScheduler(ctx context.Context, jobCache *repository.JobCacheRepository, pollingInterval int, jobService *JobService, executionService *ExecutionService, blockchainService *BlockchainService) *JobScheduler {
	ctx, cancel := context.WithCancel(ctx)

	return &JobScheduler{
		jobCache:          jobCache,
		ctx:               ctx,
		cancel:            cancel,
		pollingInterval:   pollingInterval,
		jobService:        jobService,
		executionService:  executionService,
		blockchainService: blockchainService,
	}
}

func (js *JobScheduler) logger(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx).With().Str("service", "scheduler").Logger()
	return &l
}

// Start begins the polling and execution processes
func (js *JobScheduler) Start() {
	// Start polling goroutine
	js.wg.Add(1)
	go js.pollJobs()

	// Start execution goroutine
	js.wg.Add(1)
	go js.processJobs()
}

// Stop gracefully shuts down the scheduler
func (js *JobScheduler) Stop() {
	js.cancel()
	js.wg.Wait()
}

// pollJobs polls for jobs to execute every pollingInterval seconds
func (js *JobScheduler) pollJobs() {
	defer js.wg.Done()

	ticker := time.NewTicker(time.Duration(js.pollingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-js.ctx.Done():
			return
		case <-ticker.C:
			js.pollJobLogic()
		}
	}
}

// processJobs continuously processes jobs from the queue
func (js *JobScheduler) processJobs() {
	defer js.wg.Done()

	logger := js.logger(js.ctx).With().Str("function", "processJobs").Logger()

	for {
		select {
		case <-js.ctx.Done():
			return
		default:
			// Block and wait for jobs in the queue
			job, err := js.jobCache.DequeueJob(js.ctx, 1*time.Second)
			if err != nil {
				if err == redis.Nil {
					// No jobs available, continue polling
					continue
				}

				// if context was cancelled (during shutdown), ignore error
				if js.ctx.Err() != nil {
					return
				}

				logger.Error().Err(err).Msg("Error dequeuing job")
				continue
			}

			// execute the job
			js.executeJobLogic(*job)
		}
	}
}

// pollJobsLogic checks for jobs to execute and enqueues them
func (js *JobScheduler) pollJobLogic() {
	logger := js.logger(js.ctx).With().Str("function", "pollJobLogic").Logger()

	// Step 1: Process Pending Jobs

	// Step 2: Sync Cache to Database

	// Step 3: Load Active Jobs
	jobs, err := js.jobService.GetActiveJobs(js.ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get active jobs from job service")
		return
	}

	// Step 4: Validate Job Readiness

	// Step 5: Enqueue Overdue Jobs

	for _, job := range jobs {
		// Check if job should be skipped based on status in cache
		if js.shouldSkipJob(job.ID) {
			continue
		}

		// Enqueue the job
		if err := js.jobCache.EnqueueJob(js.ctx, *job); err != nil {
			logger.Error().Err(err).Msgf("Failed to enqueue job %s", job.ID)
			continue
		}

		// Set job status to pending
		if err := js.jobCache.SetJobStatus(js.ctx, job.ID, repository.StatusPending, "Job enqueued for execution", 24*time.Hour); err != nil {
			logger.Error().Err(err).Msgf("Failed to set job status for %s", job.ID)
		}
		logger.Info().Msgf("Enqueued job: %s", job.ID)
	}
}

// shouldSkipJob checks if a job should be skipped based on its current status in Redis cache
func (js *JobScheduler) shouldSkipJob(jobID uuid.UUID) bool {
	logger := js.logger(js.ctx).With().Str("function", "shouldSkipJob").Logger()

	result, err := js.jobCache.GetJobStatus(js.ctx, jobID)
	if err != nil {
		if err == redis.Nil {
			// No status found, job can be processed
			return false
		}
		logger.Error().Err(err).Msgf("Error checking job status for %s", jobID)
		return false
	}

	// Skip if status is pending
	return result.Status == repository.StatusPending
}

// executeJobLogic executes a single job and updates its status
func (js *JobScheduler) executeJobLogic(job domain.JobModel) {
	logger := js.logger(js.ctx).With().Str("function", "executeJobLogic").Logger()
	logger.Info().Str("jobID", job.ID.String()).Msg("Executing job")

	// Step 1: Execute Job

	success, message := js.testExecuteJobLogic(job)

	// Step 2: Update Job Status

	if success {
		// Remove the job status from cache since execution was successful
		if err := js.jobCache.DeleteJobStatus(js.ctx, job.ID); err != nil {
			logger.Error().Err(err).Msgf("Failed to delete job status for %s", job.ID)
		}
		logger.Info().Msgf("Job %s completed successfully", job.ID)
		return
	} else {
		logger.Error().Msgf("Job %s failed: %s", job.ID, message)
		// Update job status in cache
		if err := js.jobCache.SetJobStatus(js.ctx, job.ID, repository.StatusFailed, message, 24*time.Hour); err != nil {
			logger.Error().Err(err).Msgf("Failed to set failed job status for %s", job.ID)
		}
	}
}

// executeJobLogic simulates the actual job execution logic
func (js *JobScheduler) testExecuteJobLogic(job domain.JobModel) (bool, string) {
	// Simulate processing time
	time.Sleep(time.Duration(100+job.ID[0]%5) * time.Millisecond)
	js.logger(js.ctx).Info().Msgf("Job %s executed successfully", job.ID)

	return true, "Job executed successfully"
}
