package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// CacheJobStatus represents the execution status of a job in cache
type CacheJobStatus string

const (
	CacheStatusPending   CacheJobStatus = "pending"
	CacheStatusFailed    CacheJobStatus = "failed"
	CacheStatusCompleted CacheJobStatus = "completed"
)

// JobCache contains the execution result
type JobCache struct {
	JobID      uuid.UUID      `json:"job_id"`
	ChainID    int64          `json:"chain_id"`
	UserOpHash common.Hash    `json:"user_op_hash"`
	Status     CacheJobStatus `json:"status"`
	Error      string         `json:"error"`
	UpdatedAt  time.Time      `json:"updated_at"`
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

// SetJobStatus updates the job status in Redis cache with 24-hour expiration
func (r *JobCacheRepository) SetJobStatus(ctx context.Context, jobID uuid.UUID, status CacheJobStatus, message *string) error {
	statusKey := fmt.Sprintf("%s:%s", r.statusCache, jobID)
	result := JobCache{
		JobID:     jobID,
		Status:    status,
		Error:     *message,
		UpdatedAt: time.Now(),
	}

	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal job result: %w", err)
	}

	return r.redis.Set(ctx, statusKey, resultData, 24*time.Hour).Err()
}

// DeleteJobCache removes the JobCache by jobID from Redis
func (r *JobCacheRepository) DeleteJobCache(ctx context.Context, jobID uuid.UUID) error {
	statusKey := fmt.Sprintf("%s:%s", r.statusCache, jobID)
	return r.redis.Del(ctx, statusKey).Err()
}

// AddJobCache stores a complete JobCache object in Redis with 24-hour expiration
func (r *JobCacheRepository) AddJobCache(ctx context.Context, jobCache *JobCache) error {
	statusKey := fmt.Sprintf("%s:%s", r.statusCache, jobCache.JobID)

	// Update the timestamp
	jobCache.UpdatedAt = time.Now()

	jobData, err := json.Marshal(jobCache)
	if err != nil {
		return fmt.Errorf("failed to marshal job cache: %w", err)
	}

	// Set with 24-hour expiration
	return r.redis.Set(ctx, statusKey, jobData, 24*time.Hour).Err()
}

// GetAllStatusKeys retrieves all status keys matching the pattern
func (r *JobCacheRepository) GetAllStatusKeys(ctx context.Context) ([]string, error) {
	pattern := fmt.Sprintf("%s:*", r.statusCache)
	return r.redis.Keys(ctx, pattern).Result()
}

// GetJobCachesByStatus retrieves all job caches with the specified status
func (r *JobCacheRepository) GetJobCachesByStatus(ctx context.Context, status CacheJobStatus) ([]*JobCache, error) {
	keys, err := r.GetAllStatusKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get status keys: %w", err)
	}

	var jobCaches []*JobCache
	for _, key := range keys {
		statusData, err := r.redis.Get(ctx, key).Result()
		if err != nil {
			// Skip keys that no longer exist (expired or deleted)
			if err == redis.Nil {
				continue
			}
			return nil, fmt.Errorf("failed to get job cache for key %s: %w", key, err)
		}

		var jobCache JobCache
		if err := json.Unmarshal([]byte(statusData), &jobCache); err != nil {
			return nil, fmt.Errorf("failed to unmarshal job cache for key %s: %w", key, err)
		}

		if jobCache.Status == status {
			jobCaches = append(jobCaches, &jobCache)
		}
	}

	return jobCaches, nil
}
