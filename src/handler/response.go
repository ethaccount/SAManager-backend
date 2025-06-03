package handler

import (
	"errors"

	"github.com/ethaccount/backend/src/domain"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type ErrorMessage struct {
	Name       string                 `json:"name"`
	Code       int                    `json:"code"`
	Message    string                 `json:"message,omitempty"`
	RemoteCode int                    `json:"remoteCode,omitempty"`
	Detail     map[string]interface{} `json:"detail,omitempty"`
}

func respondWithError(c *gin.Context, err error) {
	errMessage := parseError(err)
	ctx := c.Request.Context()
	zerolog.Ctx(ctx).Error().Str("func", "respondWithError").Msg(errMessage.Message)
	_ = c.Error(err)
	c.AbortWithStatusJSON(errMessage.Code, errMessage)
}

func parseError(err error) ErrorMessage {
	var domainError domain.DomainError
	// We don't check if errors.As is valid or not
	// because an empty common.DomainError would return default error data.
	_ = errors.As(err, &domainError)

	return ErrorMessage{
		Name:       domainError.Name(),
		Code:       domainError.HTTPStatus(),
		Message:    domainError.ClientMsg(),
		RemoteCode: domainError.RemoteHTTPStatus(),
		Detail:     domainError.Detail(),
	}
}
