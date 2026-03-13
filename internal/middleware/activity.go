package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/store"
)

// ActivityTracker returns middleware that records each HTTP request to the activity_log table.
// It runs after auth middleware so user identity is available. Inserts are fire-and-forget.
func ActivityTracker(activityStore *store.ActivityStore) func(http.Handler) http.Handler {
	skip := func(path string) bool {
		return strings.HasPrefix(path, "/static/") ||
			path == "/favicon.ico" ||
			path == "/health" ||
			path == "/notifications/count"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			durationMS := int(time.Since(start).Milliseconds())
			l := buildActivityLog(r, sw.status, durationMS)

			// Fire-and-forget: use context.WithoutCancel so the request context
			// cancellation doesn't abort the INSERT.
			insertCtx := context.WithoutCancel(r.Context())
			go func() {
				if err := activityStore.Insert(insertCtx, l); err != nil {
					log.Printf("activity log insert: %v", err)
				}
			}()
		})
	}
}

func buildActivityLog(r *http.Request, status, durationMS int) models.ActivityLog {
	l := models.ActivityLog{
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: status,
		DurationMS: durationMS,
	}

	// Real IP: Cloudflare header > X-Forwarded-For > RemoteAddr
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		l.IPAddress = &ip
	} else if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For may be comma-separated; take the first
		if idx := strings.Index(ip, ","); idx != -1 {
			ip = strings.TrimSpace(ip[:idx])
		}
		l.IPAddress = &ip
	} else if addr := r.RemoteAddr; addr != "" {
		// Strip port
		if idx := strings.LastIndex(addr, ":"); idx != -1 {
			addr = addr[:idx]
		}
		l.IPAddress = &addr
	}

	if ua := r.Header.Get("User-Agent"); ua != "" {
		if len(ua) > 500 {
			ua = ua[:500]
		}
		l.UserAgent = &ua
	}

	if user, ok := auth.GetUser(r.Context()); ok {
		uid := user.ID
		cid := user.CompanyID
		l.UserID = &uid
		l.Username = &user.Username
		l.CompanyID = &cid
	}

	return l
}
