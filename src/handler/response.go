package handler

import (
	"errors"
	"net/http"

	"github.com/ethaccount/backend/src/domain"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// StandardResponse represents the standard API response format
type StandardResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// ErrorDetail represents detailed error information
type ErrorDetail struct {
	Field  string `json:"field,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// Legacy error message format (kept for backward compatibility)
type ErrorMessage struct {
	Name       string                 `json:"name"`
	Code       int                    `json:"code"`
	Message    string                 `json:"message,omitempty"`
	RemoteCode int                    `json:"remoteCode,omitempty"`
	Detail     map[string]interface{} `json:"detail,omitempty"`
}

// respondWithSuccess sends a successful response with the standard format
func respondWithSuccess(c *gin.Context, data interface{}) {
	msg := "OK"

	response := StandardResponse{
		Code:    0,
		Message: msg,
		Data:    data,
	}

	c.JSON(http.StatusOK, response)
}

// respondWithSuccessAndStatus sends a successful response with custom HTTP status
func respondWithSuccessAndStatus(c *gin.Context, httpStatus int, data interface{}, message ...string) {
	msg := "OK"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	response := StandardResponse{
		Code:    0,
		Message: msg,
		Data:    data,
	}

	c.JSON(httpStatus, response)
}

// respondWithError sends an error response with the standard format
func respondWithError(c *gin.Context, err error) {
	domainErr := parseDomainError(err)

	// Use the original error message if the domain error has no client message
	message := domainErr.ClientMsg()
	if message == "" {
		message = err.Error()
	}

	response := StandardResponse{
		Code:    mapDomainErrorToCode(domainErr),
		Message: message,
	}

	// Add error details if available
	if detail := domainErr.Detail(); detail != nil {
		response.Error = detail
	}

	ctx := c.Request.Context()
	zerolog.Ctx(ctx).Error().
		Str("function", "respondWithError").
		Int("error_code", response.Code).
		Msg(response.Message)

	_ = c.Error(err)
	c.AbortWithStatusJSON(domainErr.HTTPStatus(), response)
}

// respondWithCustomError sends a custom error response
func respondWithCustomError(c *gin.Context, httpStatus int, code int, message string, errorDetail interface{}) {
	response := StandardResponse{
		Code:    code,
		Message: message,
		Error:   errorDetail,
	}

	ctx := c.Request.Context()
	zerolog.Ctx(ctx).Error().
		Str("function", "respondWithCustomError").
		Int("error_code", code).
		Msg(message)

	c.AbortWithStatusJSON(httpStatus, response)
}

// parseDomainError extracts domain error information
func parseDomainError(err error) domain.DomainError {
	var domainError domain.DomainError
	// We don't check if errors.As is valid or not
	// because an empty domain.DomainError would return default error data.
	_ = errors.As(err, &domainError)
	return domainError
}

// mapDomainErrorToCode maps domain error codes to API response codes
func mapDomainErrorToCode(domainErr domain.DomainError) int {
	switch domainErr.Name() {
	case "PARAMETER_INVALID":
		return 1001
	case "RESOURCE_NOT_FOUND":
		return 1002
	case "AUTH_PERMISSION_DENIED":
		return 1003
	case "AUTH_NOT_AUTHENTICATED":
		return 1004
	case "INTERNAL_PROCESS":
		return 1005
	case "REMOTE_PROCESS_ERROR":
		return 1006
	default:
		return 1000 // Generic error code
	}
}
