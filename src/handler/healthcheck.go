package handler

import (
	"github.com/gin-gonic/gin"
)

// HealthCheck godoc
// @Summary Health check endpoint
// @Description Check if the service is running
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} StandardResponse
// @Router /health [get]
func HandleHealthCheck(c *gin.Context) {
	data := map[string]string{"status": "healthy"}
	respondWithSuccess(c, data, "OK")
}
