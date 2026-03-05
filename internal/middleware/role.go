package middleware

import (
	"net/http"

	"github.com/brady1408/atlinks/internal/auth"
)

// RequireRole returns middleware that restricts access to users with one of the given roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.GetUserFromRequest(r)
			if !ok || !allowed[user.Role] {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BlockRole returns middleware that rejects requests from users with any of the given roles.
// API paths (/api/) receive a JSON 403; all others receive a plain-text 403.
func BlockRole(roles ...string) func(http.Handler) http.Handler {
	blocked := make(map[string]bool, len(roles))
	for _, r := range roles {
		blocked[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if user, ok := auth.GetUserFromRequest(r); ok && blocked[user.Role] {
				if wantsJSON(r) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"error":"access restricted to mobile app"}`))
					return
				}
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
