package middleware

import (
	"context"
	"strings"

	authdomain "admin-service/internal/domain/auth"
	"admin-service/internal/domain/users"
	"admin-service/pkg/auth"
	svcerrors "admin-service/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const (
	authUserContextKey    contextKey = "auth_user"
	authRequestContextKey contextKey = "auth_request_user"
)

type AuthUser struct {
	ID          uuid.UUID
	Roles       []string
	Permissions []string
}

func AuthUserFromContext(c *gin.Context) (*AuthUser, bool) {
	val, ok := c.Get(string(authUserContextKey))
	if !ok {
		return nil, false
	}
	user, ok := val.(*AuthUser)
	return user, ok
}

func AuthMiddleware(tokenManager *auth.TokenManager, repo users.Repository, cache *authdomain.SessionCache, logger *zap.Logger) gin.HandlerFunc {
	log := logger
	if log == nil {
		log = zap.NewNop()
	}
	return func(c *gin.Context) {
		if tokenManager == nil || repo == nil || cache == nil {
			respondWithServiceError(c, svcerrors.ErrInternal)
			return
		}

		token := parseBearerHeader(c.GetHeader("Authorization"))
		if token == "" {
			respondWithServiceError(c, svcerrors.ErrUnauthorized)
			return
		}

		claims, err := tokenManager.ParseAccessToken(c.Request.Context(), token)
		if err != nil {
			log.Debug("invalid access token", zap.Error(err))
			respondWithServiceError(c, svcerrors.ErrUnauthorized)
			return
		}

		session, err := authdomain.LoadUserSession(c.Request.Context(), repo, cache, log, claims.UserID)
		if err != nil {
			log.Error("failed to load session", zap.Error(err))
			respondWithServiceError(c, svcerrors.ErrInternal)
			return
		}
		if session == nil || session.IsBanned() || session.TokenVersion != claims.TokenVersion {
			respondWithServiceError(c, svcerrors.ErrUnauthorized)
			return
		}

		user := &AuthUser{
			ID:          claims.UserID,
			Roles:       session.Roles,
			Permissions: session.Permissions,
		}
		c.Set(string(authUserContextKey), user)
		ctx := context.WithValue(c.Request.Context(), authRequestContextKey, user)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func parseBearerHeader(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

func respondWithServiceError(c *gin.Context, err svcerrors.ServiceError) {
	c.AbortWithStatusJSON(err.Status(), gin.H{
		"code":    err.Code(),
		"message": err.Message(),
	})
}
