package handler

import (
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/middleware"
	"github.com/brady1408/atlinks/internal/store"
)

type AuthHandler struct {
	users *store.UserStore
	deps  *Deps
}

func NewAuthHandler(users *store.UserStore, deps *Deps) *AuthHandler {
	return &AuthHandler{users: users, deps: deps}
}

func (h *AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", h.showLogin)
	mux.HandleFunc("POST /login", h.handleLogin)
	mux.HandleFunc("POST /logout", h.handleLogout)
}

func (h *AuthHandler) showLogin(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	if cookie, err := r.Cookie(middleware.CookieName); err == nil {
		if _, err := h.deps.JWT.ValidateToken(cookie.Value); err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	h.deps.render(w, r, "login.html", nil)
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := h.users.GetByUsername(r.Context(), username)
	if err != nil || !user.Active {
		h.deps.render(w, r, "login.html", map[string]any{
			"Error":    "Invalid username or password",
			"Username": username,
		})
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		h.deps.render(w, r, "login.html", map[string]any{
			"Error":    "Invalid username or password",
			"Username": username,
		})
		return
	}

	token, err := h.deps.JWT.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(24 * time.Hour / time.Second),
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
