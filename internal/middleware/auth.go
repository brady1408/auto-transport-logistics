package middleware

import (
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
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Require company assignment for non-super_admin users
			if claims.CompanyID == 0 && claims.Role != "super_admin" {
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
