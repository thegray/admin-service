package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type rateLimiter struct {
	limiters sync.Map
	limit    rate.Limit
	burst    int
}

func RateLimitMiddleware(r rate.Limit, burst int) gin.HandlerFunc {
	rl := &rateLimiter{limit: r, burst: burst}
	return func(c *gin.Context) {
		key := c.ClientIP()
		if key == "" {
			key = "unknown"
		}

		limiter := rl.getLimiter(key)
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}

func (r *rateLimiter) getLimiter(key string) *rate.Limiter {
	if existing, ok := r.limiters.Load(key); ok {
		return existing.(*rate.Limiter)
	}

	lim := rate.NewLimiter(r.limit, r.burst)
	r.limiters.Store(key, lim)

	// encourage GC of old limiters by evicting after a timeout
	go func() {
		time.Sleep(10 * time.Minute)
		r.limiters.Delete(key)
	}()
	return lim
}
