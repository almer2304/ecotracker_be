package middleware

import (
	"net/http"
	"strings"

	"ecotracker/internal/utils"

	"github.com/gin-gonic/gin"
)

const (
	ContextUserID = "userID"
	ContextEmail  = "userEmail"
	ContextRole   = "userRole"
)

func AuthMiddleware(jwtUtil *utils.JWTUtil) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid authorization format. Use: Bearer <token>",
			})
			return
		}

		claims, err := jwtUtil.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired token",
			})
			return
		}

		// Set user info in context
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

// RequireRole restricts endpoint to specific roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(ContextRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "No role found in context",
			})
			return
		}

		role, ok := userRole.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Invalid role type in context",
			})
			return
		}

		for _, allowed := range roles {
			if role == allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Forbidden: insufficient permissions",
		})
	}
}

// GetUserID is a helper to extract userID from Gin context
func GetUserID(c *gin.Context) string {
	id, _ := c.Get(ContextUserID)
	str, _ := id.(string)
	return str
}

// GetUserRole is a helper to extract role from Gin context
func GetUserRole(c *gin.Context) string {
	role, _ := c.Get(ContextRole)
	str, _ := role.(string)
	return str
}
