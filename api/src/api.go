// Package api implements the API routes and groups
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/diogovalentte/manga-dashboard-api/api/src/routes"
)

// SetupRouter sets up the routes for the API
func SetupRouter() *gin.Engine {
	router := gin.Default()

	v1 := router.Group("/v1")
	// Health check route
	{
		routes.HealthCheckRoute(v1)
	}
	{
		routes.MangaRoutes(v1)
	}

	return router
}
