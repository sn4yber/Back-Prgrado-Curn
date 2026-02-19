package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	apperrors "github.com/sn4yber/curn-networking/pkg/errors"
	"github.com/sn4yber/curn-networking/pkg/logger"
	"go.uber.org/zap"
)

// ipBucket almacena el conteo de intentos y el momento de reset para una IP.
type ipBucket struct {
	count   int
	resetAt time.Time
}

// rateLimiter controla los intentos por IP en memoria.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*ipBucket
	maxReqs int
	window  time.Duration
	log     logger.Logger
}

// newRateLimiter construye el rate limiter con los parámetros del config.
func newRateLimiter(maxReqs int, window time.Duration, log logger.Logger) *rateLimiter {
	rl := &rateLimiter{
		buckets: make(map[string]*ipBucket),
		maxReqs: maxReqs,
		window:  window,
		log:     log,
	}
	go rl.cleanupLoop()
	return rl
}

// allow devuelve true si la IP todavía tiene intentos disponibles.
func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.buckets[ip]

	if !exists || now.After(bucket.resetAt) {
		rl.buckets[ip] = &ipBucket{count: 1, resetAt: now.Add(rl.window)}
		return true
	}

	if bucket.count >= rl.maxReqs {
		return false
	}

	bucket.count++
	return true
}

// cleanupLoop elimina buckets expirados cada minuto para evitar memory leaks.
func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, bucket := range rl.buckets {
			if now.After(bucket.resetAt) {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitLogin devuelve el middleware de Gin que limita intentos de login por IP.
func RateLimitLogin(maxReqs int, window time.Duration, log logger.Logger) gin.HandlerFunc {
	rl := newRateLimiter(maxReqs, window, log)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !rl.allow(ip) {
			log.Audit("rate limit excedido en login",
				zap.String("ip", ip),
			)

			appErr := apperrors.ErrRateLimited
			c.AbortWithStatusJSON(appErr.Code, gin.H{
				"error": appErr.Message,
			})
			return
		}

		c.Next()
	}
}

// CORS configura las cabeceras de Cross-Origin Resource Sharing.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
