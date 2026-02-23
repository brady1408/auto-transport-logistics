package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brady1408/atlinks/internal/auth"
)

func TestRequireAuthRedirectsWithNoCookie(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret")
	mw := RequireAuth(jwtSvc)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
	if loc := w.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want /login", loc)
	}
}

func TestRequireAuthRedirectsWithBadToken(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret")
	mw := RequireAuth(jwtSvc)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/protected", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: "invalid-token"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want %d", w.Code, http.StatusSeeOther)
	}
}

func TestRequireAuthPassesWithValidToken(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret")
	token, _ := jwtSvc.GenerateToken(1, "admin", "admin")
	mw := RequireAuth(jwtSvc)

	var gotUser auth.ContextUser
	var gotOK bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotOK = auth.GetUserFromRequest(r)
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/protected", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !gotOK {
		t.Fatal("user not set in context")
	}
	if gotUser.Username != "admin" {
		t.Errorf("Username = %q, want admin", gotUser.Username)
	}
	if gotUser.ID != 1 {
		t.Errorf("UserID = %d, want 1", gotUser.ID)
	}
}

func TestRequestLoggerSetsStatus(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := RequestLogger(inner)
	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
