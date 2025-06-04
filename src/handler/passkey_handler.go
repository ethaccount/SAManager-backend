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

// RegisterBegin godoc
// @Summary Begin passkey registration
// @Description Start the WebAuthn registration process for a new passkey
// @Tags passkey
// @Accept json
// @Produce json
// @Param request body RegisterBeginRequest true "Registration request"
// @Success 201 {object} StandardResponse
// @Failure 400 {object} StandardResponse
// @Failure 500 {object} StandardResponse
// @Router /register/begin [post]
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

		respondWithSuccessAndStatus(c, http.StatusCreated, resp, "Registration options created successfully")
	}

}

// RegisterBeginRequest represents the request for beginning passkey registration
type RegisterBeginRequest struct {
	Username string `json:"username" binding:"required" example:"user@example.com"`
}
