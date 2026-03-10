package rest

import (
	"net/http"
	"time"

	domain "admin-service/internal/domain/model"
	"admin-service/internal/domain/threats"
	"admin-service/pkg/audit"
	svcerrors "admin-service/pkg/errors"
	"admin-service/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type listThreatsRequest struct {
	Limit  int `form:"limit" binding:"omitempty,min=1,max=500"`
	Offset int `form:"offset" binding:"omitempty,min=0"`
}

type createThreatRequest struct {
	Title       string  `json:"title" binding:"required"`
	Type        string  `json:"type" binding:"required"`
	Severity    string  `json:"severity" binding:"required"`
	Indicator   string  `json:"indicator" binding:"required"`
	Description *string `json:"description"`
}

type updateThreatRequest struct {
	Title       *string `json:"title"`
	Type        *string `json:"type"`
	Severity    *string `json:"severity"`
	Indicator   *string `json:"indicator"`
	Description *string `json:"description"`
}

type threatURIRequest struct {
	ID string `uri:"id" binding:"required"`
}

type threatResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Indicator   string    `json:"indicator"`
	Description string    `json:"description"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

func (h *Handler) ListThreats(c *gin.Context) {
	var req listThreatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	limit := req.Limit
	if limit == 0 {
		limit = 100
	}

	ctx := audit.AuditRequestContext(c)
	list, err := h.threatSvc.List(ctx, limit, req.Offset)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": mapThreats(list),
	})
}

func (h *Handler) GetThreat(c *gin.Context) {
	var uriReq threatURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	threatID, ok := utils.ParseID(c, uriReq.ID, h.log)
	if !ok {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	ctx := audit.AuditRequestContext(c)
	threat, err := h.threatSvc.GetByID(ctx, threatID)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapThreat(threat))
}

func (h *Handler) CreateThreat(c *gin.Context) {
	var req createThreatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	actorID := audit.AuditActorID(c)
	if actorID == nil {
		respondWithError(c, svcerrors.ErrUnauthorized)
		return
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	ctx := audit.AuditRequestContext(c)
	threat, err := h.threatSvc.Create(ctx, actorID, threats.CreateThreatInput{
		Title:       req.Title,
		Type:        req.Type,
		Severity:    req.Severity,
		Indicator:   req.Indicator,
		Description: description,
		CreatedBy:   *actorID,
	})
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, mapThreat(threat))
}

func (h *Handler) UpdateThreat(c *gin.Context) {
	var uriReq threatURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	threatID, ok := utils.ParseID(c, uriReq.ID, h.log)
	if !ok {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	var req updateThreatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	ctx := audit.AuditRequestContext(c)
	actorID := audit.AuditActorID(c)
	threat, err := h.threatSvc.Update(ctx, actorID, threatID, threats.UpdateThreatInput{
		Title:       req.Title,
		Type:        req.Type,
		Severity:    req.Severity,
		Indicator:   req.Indicator,
		Description: req.Description,
	})
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapThreat(threat))
}

func (h *Handler) DeleteThreat(c *gin.Context) {
	var uriReq threatURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	threatID, ok := utils.ParseID(c, uriReq.ID, h.log)
	if !ok {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	ctx := audit.AuditRequestContext(c)
	actorID := audit.AuditActorID(c)
	if err := h.threatSvc.Delete(ctx, actorID, threatID); err != nil {
		respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func mapThreat(t *domain.Threat) threatResponse {
	return threatResponse{
		ID:          t.ID,
		Title:       t.Title,
		Type:        t.Type,
		Severity:    t.Severity,
		Indicator:   t.Indicator,
		Description: t.Description,
		CreatedBy:   t.CreatedBy,
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
}

func mapThreats(list []*domain.Threat) []threatResponse {
	out := make([]threatResponse, 0, len(list))
	for _, t := range list {
		out = append(out, mapThreat(t))
	}
	return out
}
