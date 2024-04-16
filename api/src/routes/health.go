package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheckRoute registers the health check route
func HealthCheckRoute(group *gin.RouterGroup) {
	group.GET("/health", healthCheck)
}

// @Summary Health check route
// @Description Returns status OK
// @Success 200 {string} string OK
// @Produce plain
// @Router /health [get]
func healthCheck(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}
