package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthStatus struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

func HealthCheck(db *gorm.DB, redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := HealthStatus{
			Status:   "healthy",
			Services: make(map[string]string),
		}

		// Check PostgreSQL
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			status.Services["postgresql"] = "unhealthy"
			status.Status = "unhealthy"
		} else {
			status.Services["postgresql"] = "healthy"
		}

		// Check Redis
		ctx := context.Background()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			status.Services["redis"] = "unhealthy"
			status.Status = "unhealthy"
		} else {
			status.Services["redis"] = "healthy"
		}

		statusCode := http.StatusOK
		if status.Status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, status)
	}
}
