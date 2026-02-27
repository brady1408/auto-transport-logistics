package middleware

import "net/http"

// SuspendedChecker returns true if the current request belongs to a suspended account.
type SuspendedChecker func(*http.Request) bool

// ReadOnlyIfSuspended returns middleware that blocks write requests (POST/PUT/PATCH/DELETE)
// for suspended accounts by calling onBlocked. GET and HEAD requests always pass through.
func ReadOnlyIfSuspended(check SuspendedChecker, onBlocked http.HandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isWriteMethod(r.Method) && check(r) {
				onBlocked(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}
