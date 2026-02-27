package middleware

import (
	"net/http"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
)

// FeatureChecker returns the FeatureSet for the current request.
type FeatureChecker func(*http.Request) models.FeatureSet

// RequireFeature returns middleware that restricts access to requests where
// the company has the given feature enabled. Super_admin always passes through.
// If the feature is missing, onDenied is called.
func RequireFeature(check FeatureChecker, feature models.Feature, onDenied http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.GetUserFromRequest(r)
			if ok && user.Role == "super_admin" {
				next.ServeHTTP(w, r)
				return
			}
			fs := check(r)
			if !fs.Has(feature) {
				onDenied(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
