package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ethaccount/backend/erc4337"
	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/service"
	"github.com/ethereum/go-ethereum/common"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

const TimeFormat = "2006-01-02 15:04:05"

type JobHandler struct {
	jobService *service.JobService
}

func NewJobHandler(jobService *service.JobService) *JobHandler {
	return &JobHandler{
		jobService: jobService,
	}
}

func (s *JobHandler) logger(ctx context.Context) *zerolog.Logger {
	l := zerolog.Ctx(ctx).With().Str("handler", "job").Logger()
	return &l
}

// RegisterJobRequest represents the request payload for job registration
type RegisterJobRequest struct {
	AccountAddress string                 `json:"accountAddress" binding:"required" example:"0x1234567890123456789012345678901234567890"`
	ChainID        int64                  `json:"chainId" binding:"required" example:"11155111"`
	JobID          int64                  `json:"jobId" binding:"required" example:"1"`
	UserOperation  *erc4337.UserOperation `json:"userOperation" binding:"required"`
	EntryPoint     string                 `json:"entryPoint" binding:"required" example:"0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"`
}

// RegisterJobResponse represents the response for job registration
type RegisterJobResponse struct {
	JobUUID        string `json:"jobUuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountAddress string `json:"accountAddress" example:"0x1234567890123456789012345678901234567890"`
	ChainID        int64  `json:"chainId" example:"11155111"`
	JobID          int64  `json:"jobId" example:"1"`
	EntryPoint     string `json:"entryPoint" example:"0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"`
	CreatedAt      string `json:"createdAt" example:"2025-01-09 13:36:56"`
	UpdatedAt      string `json:"updatedAt" example:"2025-01-09 13:36:56"`
	Message        string `json:"message" example:"Job registered successfully"`
}

// JobResponse represents a job in API responses
type JobResponse struct {
	ID                string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountAddress    string          `json:"accountAddress" example:"0x1234567890123456789012345678901234567890"`
	ChainID           int64           `json:"chainId" example:"11155111"`
	OnChainJobID      int64           `json:"onChainJobId" example:"1"`
	UserOperation     json.RawMessage `json:"userOperation"`
	EntryPointAddress string          `json:"entryPointAddress" example:"0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"`
	CreatedAt         string          `json:"createdAt" example:"2025-01-09 13:36:56"`
	UpdatedAt         string          `json:"updatedAt" example:"2025-01-09 13:36:56"`
}

// toJobResponse converts a domain Job to a JobResponse with formatted time fields
func toJobResponse(job *domain.EntityJob) JobResponse {
	return JobResponse{
		ID:                job.ID.String(),
		AccountAddress:    job.AccountAddress.Hex(),
		ChainID:           job.ChainID,
		OnChainJobID:      job.OnChainJobID,
		UserOperation:     job.UserOperation,
		EntryPointAddress: job.EntryPointAddress.Hex(),
		CreatedAt:         job.CreatedAt.Format(TimeFormat),
		UpdatedAt:         job.UpdatedAt.Format(TimeFormat),
	}
}

// RegisterJob godoc
// @Summary Register a new job
// @Description Register a new job with user operation for smart account management
// @Tags jobs
// @Accept json
// @Produce json
// @Param request body RegisterJobRequest true "Job registration request"
// @Success 201 {object} StandardResponse
// @Failure 400 {object} StandardResponse
// @Failure 500 {object} StandardResponse
// @Router /jobs [post]
func (h *JobHandler) RegisterJob(c *gin.Context) {
	logger := h.logger(c.Request.Context()).With().Str("function", "RegisterJob").Logger()

	var req RegisterJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error().Err(err).Msg("invalid request payload")
		respondWithError(c, domain.NewError(domain.ErrorCodeParameterInvalid, err, domain.WithMsg("Invalid request payload")))
		return
	}

	// Validate required fields
	if req.AccountAddress == "" || req.EntryPoint == "" || req.UserOperation == nil {
		logger.Error().Msg("missing required fields")
		respondWithError(c, domain.NewError(domain.ErrorCodeParameterInvalid, errors.New("missing required fields"), domain.WithMsg("accountAddress, entryPoint, and userOperation are required")))
		return
	}

	// Validate and parse addresses
	if !common.IsHexAddress(req.AccountAddress) {
		logger.Error().Str("accountAddress", req.AccountAddress).Msg("invalid account address format")
		respondWithError(c, domain.NewError(domain.ErrorCodeParameterInvalid, errors.New("invalid account address format"), domain.WithMsg("accountAddress must be a valid hex address")))
		return
	}

	if !common.IsHexAddress(req.EntryPoint) {
		logger.Error().Str("entryPoint", req.EntryPoint).Msg("invalid entry point address format")
		respondWithError(c, domain.NewError(domain.ErrorCodeParameterInvalid, errors.New("invalid entry point address format"), domain.WithMsg("entryPoint must be a valid hex address")))
		return
	}

	accountAddress := common.HexToAddress(req.AccountAddress)
	entryPointAddress := common.HexToAddress(req.EntryPoint)

	job, err := h.jobService.RegisterJob(
		c.Request.Context(),
		accountAddress,
		req.ChainID,
		req.JobID,
		req.UserOperation,
		entryPointAddress,
	)
	if err != nil {
		respondWithError(c, err)
		return
	}

	response := RegisterJobResponse{
		JobUUID:        job.ID.String(),
		AccountAddress: job.AccountAddress.Hex(),
		ChainID:        job.ChainID,
		JobID:          job.OnChainJobID,
		EntryPoint:     job.EntryPointAddress.Hex(),
		CreatedAt:      job.CreatedAt.Format(TimeFormat),
		UpdatedAt:      job.UpdatedAt.Format(TimeFormat),
		Message:        "Job registered successfully",
	}

	logger.Info().
		Str("job_uuid", job.ID.String()).
		Str("account_address", req.AccountAddress).
		Int64("chain_id", req.ChainID).
		Int64("job_id", req.JobID).
		Msg("job registered successfully")

	respondWithSuccessAndStatus(c, http.StatusCreated, response, "Job registered successfully")
}

// GetJobList godoc
// @Summary Get all active jobs
// @Description Retrieve a list of all active jobs in the system
// @Tags jobs
// @Accept json
// @Produce json
// @Success 200 {object} StandardResponse
// @Failure 500 {object} StandardResponse
// @Router /jobs [get]
func (h *JobHandler) GetJobList(c *gin.Context) {
	logger := h.logger(c.Request.Context()).With().Str("function", "GetJobList").Logger()

	jobs, err := h.jobService.GetActiveJobs(c.Request.Context())
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve jobs")
		respondWithError(c, err)
		return
	}

	// Convert domain jobs to response DTOs with formatted time fields
	jobResponses := make([]JobResponse, len(jobs))
	for i, job := range jobs {
		jobResponses[i] = toJobResponse(job)
	}

	logger.Info().
		Int("job_count", len(jobs)).
		Msg("job list retrieved successfully")

	respondWithSuccess(c, jobResponses)
}
