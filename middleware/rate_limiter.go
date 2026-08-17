package middleware

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	count   int
	resetAt time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: make(map[string]*bucket)}
}

func (rl *RateLimiter) Allow(key string, limit int, window time.Duration) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.resetAt) {
		rl.buckets[key] = &bucket{count: 1, resetAt: now.Add(window)}
		return true, 0
	}
	if b.count >= limit {
		retryAfter := int(math.Ceil(time.Until(b.resetAt).Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}
		return false, retryAfter
	}
	b.count++
	return true, 0
}

func getClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.SplitN(fwd, ",", 2)
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return strings.TrimSpace(ip)
	}
	return "unknown"
}

func (rl *RateLimiter) Enforce(scope string, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			client := getClientIP(r)
			key := fmt.Sprintf("%s:%s", scope, client)
			allowed, retryAfter := rl.Allow(key, limit, window)
			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":             "Rate limit exceeded. Please retry later.",
					"retryAfterSeconds": retryAfter,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
