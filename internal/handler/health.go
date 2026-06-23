package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"seoulful-server-go/internal/model"
)

func HealthCheck(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			log.Printf("health check PostgreSQL ping failed: %v", err)
			c.JSON(http.StatusServiceUnavailable, model.HealthResponse{
				Status:    "DOWN",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			return
		}

		c.JSON(http.StatusOK, model.HealthResponse{
			Status:    "UP",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}
