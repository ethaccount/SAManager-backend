package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

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
	AccountAddress string                `json:"accountAddress" binding:"required" example:"0x1234567890123456789012345678901234567890"`
	ChainID        int64                 `json:"chainId" binding:"required" example:"11155111"`
	JobID          int64                 `json:"jobId" binding:"required" example:"1"`
	UserOperation  *domain.UserOperation `json:"userOperation" binding:"required"`
	EntryPoint     string                `json:"entryPoint" binding:"required" example:"0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"`
}

// RegisterJobResponse represents the response for job registration
type RegisterJobResponse struct {
	JobUUID        string `json:"jobUuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountAddress string `json:"accountAddress" example:"0x1234567890123456789012345678901234567890"`
	ChainID        int64  `json:"chainId" example:"11155111"`
	JobID          int64  `json:"jobId" example:"1"`
	EntryPoint     string `json:"entryPoint" example:"0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"`
	Message        string `json:"message" example:"Job registered successfully"`
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
	logger := h.logger(c.Request.Context()).With().Str("func", "RegisterJob").Logger()

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

	job, err := h.jobService.RegisterJob(
		c.Request.Context(),
		req.AccountAddress,
		req.ChainID,
		req.JobID,
		req.UserOperation,
		req.EntryPoint,
	)
	if err != nil {
		logger.Error().Err(err).Msg("failed to register job")
		respondWithError(c, err)
		return
	}

	response := RegisterJobResponse{
		JobUUID:        job.ID.String(),
		AccountAddress: job.AccountAddress,
		ChainID:        job.ChainID,
		JobID:          job.OnChainJobID,
		EntryPoint:     job.EntryPointAddress,
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
