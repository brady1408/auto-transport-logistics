package handler

import (
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/middleware"
	"github.com/brady1408/atlinks/internal/store"
)

type AuthHandler struct {
	users        *store.UserStore
	companyStore *store.CompanyStore
	deps         *Deps
}

func NewAuthHandler(users *store.UserStore, companyStore *store.CompanyStore, deps *Deps) *AuthHandler {
	return &AuthHandler{users: users, companyStore: companyStore, deps: deps}
}

func (h *AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", h.showLogin)
	mux.HandleFunc("POST /login", h.handleLogin)
	mux.HandleFunc("POST /logout", h.handleLogout)
	// Company-specific login
	mux.HandleFunc("GET /c/{slug}/login", h.showCompanyLogin)
	mux.HandleFunc("POST /c/{slug}/login", h.handleCompanyLogin)
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

	companyID := 0
	if user.CompanyID != nil {
		companyID = *user.CompanyID
	}
	token, err := h.deps.JWT.GenerateToken(user.ID, user.Username, user.Role, companyID)
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

func (h *AuthHandler) showCompanyLogin(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	company, err := h.companyStore.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// If already logged in, redirect to dashboard
	if cookie, err := r.Cookie(middleware.CookieName); err == nil {
		if _, err := h.deps.JWT.ValidateToken(cookie.Value); err == nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	h.deps.render(w, r, "login.html", map[string]any{
		"CompanyName": company.CompanyName,
		"LoginAction": "/c/" + slug + "/login",
	})
}

func (h *AuthHandler) handleCompanyLogin(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	company, err := h.companyStore.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	loginData := map[string]any{
		"Error":       "Invalid username or password",
		"Username":    username,
		"CompanyName": company.CompanyName,
		"LoginAction": "/c/" + slug + "/login",
	}

	user, err := h.users.GetByUsername(r.Context(), username)
	if err != nil || !user.Active {
		h.deps.render(w, r, "login.html", loginData)
		return
	}

	// Verify user belongs to this company
	if user.CompanyID == nil || *user.CompanyID != company.ID {
		h.deps.render(w, r, "login.html", loginData)
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		h.deps.render(w, r, "login.html", loginData)
		return
	}

	token, err := h.deps.JWT.GenerateToken(user.ID, user.Username, user.Role, company.ID)
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
