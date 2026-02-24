package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfFormField  = "csrf_token"
	csrfTokenLen   = 32
)

// CSRF returns middleware that enforces double-submit cookie CSRF protection.
// GET/HEAD/OPTIONS requests set the token cookie if missing.
// POST/PUT/DELETE requests require the token in a header or form field matching the cookie.
func CSRF(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(csrfCookieName)

			// Ensure the token cookie exists
			if err != nil || cookie.Value == "" {
				token, genErr := generateCSRFToken()
				if genErr != nil {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
				http.SetCookie(w, &http.Cookie{
					Name:     csrfCookieName,
					Value:    token,
					Path:     "/",
					MaxAge:   86400, // 24 hours
					Secure:   secure,
					SameSite: http.SameSiteLaxMode,
					// Not HttpOnly — JS/HTMX needs to read it
				})
				cookie = &http.Cookie{Value: token}
			}

			// For safe methods, just pass through
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// For state-changing methods, validate the token
			submitted := r.Header.Get(csrfHeaderName)
			if submitted == "" {
				submitted = r.FormValue(csrfFormField)
			}

			if submitted == "" || submitted != cookie.Value {
				http.Error(w, "Forbidden - invalid CSRF token", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
