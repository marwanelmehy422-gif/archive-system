package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"archive-system/internal/services"
)

const (
	ContextUserID   = "user_id"
	ContextUsername = "username"
	ContextOrgID    = "org_id"
	ContextRole     = "role"
)

// AuthMiddleware - بيتأكد من الـ JWT token في كل request
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// جيب الـ token من الـ header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			return
		}

		// الـ header المفروض يكون: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header format must be: Bearer <token>",
			})
			return
		}

		// تحقق من الـ token
		claims, err := authService.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			return
		}

		// حط بيانات اليوزر في الـ context عشان الـ handlers تستخدمها
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUsername, claims.Username)
		c.Set(ContextOrgID, claims.OrgID)
		c.Set(ContextRole, claims.Role)

		c.Next()
	}
}

// GetCurrentUserID - helper لجيب الـ user ID من الـ context
func GetCurrentUserID(c *gin.Context) string {
	return c.GetString(ContextUserID)
}

// GetCurrentOrgID - helper لجيب الـ org ID من الـ context
func GetCurrentOrgID(c *gin.Context) string {
	return c.GetString(ContextOrgID)
}
