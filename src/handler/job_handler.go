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
	AccountAddress string                `json:"accountAddress" binding:"required"`
	ChainID        int64                 `json:"chainId" binding:"required"`
	JobID          int64                 `json:"jobId" binding:"required"`
	UserOperation  *domain.UserOperation `json:"userOperation" binding:"required"`
	EntryPoint     string                `json:"entryPoint" binding:"required"`
}

// RegisterJobResponse represents the response for job registration
type RegisterJobResponse struct {
	JobUUID        string `json:"jobUuid"`
	AccountAddress string `json:"accountAddress"`
	ChainID        int64  `json:"chainId"`
	JobID          int64  `json:"jobId"`
	EntryPoint     string `json:"entryPoint"`
	Message        string `json:"message"`
}

// RegisterJob handles POST /jobs endpoint for job registration
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

	c.JSON(http.StatusCreated, response)
}
