package rest

import (
	"net/http"
	"time"

	"admin-service/internal/domain/example"
	"admin-service/internal/domain/users"
	svcerrors "admin-service/pkg/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler struct {
	exampleSvc  *example.Service
	userSvc     *users.Service
	log         *zap.Logger
	rateLimiter gin.HandlerFunc
}

func NewHandler(
	exampleSvc *example.Service,
	userSvc *users.Service,
	logger *zap.Logger,
	rateLimiter gin.HandlerFunc,
) *Handler {
	return &Handler{
		exampleSvc:  exampleSvc,
		userSvc:     userSvc,
		log:         logger.Named("admin-api"),
		rateLimiter: rateLimiter,
	}
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) ExamplePost(c *gin.Context) {
	ctx := c.Request.Context()
	var req createExampleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}
	ex, err := h.exampleSvc.Create(ctx, req.Param)
	if err != nil {
		respondWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, exampleResponse{
		ID:        ex.ID,
		Data:      ex.Message,
		CreatedAt: ex.CreatedAt,
	})
}

func (h *Handler) ExampleGet(c *gin.Context) {
	ctx := c.Request.Context()
	var req getExampleRequest
	if err := c.ShouldBindUri(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}
	ex, err := h.exampleSvc.GetByID(ctx, int64(req.ID))
	if err != nil {
		respondWithError(c, err)
		return
	}
	c.JSON(http.StatusOK, exampleResponse{
		ID:        ex.ID,
		Data:      ex.Message,
		CreatedAt: ex.CreatedAt,
	})
}
