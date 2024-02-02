// Package routes implements the health check route
package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheckRoute registers the health check route
func HealthCheckRoute(group *gin.RouterGroup) {
	group.GET("/health", healthCheck)
}

func healthCheck(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}
