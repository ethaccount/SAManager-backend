package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
