package audit

import (
	"context"

	auditdomain "admin-service/internal/domain/audit"
	"admin-service/pkg/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func AuditRequestContext(c *gin.Context) context.Context {
	return auditdomain.WithRequestMetadata(c.Request.Context(), auditdomain.RequestMetadata{
		IP:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	})
}

func AuditActorID(c *gin.Context) *uuid.UUID {
	user, ok := middleware.AuthUserFromContext(c)
	if !ok {
		return nil
	}
	actorID := user.ID
	return &actorID
}
