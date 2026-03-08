package rest

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {

	// global middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// system routes
	r.GET("/health", h.HealthCheck)

	v1 := r.Group("/api/v1")

	h.initExampleRoutes(v1)
}

func (h *Handler) initExampleRoutes(rg *gin.RouterGroup) {
	example := rg.Group("/examples")

	if h.rateLimiter != nil {
		example.Use(h.rateLimiter)
	}

	example.POST("/", h.ExamplePost)
	example.GET("/:id", h.ExampleGet)
}
