package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nurhudajoantama/hmauto/internal/util"
	"github.com/rs/zerolog/hlog"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int           // requests per interval
	interval time.Duration // time window
	maxBurst int           // maximum burst size
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, interval time.Duration, maxBurst int) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*bucket),
		rate:     rate,
		interval: interval,
		maxBurst: maxBurst,
	}

	// Cleanup old buckets every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// Allow checks if a request from the given identifier is allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[identifier]
	
	if !exists {
		rl.buckets[identifier] = &bucket{
			tokens:   rl.maxBurst - 1,
			lastSeen: now,
		}
		return true
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(b.lastSeen)
	tokensToAdd := int(elapsed.Seconds() / rl.interval.Seconds() * float64(rl.rate))
	b.tokens = util.Min(b.tokens+tokensToAdd, rl.maxBurst)
	b.lastSeen = now

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanup removes old buckets
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for key, b := range rl.buckets {
		if b.lastSeen.Before(cutoff) {
			delete(rl.buckets, key)
		}
	}
}

// RateLimit middleware applies rate limiting per IP address
func RateLimit(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := hlog.FromRequest(r)
			
			// Use IP address as identifier
			// Extract client IP from X-Forwarded-For (take first IP which is the client)
			// or fall back to RemoteAddr
			ip := r.RemoteAddr
			if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
				// X-Forwarded-For format: client, proxy1, proxy2
				// Only trust the first IP (leftmost)
				if idx := strings.Index(forwardedFor, ","); idx > 0 {
					ip = strings.TrimSpace(forwardedFor[:idx])
				} else {
					ip = strings.TrimSpace(forwardedFor)
				}
			}

			if !limiter.Allow(ip) {
				l.Warn().Str("ip", ip).Msg("Rate limit exceeded")
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
