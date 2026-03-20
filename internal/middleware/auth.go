package middleware

import (
	"net/http"
	"strings"

	"github.com/ecotracker/backend/internal/domain"
	"github.com/ecotracker/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

const (
	ContextUserID = "user_id"
	ContextEmail  = "user_email"
	ContextRole   = "user_role"
)

// AuthMiddleware memvalidasi JWT token dan menyimpan info user ke context
func AuthMiddleware(jwtManager *utils.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Token autentikasi diperlukan",
			})
			c.Abort()
			return
		}

		// Format: "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Format token tidak valid, gunakan: Bearer <token>",
			})
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Token tidak valid atau sudah kadaluarsa",
			})
			c.Abort()
			return
		}

		// Simpan info user ke context
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextRole, claims.Role)

		c.Next()
	}
}

// RequireRole middleware untuk membatasi akses berdasarkan role
func RequireRole(roles ...domain.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleStr, exists := c.Get(ContextRole)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Tidak terautentikasi"})
			c.Abort()
			return
		}

		userRole := domain.UserRole(roleStr.(string))
		for _, allowedRole := range roles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Akses ditolak: role tidak memiliki izin untuk endpoint ini",
		})
		c.Abort()
	}
}

// GetUserID helper untuk mengambil user ID dari context
func GetUserID(c *gin.Context) string {
	id, _ := c.Get(ContextUserID)
	return id.(string)
}

// GetUserRole helper untuk mengambil role dari context
func GetUserRole(c *gin.Context) domain.UserRole {
	role, _ := c.Get(ContextRole)
	return domain.UserRole(role.(string))
}
