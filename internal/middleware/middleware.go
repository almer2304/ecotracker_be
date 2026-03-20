package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// ============================================================
// RATE LIMITER
// ============================================================

// RateLimiter membatasi jumlah request per IP menggunakan token bucket dengan Redis
func RateLimiter(redisClient *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Jika Redis tidak tersedia, skip rate limiting
		if redisClient == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s", ip)
		ctx := context.Background()

		// Increment counter
		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			// Redis error - izinkan request tetap jalan
			c.Next()
			return
		}

		// Set TTL hanya pada request pertama dalam window
		if count == 1 {
			redisClient.Expire(ctx, key, window)
		}

		// Tambahkan header info rate limit
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", maxRequests-int(count)))

		if int(count) > maxRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error":   "Terlalu banyak request. Coba lagi dalam beberapa saat.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ============================================================
// CORS
// ============================================================

// CORS mengizinkan cross-origin requests dari origins yang terdaftar
func CORS(allowedOrigins string) gin.HandlerFunc {
	origins := strings.Split(allowedOrigins, ",")
	originMap := make(map[string]bool)
	for _, o := range origins {
		originMap[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Cek apakah origin diizinkan
		if originMap[origin] || allowedOrigins == "*" {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, Accept, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// ============================================================
// LOGGER
// ============================================================

// Logger mencatat setiap HTTP request dengan logrus
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		entry := logrus.WithFields(logrus.Fields{
			"status":     statusCode,
			"method":     c.Request.Method,
			"path":       path,
			"query":      raw,
			"ip":         c.ClientIP(),
			"latency_ms": latency.Milliseconds(),
			"user_agent": c.Request.UserAgent(),
		})

		if userID, exists := c.Get(ContextUserID); exists {
			entry = entry.WithField("user_id", userID)
		}

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.String())
		} else if statusCode >= 500 {
			entry.Error("Server error")
		} else if statusCode >= 400 {
			entry.Warn("Client error")
		} else {
			entry.Info("Request selesai")
		}
	}
}

// ============================================================
// RECOVERY
// ============================================================

// Recovery menangani panic dan mengembalikan response 500
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logrus.WithField("panic", err).Error("Panic recovered")
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Terjadi kesalahan internal yang tidak terduga",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
