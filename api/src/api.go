// Package api implements the API routes and groups
package api

import (
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	docs "github.com/diogovalentte/mantium/api/docs"
	"github.com/diogovalentte/mantium/api/src/routes"
)

// SetupRouter sets up the routes for the API
func SetupRouter() *gin.Engine {
	router := gin.Default()

	docs.SwaggerInfo.Title = "Mantium API"
	docs.SwaggerInfo.Description = "API for Mantium, a manga dashboard."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.BasePath = "/v1"

	v1 := router.Group("/v1")
	{
		routes.HealthCheckRoute(v1)
	}
	{
		routes.MangaRoutes(v1)
	}
	{
		routes.DashboardRoutes(v1)
	}

	v1.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	return router
}
