package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/denisAlshanov/stPlaner/internal/config"
	"github.com/denisAlshanov/stPlaner/internal/utils"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, times := range rl.requests {
			// Remove old entries
			validTimes := []time.Time{}
			for _, t := range times {
				if now.Sub(t) <= rl.window {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validTimes
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) isAllowed(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get existing requests
	times, exists := rl.requests[key]
	if !exists {
		rl.requests[key] = []time.Time{now}
		return true
	}

	// Count requests within window
	validTimes := []time.Time{}
	for _, t := range times {
		if now.Sub(t) <= rl.window {
			validTimes = append(validTimes, t)
		}
	}

	// Check if limit exceeded
	if len(validTimes) >= rl.limit {
		rl.requests[key] = validTimes
		return false
	}

	// Add new request
	validTimes = append(validTimes, now)
	rl.requests[key] = validTimes
	return true
}

func RateLimitMiddleware(cfg *config.APIConfig) gin.HandlerFunc {
	limiter := newRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindow)

	return func(c *gin.Context) {
		// Use IP address as rate limit key
		// In production, you might want to use user ID or API key
		key := c.ClientIP()

		// Override with user ID if available (from auth middleware)
		if userID, exists := c.Get("user_id"); exists {
			key = userID.(string)
		}

		if !limiter.isAllowed(key) {
			c.JSON(429, gin.H{
				"error":      utils.NewRateLimitError(),
				"request_id": c.GetString("request_id"),
				"timestamp":  time.Now().Format(time.RFC3339),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
