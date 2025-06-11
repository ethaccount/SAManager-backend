package service

import (
	"context"
	"sync"
	"time"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/repository"
	"github.com/ethereum/go-ethereum/common"
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

	// Step 1: Process Pending Jobs: check receipt for pending jobs and update job cache
	js.checkReceiptsForPendingJobs()

	// Step 2: Sync Cache to Database
	js.syncCacheToDatabase()

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
		if err := js.jobCache.SetJobStatus(js.ctx, job.ID, repository.CacheStatusPending, "Job enqueued for execution", 24*time.Hour); err != nil {
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
	return result.Status == repository.CacheStatusPending
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
		if err := js.jobCache.DeleteJobCache(js.ctx, job.ID); err != nil {
			logger.Error().Err(err).Msgf("Failed to delete job status for %s", job.ID)
		}
		logger.Info().Msgf("Job %s completed successfully", job.ID)
		return
	} else {
		logger.Error().Msgf("Job %s failed: %s", job.ID, message)
		// Update job status in cache
		if err := js.jobCache.SetJobStatus(js.ctx, job.ID, repository.CacheStatusFailed, message, 24*time.Hour); err != nil {
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

// groupJobsByChainID groups job caches by their chain ID for batch processing
func (js *JobScheduler) groupJobsByChainID(jobs []*repository.JobCache) map[int64][]*repository.JobCache {
	jobsByChain := make(map[int64][]*repository.JobCache)
	for _, job := range jobs {
		jobsByChain[job.ChainID] = append(jobsByChain[job.ChainID], job)
	}
	return jobsByChain
}

// syncCacheToDatabase syncs failed and completed jobs from cache to database
func (js *JobScheduler) syncCacheToDatabase() {
	logger := js.logger(js.ctx).With().Str("function", "syncCacheToDatabase").Logger()

	// Get failed jobs from cache
	failedJobs, err := js.jobCache.GetJobCachesByStatus(js.ctx, repository.CacheStatusFailed)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get failed jobs from cache")
	} else {
		js.syncJobsToDatabase(failedJobs, repository.CacheStatusFailed)
	}

	// Get completed jobs from cache
	completedJobs, err := js.jobCache.GetJobCachesByStatus(js.ctx, repository.CacheStatusCompleted)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get completed jobs from cache")
	} else {
		js.syncJobsToDatabase(completedJobs, repository.CacheStatusCompleted)
	}
}

// convertCacheStatusToDBStatus converts cache JobStatus to database JobStatus
func (js *JobScheduler) convertCacheStatusToDBStatus(cacheStatus repository.CacheJobStatus) domain.DBJobStatus {
	switch cacheStatus {
	case repository.CacheStatusFailed:
		return domain.DBJobStatusFailed
	case repository.CacheStatusCompleted:
		return domain.DBJobStatusCompleted
	default:
		// This shouldn't happen for failed/completed jobs, but default to queuing
		return domain.DBJobStatusQueuing
	}
}

// syncJobsToDatabase syncs a list of jobs to database and cleans up cache
func (js *JobScheduler) syncJobsToDatabase(jobs []*repository.JobCache, cacheStatus repository.CacheJobStatus) {
	// Convert cache status to database status
	dbStatus := js.convertCacheStatusToDBStatus(cacheStatus)

	logger := js.logger(js.ctx).With().
		Str("function", "syncJobsToDatabase").
		Str("cache_status", string(cacheStatus)).
		Str("db_status", string(dbStatus)).
		Int("count", len(jobs)).
		Logger()

	if len(jobs) == 0 {
		return
	}

	logger.Info().Msg("Syncing jobs from cache to database")

	for _, job := range jobs {
		jobLogger := logger.With().Str("job_id", job.JobID.String()).Logger()

		// Update job status in database
		var err error
		if cacheStatus == repository.CacheStatusFailed {
			err = js.jobService.UpdateJobStatus(js.ctx, job.JobID.String(), dbStatus, &job.Error)
		} else {
			err = js.jobService.UpdateJobStatus(js.ctx, job.JobID.String(), dbStatus, nil)
		}

		if err != nil {
			jobLogger.Error().Err(err).Msg("Failed to update job status in database")
			continue
		}

		// Clean up job from cache
		if err := js.jobCache.DeleteJobCache(js.ctx, job.JobID); err != nil {
			jobLogger.Error().Err(err).Msg("Failed to delete job from cache after database sync")
		} else {
			jobLogger.Info().Msg("Job synced to database and removed from cache")
		}
	}
}

// checkReceiptsForPendingJobs checks user operation receipts for pending jobs
func (js *JobScheduler) checkReceiptsForPendingJobs() {
	logger := js.logger(js.ctx).With().Str("function", "checkReceiptsForPendingJobs").Logger()

	pendingJobs, err := js.jobCache.GetJobCachesByStatus(js.ctx, repository.CacheStatusPending)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get pending jobs from job cache")
		return
	}

	if len(pendingJobs) == 0 {
		return
	}

	// Group jobs by chainId for batch processing
	jobsByChain := js.groupJobsByChainID(pendingJobs)

	logger.Info().
		Int("total_pending_jobs", len(pendingJobs)).
		Int("chains_count", len(jobsByChain)).
		Msg("Processing pending jobs for receipt checking")

	// Process each chain
	for chainID, chainJobs := range jobsByChain {
		js.checkReceiptsForChain(chainID, chainJobs)
	}
}

