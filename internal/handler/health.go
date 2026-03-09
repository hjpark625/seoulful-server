package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"seoulful-server-go/internal/model"
)

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{
		Status:    "UP",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
