package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/brady1408/atlinks/internal/auth"
)

const CookieName = "atlinks_token"

func RequireAuth(jwt *auth.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(CookieName)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			claims, err := jwt.ValidateToken(cookie.Value)
			if err != nil {
				clearAuthCookie(w)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Require company assignment for non-super_admin users
			if claims.CompanyID == 0 && claims.Role != "super_admin" {
				clearAuthCookie(w)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := auth.SetUser(r.Context(), auth.ContextUser{
				ID:        claims.UserID,
				Username:  claims.Username,
				Role:      claims.Role,
				CompanyID: claims.CompanyID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func RequireAPIKey(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if apiKey == "" {
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{"error": "API not configured"})
				return
			}

			key := r.Header.Get("X-API-Key")
			if key == "" || key != apiKey {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid or missing API key"})
				return
			}

			// Inject synthetic super_admin context so store queries see all data
			ctx := auth.SetUser(r.Context(), auth.ContextUser{
				ID:        0,
				Username:  "api",
				Role:      "super_admin",
				CompanyID: 0,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
