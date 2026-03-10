package rest

import (
	"net/http"

	"admin-service/pkg/audit"
	svcerrors "admin-service/pkg/errors"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type authResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	ctx := audit.AuditRequestContext(c)
	tokens, _, err := h.authSvc.Login(ctx, req.Email, req.Password)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, authResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	ctx := audit.AuditRequestContext(c)
	tokens, _, err := h.authSvc.Refresh(ctx, req.RefreshToken)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, authResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, svcerrors.ErrInvalidPayload)
		return
	}

	ctx := audit.AuditRequestContext(c)
	_, err := h.authSvc.Logout(ctx, req.RefreshToken)
	if err != nil {
		respondWithError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
