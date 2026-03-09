package rest

import (
	"net/http"
	"time"

	"admin-service/internal/domain"
	"admin-service/internal/domain/users"
	svcerrors "admin-service/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type listUsersRequest struct {
	Limit  int `form:"limit" binding:"omitempty,min=1,max=500"`
	Offset int `form:"offset" binding:"omitempty,min=0"`
}

type createUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	IsActive *bool  `json:"is_active"`
}

type updateUserRequest struct {
	Email    *string `json:"email" binding:"omitempty,email"`
	Password *string `json:"password" binding:"omitempty,min=8"`
	IsActive *bool   `json:"is_active"`
}

type userURIRequest struct {
	ID uuid.UUID `uri:"id" binding:"required,uuid"`
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	IsActive  bool      `json:"is_active"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

func (h *Handler) ListUsers(c *gin.Context) {
	var req listUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}
	limit := req.Limit
	if limit == 0 {
		limit = 100
	}

	ctx := c.Request.Context()
	usersList, err := h.userSvc.List(ctx, limit, req.Offset)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": mapUsers(usersList),
	})
}

func (h *Handler) GetUser(c *gin.Context) {
	var req userURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	ctx := c.Request.Context()
	user, err := h.userSvc.GetByID(ctx, req.ID)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapUser(user))
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	ctx := c.Request.Context()
	user, err := h.userSvc.Create(ctx, users.CreateUserInput{
		Email:    req.Email,
		Password: req.Password,
		IsActive: isActive,
	})
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, mapUser(user))
}

func (h *Handler) UpdateUser(c *gin.Context) {
	var uriReq userURIRequest
	if err := c.ShouldBindUri(&uriReq); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	var body updateUserRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	ctx := c.Request.Context()
	user, err := h.userSvc.Update(ctx, uriReq.ID, users.UpdateUserInput{
		Email:    body.Email,
		Password: body.Password,
		IsActive: body.IsActive,
	})
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, mapUser(user))
}

func (h *Handler) DeleteUser(c *gin.Context) {
	var req userURIRequest
	if err := c.ShouldBindUri(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	ctx := c.Request.Context()
	if err := h.userSvc.Delete(ctx, req.ID); err != nil {
		respondWithError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func mapUser(u *domain.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Email:     u.Email,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}
}

func mapUsers(list []*domain.User) []userResponse {
	out := make([]userResponse, 0, len(list))
	for _, u := range list {
		out = append(out, mapUser(u))
	}
	return out
}
