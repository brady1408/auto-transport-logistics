package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParseID(t *testing.T) {
	tests := []struct {
		name    string
		pathVal string
		want    int
		wantErr bool
	}{
		{"valid", "42", 42, false},
		{"zero", "0", 0, false},
		{"negative", "-1", -1, false},
		{"non-numeric", "abc", 0, true},
		{"float", "3.14", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a real ServeMux to set path values
			var gotID int
			var gotErr error
			mux := http.NewServeMux()
			mux.HandleFunc("GET /test/{id}", func(w http.ResponseWriter, r *http.Request) {
				gotID, gotErr = parseID(r)
			})

			r := httptest.NewRequest("GET", "/test/"+tt.pathVal, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)

			if tt.wantErr {
				if gotErr == nil {
					t.Error("expected error")
				}
			} else {
				if gotErr != nil {
					t.Errorf("unexpected error: %v", gotErr)
				}
				if gotID != tt.want {
					t.Errorf("got %d, want %d", gotID, tt.want)
				}
			}
		})
	}
}

func TestIsHTMX(t *testing.T) {
	r1 := httptest.NewRequest("GET", "/", nil)
	if isHTMX(r1) {
		t.Error("should be false without header")
	}

	r2 := httptest.NewRequest("GET", "/", nil)
	r2.Header.Set("HX-Request", "true")
	if !isHTMX(r2) {
		t.Error("should be true with HX-Request: true")
	}

	r3 := httptest.NewRequest("GET", "/", nil)
	r3.Header.Set("HX-Request", "false")
	if isHTMX(r3) {
		t.Error("should be false with HX-Request: false")
	}
}

func TestFormString(t *testing.T) {
	form := url.Values{"name": {"Alice"}, "empty": {""}, "spaces": {"  "}}
	r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got := formString(r, "name")
	if got == nil || *got != "Alice" {
		t.Errorf("formString(name) = %v, want Alice", got)
	}

	got = formString(r, "empty")
	if got != nil {
		t.Errorf("formString(empty) = %v, want nil", got)
	}

	got = formString(r, "spaces")
	if got != nil {
		t.Errorf("formString(spaces) = %v, want nil", got)
	}

	got = formString(r, "missing")
	if got != nil {
		t.Errorf("formString(missing) = %v, want nil", got)
	}
}

func TestFormStringRequired(t *testing.T) {
	form := url.Values{"name": {"  Alice  "}}
	r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if got := formStringRequired(r, "name"); got != "Alice" {
		t.Errorf("got %q, want Alice", got)
	}
	if got := formStringRequired(r, "missing"); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestFormInt(t *testing.T) {
	form := url.Values{"age": {"25"}, "empty": {""}, "bad": {"abc"}}
	r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	got := formInt(r, "age")
	if got == nil || *got != 25 {
		t.Errorf("formInt(age) = %v, want 25", got)
	}

	got = formInt(r, "empty")
	if got != nil {
		t.Errorf("formInt(empty) = %v, want nil", got)
	}

	got = formInt(r, "bad")
	if got != nil {
		t.Errorf("formInt(bad) = %v, want nil", got)
	}
}

func TestFormBool(t *testing.T) {
	form := url.Values{"on": {"on"}, "true": {"true"}, "one": {"1"}, "off": {"off"}, "empty": {""}}
	r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	tests := []struct {
		key  string
		want bool
	}{
		{"on", true},
		{"true", true},
		{"one", true},
		{"off", false},
		{"empty", false},
		{"missing", false},
	}
	for _, tt := range tests {
		if got := formBool(r, tt.key); got != tt.want {
			t.Errorf("formBool(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestFlashCookie(t *testing.T) {
	// setFlash sets a cookie
	w := httptest.NewRecorder()
	setFlash(w, "Item created")

	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "flash" {
			found = true
			if c.Value != "Item created" {
				t.Errorf("flash value = %q, want %q", c.Value, "Item created")
			}
		}
	}
	if !found {
		t.Error("flash cookie not set")
	}

	// getFlash reads and clears
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "flash", Value: "Hello"})
	w2 := httptest.NewRecorder()
	msg := getFlash(w2, r)
	if msg != "Hello" {
		t.Errorf("getFlash = %q, want Hello", msg)
	}
	// Should have set a clearing cookie
	for _, c := range w2.Result().Cookies() {
		if c.Name == "flash" && c.MaxAge != -1 {
			t.Error("flash cookie should be cleared (MaxAge=-1)")
		}
	}
}

func TestGetFlashNoCookie(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	msg := getFlash(w, r)
	if msg != "" {
		t.Errorf("getFlash with no cookie = %q, want empty", msg)
	}
}
