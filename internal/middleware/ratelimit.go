package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateLimitEntry struct {
	count       int
	windowStart time.Time
}

type rateLimiter struct {
	maxReqs int
	window  time.Duration
	entries sync.Map
}

// RateLimitFunc returns a function that wraps an http.HandlerFunc with
// IP-based rate limiting. Exceeding the limit returns HTTP 429 with
// {"error":"slow_down"} and a Retry-After header (OAuth2 device flow spec).
func RateLimitFunc(maxReqs int, window time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	rl := &rateLimiter{
		maxReqs: maxReqs,
		window:  window,
	}
	go rl.cleanup(2 * window)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()

			val, loaded := rl.entries.Load(ip)
			if !loaded {
				rl.entries.Store(ip, &rateLimitEntry{count: 1, windowStart: now})
				next(w, r)
				return
			}

			entry := val.(*rateLimitEntry)
			if now.Sub(entry.windowStart) >= rl.window {
				entry.count = 1
				entry.windowStart = now
				next(w, r)
				return
			}

			entry.count++
			if entry.count > rl.maxReqs {
				retryAfter := max(rl.window-now.Sub(entry.windowStart), time.Second)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "slow_down"})
				return
			}

			next(w, r)
		}
	}
}

func (rl *rateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		<-ticker.C
		now := time.Now()
		rl.entries.Range(func(key, val any) bool {
			entry := val.(*rateLimitEntry)
			if now.Sub(entry.windowStart) >= rl.window {
				rl.entries.Delete(key)
			}
			return true
		})
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (client's real IP)
		if i := strings.IndexByte(xff, ','); i > 0 {
			xff = strings.TrimSpace(xff[:i])
		}
		return xff
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
