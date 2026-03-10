package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	PermissionUsersRead   = "users:read"
	PermissionUsersWrite  = "users:write"
	PermissionUsersDelete = "users:delete"
)

func RequirePermission(required string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := AuthUserFromContext(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "permission_denied",
				"message": "permission denied",
			})
			return
		}

		for _, perm := range user.Permissions {
			if perm == required {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code":    "permission_denied",
			"message": "permission denied",
		})
	}
}
