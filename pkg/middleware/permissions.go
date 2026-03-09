package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	PermissionUsersRead   = "users:read"
	PermissionUsersWrite  = "users:write"
	PermissionUsersDelete = "users:delete"
)

func RequirePermission(required string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("X-User-Permissions") // temp, will be replaced by jwt
		if header == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "permission_denied",
				"message": "missing permission grant",
			})
			return
		}

		perms := strings.Split(header, ",")
		for _, p := range perms {
			if strings.TrimSpace(p) == required {
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
