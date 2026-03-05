package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
)

// ApiKeyLookup is the subset of ApiKeyStore used by RequireAPIKey.
type ApiKeyLookup interface {
	GetByKeyHash(ctx context.Context, hash string) (*models.ApiKey, error)
	UpdateLastUsed(ctx context.Context, id int)
}

// wantsJSON returns true if the request prefers a JSON response (API clients, mobile apps).
func wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json") ||
		strings.HasPrefix(r.URL.Path, "/api/")
}

const CookieName = "atlinks_token"

func RequireAuth(jwt *auth.JWTService, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenString string

			// Try Bearer token first (mobile app)
			if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			} else if cookie, err := r.Cookie(CookieName); err == nil {
				// Fall back to cookie (web app)
				tokenString = cookie.Value
			}

			if tokenString == "" {
				if wantsJSON(r) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
					return
				}
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			claims, err := jwt.ValidateToken(tokenString)
			if err != nil {
				if wantsJSON(r) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired token"})
					return
				}
				clearAuthCookie(w, secure)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Require company assignment for non-super_admin users
			if claims.CompanyID == 0 && claims.Role != "super_admin" {
				if wantsJSON(r) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]string{"error": "no company assigned"})
					return
				}
				clearAuthCookie(w, secure)
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

func clearAuthCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func RequireAPIKey(store ApiKeyLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			raw := r.Header.Get("X-API-Key")
			if raw == "" {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "missing API key"})
				return
			}

			hash := auth.HashAPIKey(raw)
			key, err := store.GetByKeyHash(r.Context(), hash)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid or inactive API key"})
				return
			}

			go store.UpdateLastUsed(context.Background(), key.ID)

			ctx := auth.SetUser(r.Context(), auth.ContextUser{
				ID:        key.UserID,
				Username:  key.Username,
				Role:      "super_admin",
				CompanyID: key.CompanyID,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
