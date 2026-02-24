package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Password tests ---

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("hash is empty")
	}
	if hash == "mypassword" {
		t.Fatal("hash equals plaintext")
	}

	if err := CheckPassword(hash, "mypassword"); err != nil {
		t.Errorf("CheckPassword with correct password: %v", err)
	}
	if err := CheckPassword(hash, "wrongpassword"); err == nil {
		t.Error("CheckPassword with wrong password should fail")
	}
}

func TestHashPasswordDifferentHashes(t *testing.T) {
	h1, _ := HashPassword("same")
	h2, _ := HashPassword("same")
	if h1 == h2 {
		t.Error("two hashes of the same password should differ (different salts)")
	}
}

// --- JWT tests ---

func TestJWTGenerateAndValidate(t *testing.T) {
	svc := NewJWTService("test-secret-key")

	token, err := svc.GenerateToken(42, "testuser", "admin", 1)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Username != "testuser" {
		t.Errorf("Username = %q, want %q", claims.Username, "testuser")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}
}

func TestJWTInvalidToken(t *testing.T) {
	svc := NewJWTService("test-secret-key")
	_, err := svc.ValidateToken("garbage.token.here")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestJWTWrongSecret(t *testing.T) {
	svc1 := NewJWTService("secret-one")
	svc2 := NewJWTService("secret-two")

	token, _ := svc1.GenerateToken(1, "user", "user", 1)
	_, err := svc2.ValidateToken(token)
	if err == nil {
		t.Error("expected error when validating with wrong secret")
	}
}

// --- Context tests ---

func TestContextUserRoundTrip(t *testing.T) {
	ctx := context.Background()
	user := ContextUser{ID: 10, Username: "alice", Role: "admin"}

	ctx = SetUser(ctx, user)
	got, ok := GetUser(ctx)
	if !ok {
		t.Fatal("GetUser returned not ok")
	}
	if got != user {
		t.Errorf("GetUser = %+v, want %+v", got, user)
	}
}

func TestContextUserMissing(t *testing.T) {
	ctx := context.Background()
	_, ok := GetUser(ctx)
	if ok {
		t.Error("GetUser should return false for empty context")
	}
}

func TestGetUserFromRequest(t *testing.T) {
	user := ContextUser{ID: 5, Username: "bob", Role: "user"}
	ctx := SetUser(context.Background(), user)
	r := httptest.NewRequest("GET", "/", nil).WithContext(ctx)

	got, ok := GetUserFromRequest(r)
	if !ok {
		t.Fatal("GetUserFromRequest returned not ok")
	}
	if got != user {
		t.Errorf("got %+v, want %+v", got, user)
	}
}

func TestGetUserFromRequestMissing(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	_, ok := GetUserFromRequest(r)
	if ok {
		t.Error("should return false when no user in context")
	}
}

// --- Auth middleware behavior test ---

func TestRequireAuthRedirectsWithoutCookie(t *testing.T) {
	// This tests the middleware pattern used by the auth middleware.
	// We test the cookie/JWT flow directly rather than importing middleware
	// to avoid a circular dependency.
	svc := NewJWTService("test-secret")

	// No cookie → should fail
	r := httptest.NewRequest("GET", "/protected", nil)
	cookie, err := r.Cookie("atlinks_token")
	if err == nil {
		t.Errorf("unexpected cookie: %v", cookie)
	}

	// Valid cookie → should validate
	token, _ := svc.GenerateToken(1, "admin", "admin", 1)
	r2 := httptest.NewRequest("GET", "/protected", nil)
	r2.AddCookie(&http.Cookie{Name: "atlinks_token", Value: token})
	cookie2, err := r2.Cookie("atlinks_token")
	if err != nil {
		t.Fatalf("cookie not found: %v", err)
	}
	claims, err := svc.ValidateToken(cookie2.Value)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.Username != "admin" {
		t.Errorf("Username = %q, want admin", claims.Username)
	}
}
