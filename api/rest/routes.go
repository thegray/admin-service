package rest

import (
	"admin-service/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {

	// global middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// system routes
	r.GET("/health", h.HealthCheck)

	authGroup := r.Group("/auth")
	if h.rateLimiter != nil {
		authGroup.Use(h.rateLimiter)
	}
	authGroup.POST("/login", h.Login)
	authGroup.POST("/refresh", h.Refresh)
	authGroup.POST("/logout", h.Logout)

	v1 := r.Group("/api/v1")
	if h.authMiddleware != nil {
		v1.Use(h.authMiddleware)
	}

	h.initExampleRoutes(v1)
	h.initUserRoutes(v1)
}

func (h *Handler) initExampleRoutes(rg *gin.RouterGroup) {
	example := rg.Group("/examples")

	if h.rateLimiter != nil {
		example.Use(h.rateLimiter)
	}

	example.POST("/", h.ExamplePost)
	example.GET("/:id", h.ExampleGet)
}

func (h *Handler) initUserRoutes(rg *gin.RouterGroup) {
	usersGroup := rg.Group("/users")

	if h.rateLimiter != nil {
		usersGroup.Use(h.rateLimiter)
	}

	usersGroup.GET("/", middleware.RequirePermission(middleware.PermissionUsersRead), h.ListUsers)
	usersGroup.GET("/:id", middleware.RequirePermission(middleware.PermissionUsersRead), h.GetUser)
	usersGroup.POST("/", middleware.RequirePermission(middleware.PermissionUsersWrite), h.CreateUser)
	usersGroup.PUT("/:id", middleware.RequirePermission(middleware.PermissionUsersWrite), h.UpdateUser)
	usersGroup.DELETE("/:id", middleware.RequirePermission(middleware.PermissionUsersDelete), h.DeleteUser)
}
