package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethaccount/backend/src/domain"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// JobStatus represents the execution status of a job
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusFailed    JobStatus = "failed"
	StatusCompleted JobStatus = "completed"
)

// JobCache contains the execution result
type JobCache struct {
	JobID     uuid.UUID `json:"job_id"`
	Status    JobStatus `json:"status"`
	Error     string    `json:"error"`
	UpdatedAt time.Time `json:"updated_at"`
}

// JobCacheRepository handles Redis operations for job scheduling and status management
type JobCacheRepository struct {
	redis       *redis.Client
	queueName   string
	statusCache string
}

// NewJobCacheRepository creates a new job cache repository instance
func NewJobCacheRepository(redis *redis.Client, queueName string) *JobCacheRepository {
	return &JobCacheRepository{
		redis:       redis,
		queueName:   queueName,
		statusCache: queueName + ":status",
	}
}

// EnqueueJob adds a job to the Redis queue
func (r *JobCacheRepository) EnqueueJob(ctx context.Context, job domain.JobModel) error {
	jobData, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	return r.redis.LPush(ctx, r.queueName, jobData).Err()
}

// DequeueJob pops a job from the Redis queue
func (r *JobCacheRepository) DequeueJob(ctx context.Context, timeout time.Duration) (*domain.JobModel, error) {
	result, err := r.redis.BRPop(ctx, timeout, r.queueName).Result()
	if err != nil {
		return nil, err
	}

	var job domain.JobModel
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job: %w", err)
	}

	return &job, nil
}

// GetJobStatus retrieves the job status from Redis cache
func (r *JobCacheRepository) GetJobStatus(ctx context.Context, jobID uuid.UUID) (*JobCache, error) {
	statusKey := fmt.Sprintf("%s:%s", r.statusCache, jobID)
	statusData, err := r.redis.Get(ctx, statusKey).Result()
	if err != nil {
		return nil, err
	}

	var result JobCache
	if err := json.Unmarshal([]byte(statusData), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job status: %w", err)
	}

	return &result, nil
}

// SetJobStatus updates the job status in Redis cache with expiration
func (r *JobCacheRepository) SetJobStatus(ctx context.Context, jobID uuid.UUID, status JobStatus, message string, expiration time.Duration) error {
	statusKey := fmt.Sprintf("%s:%s", r.statusCache, jobID)
	result := JobCache{
		JobID:     jobID,
		Status:    status,
		Error:     message,
		UpdatedAt: time.Now(),
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal job result: %w", err)
	}

	return r.redis.Set(ctx, statusKey, resultData, expiration).Err()
}

// DeleteJobStatus removes the job status from Redis cache
func (r *JobCacheRepository) DeleteJobStatus(ctx context.Context, jobID uuid.UUID) error {
	statusKey := fmt.Sprintf("%s:%s", r.statusCache, jobID)
	return r.redis.Del(ctx, statusKey).Err()
}

// GetAllStatusKeys retrieves all status keys matching the pattern
func (r *JobCacheRepository) GetAllStatusKeys(ctx context.Context) ([]string, error) {
	pattern := fmt.Sprintf("%s:*", r.statusCache)
	return r.redis.Keys(ctx, pattern).Result()
}
