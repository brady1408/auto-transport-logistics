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
	token, _ := jwtSvc.GenerateToken(1, "admin", "admin", 1)
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

// --- CSRF tests ---

func TestCSRFSetsCookieOnGET(t *testing.T) {
	mw := CSRF(false)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mw(inner).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "csrf_token" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error("csrf_token cookie not set on GET")
	}
}

func TestCSRFBlocksPOSTWithoutToken(t *testing.T) {
	mw := CSRF(false)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("POST", "/", nil)
	r.AddCookie(&http.Cookie{Name: "csrf_token", Value: "abc123"})
	w := httptest.NewRecorder()
	mw(inner).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFBlocksPOSTWithWrongToken(t *testing.T) {
	mw := CSRF(false)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("POST", "/", nil)
	r.AddCookie(&http.Cookie{Name: "csrf_token", Value: "abc123"})
	r.Header.Set("X-CSRF-Token", "wrong-token")
	w := httptest.NewRecorder()
	mw(inner).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCSRFAllowsPOSTWithMatchingHeader(t *testing.T) {
	mw := CSRF(false)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("POST", "/", nil)
	r.AddCookie(&http.Cookie{Name: "csrf_token", Value: "abc123"})
	r.Header.Set("X-CSRF-Token", "abc123")
	w := httptest.NewRecorder()
	mw(inner).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- RequireRole tests ---

func TestRequireRoleBlocksUnauthorized(t *testing.T) {
	mw := RequireRole("super_admin")
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// No user context at all
	r := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	mw(inner).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (no user)", w.Code, http.StatusForbidden)
	}
}

func TestRequireRoleBlocksWrongRole(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret")
	token, _ := jwtSvc.GenerateToken(1, "user", "company_user", 1)
	authMw := RequireAuth(jwtSvc)
	roleMw := RequireRole("super_admin")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/admin", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	w := httptest.NewRecorder()
	authMw(roleMw(inner)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d (wrong role)", w.Code, http.StatusForbidden)
	}
}

func TestRequireRoleAllowsMatchingRole(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret")
	token, _ := jwtSvc.GenerateToken(1, "admin", "super_admin", 0)
	authMw := RequireAuth(jwtSvc)
	roleMw := RequireRole("super_admin")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/admin", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	w := httptest.NewRecorder()
	authMw(roleMw(inner)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireRoleAllowsMultipleRoles(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret")
	token, _ := jwtSvc.GenerateToken(1, "admin", "company_admin", 1)
	authMw := RequireAuth(jwtSvc)
	roleMw := RequireRole("company_admin", "super_admin")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/settings", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: token})
	w := httptest.NewRecorder()
	authMw(roleMw(inner)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
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
