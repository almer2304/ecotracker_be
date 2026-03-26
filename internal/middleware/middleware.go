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

func RateLimiter(redisClient *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if redisClient == nil {
			c.Next()
			return
		}

		ip := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s", ip)
		ctx := context.Background()

		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			redisClient.Expire(ctx, key, window)
		}

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

func CORS(allowedOrigins string) gin.HandlerFunc {
	origins := strings.Split(allowedOrigins, ",")
	originMap := make(map[string]bool)
	for _, o := range origins {
		originMap[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

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
// LOGGER — lebih bersih & filter noise
// ============================================================

// path yang di-skip dari logging (terlalu banyak noise)
var skipLogPaths = map[string]bool{
	"/health": true,
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		// Skip path yang tidak perlu di-log
		if skipLogPaths[path] {
			return
		}

		// Skip OPTIONS preflight — tidak informatif
		if c.Request.Method == http.MethodOptions {
			return
		}

		// Skip WebSocket yang gagal upgrade (401) — ini noise dari reconnect loop
		// Hanya log WS yang berhasil (101 Switching Protocols)
		statusCode := c.Writer.Status()
		if path == "/ws" && statusCode != http.StatusSwitchingProtocols {
			// Log hanya sekali per menit per IP untuk WS failure (suppress spam)
			// Cukup debug level saja
			logrus.WithFields(logrus.Fields{
				"status": statusCode,
				"ip":     c.ClientIP(),
			}).Debug("WS connect gagal")
			return
		}

		latency := time.Since(start)

		// Bangun query string yang bersih (sembunyikan token di URL)
		cleanQuery := query
		if strings.Contains(query, "token=") {
			cleanQuery = "[token hidden]"
		}

		entry := logrus.WithFields(logrus.Fields{
			"status":     statusCode,
			"method":     c.Request.Method,
			"path":       path,
			"latency_ms": latency.Milliseconds(),
			"ip":         c.ClientIP(),
		})

		// Tambah query hanya kalau ada dan bukan token
		if cleanQuery != "" {
			entry = entry.WithField("query", cleanQuery)
		}

		// Tambah user_id kalau ada
		if userID, exists := c.Get(ContextUserID); exists {
			entry = entry.WithField("user_id", userID)
		}

		switch {
		case len(c.Errors) > 0:
			entry.WithField("errors", c.Errors.String()).Error("Handler error")
		case statusCode >= 500:
			entry.Error("Server error")
		case statusCode >= 400:
			// Jangan log 401 sebagai warning — terlalu banyak noise dari token expired
			if statusCode == 401 {
				entry.Debug("Unauthorized")
			} else {
				entry.Warn("Client error")
			}
		default:
			entry.Info("OK")
		}
	}
}

// ============================================================
// RECOVERY
// ============================================================

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