// checkReceiptsForChain checks receipts for all jobs on a specific chain
func (js *JobScheduler) checkReceiptsForChain(chainID int64, jobs []*repository.JobCache) {
	logger := js.logger(js.ctx).With().
		Str("function", "checkReceiptsForChain").
		Int64("chain_id", chainID).
		Logger()

	// Get bundler client for this chain
	bundlerClient, err := js.blockchainService.GetBundlerClient(js.ctx, chainID)
	if err != nil {
		logger.Error().Err(err).
			Int64("chain_id", chainID).
			Msg("Failed to get bundler client")
		return
	}

	logger.Debug().
		Int64("chain_id", chainID).
		Int("jobs_count", len(jobs)).
		Msg("Checking receipts for jobs on chain")

	// Check receipts for each job
	for _, job := range jobs {
		js.checkSingleJobReceipt(bundlerClient, job)
	}
}

// checkSingleJobReceipt checks the receipt for a single job
func (js *JobScheduler) checkSingleJobReceipt(bundlerClient interface{}, job *repository.JobCache) {
	logger := js.logger(js.ctx).With().
		Str("function", "checkSingleJobReceipt").
		Str("job_id", job.JobID.String()).
		Logger()

	// Cast to BundlerClient to access GetUserOperationReceipt
	client, ok := bundlerClient.(*erc4337.BundlerClient)
	if !ok {
		logger.Error().Msg("Failed to cast bundler client to BundlerClient type")
		return
	}

	// Check if UserOpHash is valid (not zero)
	if job.UserOpHash == (common.Hash{}) {
		logger.Warn().
			Str("job_id", job.JobID.String()).
			Msg("Job has empty user operation hash, skipping receipt check")
		return
	}

	// Get the receipt
	receipt, err := client.GetUserOperationReceipt(js.ctx, job.UserOpHash)
	if err != nil {
		logger.Error().Err(err).
			Str("job_id", job.JobID.String()).
			Str("user_op_hash", job.UserOpHash.Hex()).
			Msg("Failed to get user operation receipt")
		return
	}

	// Handle receipt result
	if receipt == nil {
		// Receipt not found yet, job is still pending
		logger.Debug().
			Str("job_id", job.JobID.String()).
			Str("user_op_hash", job.UserOpHash.Hex()).
			Msg("Receipt not found yet, job still pending")
		return
	}

	// Receipt found - check if it's successful
	logger.Info().
		Str("job_id", job.JobID.String()).
		Str("user_op_hash", receipt.UserOpHash.Hex()).
		Bool("success", receipt.Success).
		Msg("Receipt found for pending job")

	if receipt.Success {
		// Job completed successfully, remove from cache
		if err := js.jobCache.DeleteJobCache(js.ctx, job.JobID); err != nil {
			logger.Error().Err(err).
				Str("job_id", job.JobID.String()).
				Msg("Failed to delete successful job status from cache")
		} else {
			logger.Info().
				Str("job_id", job.JobID.String()).
				Msg("Successfully completed job removed from cache")
		}
	} else {
		// Job failed, update status
		errorMsg := "User operation failed on-chain"
		if err := js.jobCache.SetJobStatus(js.ctx, job.JobID, repository.CacheStatusFailed, errorMsg, 24*time.Hour); err != nil {
			logger.Error().Err(err).
				Str("job_id", job.JobID.String()).
				Msg("Failed to update failed job status in cache")
		} else {
			logger.Info().
				Str("job_id", job.JobID.String()).
				Msg("Job marked as failed due to on-chain failure")
		}
	}
}
