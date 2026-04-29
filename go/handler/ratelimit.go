package handler

import (
	"net/http"
	"sync"
	"time"
)

// ── Token Bucket Rate Limiter ────────────────────────

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64 // tokens per second
	burst   int     // max tokens
}

func newRateLimiter(perMinute int) *rateLimiter {
	return &rateLimiter{
		buckets: make(map[string]*bucket),
		rate:    float64(perMinute) / 60.0,
		burst:   perMinute,
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[ip]
	now := time.Now()

	if !exists {
		rl.buckets[ip] = &bucket{tokens: float64(rl.burst) - 1, lastRefill: now}
		return true
	}

	// Refill tokens
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Cleanup old buckets periodically (call in a goroutine)
func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for ip, b := range rl.buckets {
			if b.lastRefill.Before(cutoff) {
				delete(rl.buckets, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Pre-configured limiters
var (
	RegisterLimiter = newRateLimiter(5)  // 5 per minute
	LoginLimiter    = newRateLimiter(10) // 10 per minute
	AvatarLimiter   = newRateLimiter(3)  // 3 per minute
)

func init() {
	go RegisterLimiter.cleanup()
	go LoginLimiter.cleanup()
	go AvatarLimiter.cleanup()
}

// RateLimit wraps a handler with rate limiting
func RateLimit(rl *rateLimiter, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		// Try X-Forwarded-For for proxied requests
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ip = xff
		}
		if !rl.allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"too many requests, please try again later"}`))
			return
		}
		next(w, r)
	}
}
