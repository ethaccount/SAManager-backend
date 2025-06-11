package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ethaccount/backend/src/domain"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// JobStatus represents the execution status of a job
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusFailed    JobStatus = "failed"
	StatusCompleted JobStatus = "completed"
)

// JobResult contains the execution result
type JobResult struct {
	JobID     uuid.UUID `json:"job_id"`
	Status    JobStatus `json:"status"`
	Error     string    `json:"error"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JobScheduler manages job scheduling and execution
type JobScheduler struct {
	redis             *redis.Client
	queueName         string
	statusCache       string
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	pollingInterval   int
	jobService        *JobService
	executionService  *ExecutionService
	blockchainService *BlockchainService
}

// NewJobScheduler creates a new job scheduler instance
func NewJobScheduler(ctx context.Context, redis *redis.Client, queueName string, pollingInterval int, jobService *JobService, executionService *ExecutionService, blockchainService *BlockchainService) *JobScheduler {
	ctx, cancel := context.WithCancel(ctx)

	return &JobScheduler{
		redis:             redis,
		queueName:         queueName,
		statusCache:       queueName + ":status",
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

// pollJobs runs every 10 seconds to check for jobs to execute
func (js *JobScheduler) pollJobs() {
	defer js.wg.Done()

	ticker := time.NewTicker(time.Duration(js.pollingInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-js.ctx.Done():
			return
		case <-ticker.C:
			js.poll()
		}
	}
}

func (js *JobScheduler) poll() {
	logger := js.logger(js.ctx).With().Str("function", "checkAndEnqueueJobs").Logger()

	// Get failed jobs from cache and update to DB

	// Get all active jobs
	jobs, err := js.jobService.GetActiveJobs(js.ctx)
	if err != nil {
		js.logger(js.ctx).Error().Err(err).Str("function", "checkAndEnqueueJobs").Msg("Failed to get jobs from job service")
		return
	}

	for _, job := range jobs {
		// Check if job should be skipped based on status in cache
		if js.shouldSkipJob(job.ID) {
			continue
		}

		// Enqueue the job
		if err := js.enqueueJob(*job); err != nil {
			logger.Error().Err(err).Msgf("Failed to enqueue job %s", job.ID)
			continue
		}

		// Set job status to pending
		js.setJobStatus(job.ID, StatusPending, "Job enqueued for execution")
		logger.Info().Msgf("Enqueued job: %s", job.ID)
	}
}

// shouldSkipJob checks if a job should be skipped based on its current status in Redis cache
func (js *JobScheduler) shouldSkipJob(jobID uuid.UUID) bool {
	logger := js.logger(js.ctx).With().Str("function", "shouldSkipJob").Logger()
	statusKey := fmt.Sprintf("%s:%s", js.statusCache, jobID)
	statusData, err := js.redis.Get(js.ctx, statusKey).Result()
	if err != nil {
		if err == redis.Nil {
			// No status found, job can be processed
			return false
		}
		logger.Error().Err(err).Msgf("Error checking job status for %s", jobID)
		return false
	}

	var result JobResult
	if err := json.Unmarshal([]byte(statusData), &result); err != nil {
		logger.Error().Err(err).Msgf("Error unmarshaling job status for %s", jobID)
		return false
	}

	// Skip if status is pending
	return result.Status == StatusPending
}

// enqueueJob adds a job to the Redis queue
func (js *JobScheduler) enqueueJob(job domain.Job) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return js.redis.LPush(js.ctx, js.queueName, jobData).Err()
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
			result, err := js.redis.BRPop(js.ctx, 1*time.Second, js.queueName).Result()
			if err != nil {
				if err == redis.Nil {
					// No jobs available, continue polling
					continue
				}

				// if context was cancelled (during shutdown), ignore error
				if js.ctx.Err() != nil {
					return
				}

				logger.Error().Err(err).Msg("Error popping from queue")
				continue
			}

			// Parse the job
			var job domain.Job
			if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
				logger.Error().Err(err).Msg("Error unmarshaling job")
				continue
			}

			// Execute the job
			js.processJob(job)
		}
	}
}

// processJob executes a single job and updates its status
func (js *JobScheduler) processJob(job domain.Job) {
	logger := js.logger(js.ctx).With().Str("function", "processJob").Logger()
	logger.Info().Msgf("Processing job: %s", job.ID)

	// Simulate job execution
	success, message := js.executeJobLogic(job)

	var status JobStatus
	if success {
		// Remove the job status from cache since execution was successful
		statusKey := fmt.Sprintf("%s:%s", js.statusCache, job.ID)
		js.redis.Del(js.ctx, statusKey)
		logger.Info().Msgf("Job %s completed successfully", job.ID)
		return
	} else {
		status = StatusFailed
		logger.Error().Msgf("Job %s failed: %s", job.ID, message)
	}

	// Update job status in cache
	js.setJobStatus(job.ID, status, message)
}

// executeJobLogic simulates the actual job execution logic
func (js *JobScheduler) executeJobLogic(job domain.Job) (bool, string) {
	// Simulate processing time
	time.Sleep(time.Duration(100+job.ID[0]%5) * time.Millisecond)
	js.logger(js.ctx).Info().Msgf("Job %s executed successfully", job.ID)

	return true, "Job executed successfully"
}

// setJobStatus updates the job status in Redis cache
func (js *JobScheduler) setJobStatus(jobID uuid.UUID, status JobStatus, message string) {
	logger := js.logger(js.ctx).With().Str("function", "setJobStatus").Logger()

	statusKey := fmt.Sprintf("%s:%s", js.statusCache, jobID)
	result := JobResult{
		JobID:     jobID,
		Status:    status,
		Error:     message,
		UpdatedAt: time.Now(),
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		logger.Error().Err(err).Msgf("Error marshaling job result for %s", jobID)
		return
	}

	// Set with expiration (e.g., 24 hours)
	err = js.redis.Set(js.ctx, statusKey, resultData, 24*time.Hour).Err()
	if err != nil {
		logger.Error().Err(err).Msgf("Error setting job status for %s", jobID)
	}
}

// updateJobStatuses updates the status of already executed jobs
func (js *JobScheduler) updateJobStatuses() {
	logger := js.logger(js.ctx).With().Str("function", "updateJobStatuses").Logger()
	// Get all status keys
	pattern := fmt.Sprintf("%s:*", js.statusCache)
	keys, err := js.redis.Keys(js.ctx, pattern).Result()
	if err != nil {
		logger.Error().Err(err).Msg("Error getting status keys")
		return
	}

	// Check each job status and perform updates if needed
	for _, statusKey := range keys {
		statusData, err := js.redis.Get(js.ctx, statusKey).Result()
		if err != nil {
			if err != redis.Nil {
				logger.Error().Err(err).Msgf("Error getting job status from %s", statusKey)
			}
			return
		}

		var result JobResult
		if err := json.Unmarshal([]byte(statusData), &result); err != nil {
			logger.Error().Err(err).Msgf("Error unmarshaling job status from %s", statusKey)
			return
		}

		// Example: Update failed jobs older than 5 minutes to allow retry
		if result.Status == StatusFailed && time.Since(result.UpdatedAt) > 5*time.Minute {
			logger.Info().Msgf("Clearing failed status for job %s to allow retry", result.JobID)
			js.redis.Del(js.ctx, statusKey)
		}
	}
}
