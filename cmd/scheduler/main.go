package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Job represents a job to be executed
type Job struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Payload     map[string]string `json:"payload"`
	ScheduledAt time.Time         `json:"scheduled_at"`
	CreatedAt   time.Time         `json:"created_at"`
}

// JobStatus represents the execution status of a job
type JobStatus string

const (
	StatusPending JobStatus = "pending"
	StatusSuccess JobStatus = "success"
	StatusFailed  JobStatus = "failed"
)

// JobCache contains the execution result
type JobCache struct {
	JobID     string    `json:"job_id"`
	Status    JobStatus `json:"status"`
	Message   string    `json:"message"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JobScheduler manages job scheduling and execution
type JobScheduler struct {
	redis       *redis.Client
	queueName   string
	statusCache string
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewJobScheduler creates a new job scheduler instance
func NewJobScheduler(redisAddr, queueName string) *JobScheduler {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx, cancel := context.WithCancel(context.Background())

	return &JobScheduler{
		redis:       rdb,
		queueName:   queueName,
		statusCache: queueName + ":status",
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the polling and execution processes
func (js *JobScheduler) Start() {
	log.Println("Starting job scheduler...")

	// Start polling goroutine
	js.wg.Add(1)
	go js.pollJobs()

	// Start execution goroutine
	js.wg.Add(1)
	go js.executeJobs()
}

// Stop gracefully shuts down the scheduler
func (js *JobScheduler) Stop() {
	log.Println("Stopping job scheduler...")
	js.cancel()
	js.wg.Wait()
	js.redis.Close()
}

// pollJobs runs every 10 seconds to check for jobs to execute
func (js *JobScheduler) pollJobs() {
	defer js.wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-js.ctx.Done():
			return
		case <-ticker.C:
			js.checkAndEnqueueJobs()
			js.updateJobStatuses()
		}
	}
}

// checkAndEnqueueJobs finds jobs that need to be executed and enqueues them
func (js *JobScheduler) checkAndEnqueueJobs() {
	// Simulate getting jobs from database or external source
	jobsToCheck := js.getJobsToCheck()

	for _, job := range jobsToCheck {
		// Check if job should be skipped based on status
		if js.shouldSkipJob(job.ID) {
			continue
		}

		// Enqueue the job
		if err := js.enqueueJob(job); err != nil {
			log.Printf("Failed to enqueue job %s: %v", job.ID, err)
			continue
		}

		// Set status to pending
		js.setJobStatus(job.ID, StatusPending, "Job enqueued for execution")
		log.Printf("Enqueued job: %s", job.ID)
	}
}

// shouldSkipJob checks if a job should be skipped based on its current status
func (js *JobScheduler) shouldSkipJob(jobID string) bool {
	statusKey := fmt.Sprintf("%s:%s", js.statusCache, jobID)
	statusData, err := js.redis.Get(js.ctx, statusKey).Result()
	if err != nil {
		if err == redis.Nil {
			// No status found, job can be processed
			return false
		}
		log.Printf("Error checking job status for %s: %v", jobID, err)
		return false
	}

	var result JobCache
	if err := json.Unmarshal([]byte(statusData), &result); err != nil {
		log.Printf("Error unmarshaling job status for %s: %v", jobID, err)
		return false
	}

	// Skip if status is pending or success
	return result.Status == StatusPending || result.Status == StatusSuccess
}

// enqueueJob adds a job to the Redis queue
func (js *JobScheduler) enqueueJob(job Job) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return js.redis.LPush(js.ctx, js.queueName, jobData).Err()
}

// executeJobs continuously processes jobs from the queue
func (js *JobScheduler) executeJobs() {
	defer js.wg.Done()

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
				log.Printf("Error popping from queue: %v", err)
				continue
			}

			// Parse the job
			var job Job
			if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
				log.Printf("Error unmarshaling job: %v", err)
				continue
			}

			// Execute the job
			js.processJob(job)
		}
	}
}

// processJob executes a single job and updates its status
func (js *JobScheduler) processJob(job Job) {
	log.Printf("Processing job: %s (type: %s)", job.ID, job.Type)

	// Simulate job execution
	success, message := js.executeJobLogic(job)

	var status JobStatus
	if success {
		status = StatusSuccess
		log.Printf("Job %s completed successfully", job.ID)
	} else {
		status = StatusFailed
		log.Printf("Job %s failed: %s", job.ID, message)
	}

	// Update job status in cache
	js.setJobStatus(job.ID, status, message)
}

// executeJobLogic simulates the actual job execution logic
func (js *JobScheduler) executeJobLogic(job Job) (bool, string) {
	// Simulate processing time
	time.Sleep(time.Duration(100+job.ID[0]%5) * time.Millisecond)

	// Simulate success/failure based on job type
	switch job.Type {
	case "email":
		return true, "Email sent successfully"
	case "report":
		// Simulate occasional failure
		if time.Now().Unix()%7 == 0 {
			return false, "Report generation failed: database timeout"
		}
		return true, "Report generated successfully"
	case "cleanup":
		return true, "Cleanup completed"
	default:
		return false, "Unknown job type"
	}
}

// setJobStatus updates the job status in Redis cache
func (js *JobScheduler) setJobStatus(jobID string, status JobStatus, message string) {
	statusKey := fmt.Sprintf("%s:%s", js.statusCache, jobID)
	result := JobCache{
		JobID:     jobID,
		Status:    status,
		Message:   message,
		UpdatedAt: time.Now(),
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error marshaling job result for %s: %v", jobID, err)
		return
	}

	// Set with expiration (e.g., 24 hours)
	err = js.redis.Set(js.ctx, statusKey, resultData, 24*time.Hour).Err()
	if err != nil {
		log.Printf("Error setting job status for %s: %v", jobID, err)
	}
}

// updateJobStatuses updates the status of already executed jobs
func (js *JobScheduler) updateJobStatuses() {
	// Get all status keys
	pattern := fmt.Sprintf("%s:*", js.statusCache)
	keys, err := js.redis.Keys(js.ctx, pattern).Result()
	if err != nil {
		log.Printf("Error getting status keys: %v", err)
		return
	}

	// Check each job status and perform updates if needed
	for _, key := range keys {
		js.checkJobStatusUpdate(key)
	}
}

// checkJobStatusUpdate checks if a job status needs updating
func (js *JobScheduler) checkJobStatusUpdate(statusKey string) {
	statusData, err := js.redis.Get(js.ctx, statusKey).Result()
	if err != nil {
		if err != redis.Nil {
			log.Printf("Error getting job status from %s: %v", statusKey, err)
		}
		return
	}

	var result JobCache
	if err := json.Unmarshal([]byte(statusData), &result); err != nil {
		log.Printf("Error unmarshaling job status from %s: %v", statusKey, err)
		return
	}

	// Example: Update failed jobs older than 5 minutes to allow retry
	if result.Status == StatusFailed && time.Since(result.UpdatedAt) > 5*time.Minute {
		log.Printf("Clearing failed status for job %s to allow retry", result.JobID)
		js.redis.Del(js.ctx, statusKey)
	}
}

// getJobsToCheck simulates fetching jobs from a database or external source
func (js *JobScheduler) getJobsToCheck() []Job {
	// This would typically query your database for jobs that need to be executed
	// based on their scheduled time, status, etc.
	now := time.Now()

	return []Job{
		{
			ID:          fmt.Sprintf("job_%d_1", now.Unix()),
			Type:        "email",
			Payload:     map[string]string{"recipient": "user@example.com", "template": "welcome"},
			ScheduledAt: now,
			CreatedAt:   now.Add(-1 * time.Minute),
		},
		{
			ID:          fmt.Sprintf("job_%d_2", now.Unix()),
			Type:        "report",
			Payload:     map[string]string{"report_type": "daily_summary", "user_id": "123"},
			ScheduledAt: now,
			CreatedAt:   now.Add(-2 * time.Minute),
		},
		{
			ID:          fmt.Sprintf("job_%d_3", now.Unix()),
			Type:        "cleanup",
			Payload:     map[string]string{"resource": "temp_files", "older_than": "24h"},
			ScheduledAt: now,
			CreatedAt:   now.Add(-30 * time.Second),
		},
	}
}

// GetJobCache retrieves the current status of a job
func (js *JobScheduler) GetJobCache(jobID string) (*JobCache, error) {
	statusKey := fmt.Sprintf("%s:%s", js.statusCache, jobID)
	statusData, err := js.redis.Get(js.ctx, statusKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("job status not found")
		}
		return nil, err
	}

	var result JobCache
	err = json.Unmarshal([]byte(statusData), &result)
	return &result, err
}

func main() {
	// Initialize the job scheduler
	scheduler := NewJobScheduler("localhost:6379", "job_queue")

	// Start the scheduler
	scheduler.Start()

	// Run for demo purposes
	time.Sleep(2 * time.Minute)

	// Gracefully shutdown
	scheduler.Stop()
	log.Println("Job scheduler stopped")
}
