package handler

import (
	"net/http"

	"github.com/ethaccount/backend/src/domain"
	"github.com/ethaccount/backend/src/service"
	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
)

type PasskeyHandler struct {
	service *service.PasskeyService
}

func NewPasskeyHandler(service *service.PasskeyService) *PasskeyHandler {
	return &PasskeyHandler{
		service: service,
	}
}

func (h *PasskeyHandler) RegisterBegin() gin.HandlerFunc {
	type Body struct {
		Username string `json:"username" binding:"required"`
	}

	type Response struct {
		Options *protocol.CredentialCreation `json:"options"`
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		var body Body
		err := c.ShouldBind(&body)
		if err != nil {
			respondWithError(c, domain.NewError(domain.ErrorCodeParameterInvalid, err, domain.WithMsg("invalid parameter")))
			return
		}

		options, _, err := h.service.BeginRegistration(ctx, body.Username)
		if err != nil {
			respondWithError(c, err)
			return
		}

		resp := Response{
			Options: options,
		}

		c.JSON(http.StatusCreated, resp)
	}

}
