package handler

import (
	"net/http"
	"strings"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/pages"
	"github.com/brady1408/auto-transport-logistics/internal/middleware"
)

type LandingHandler struct {
	deps *Deps
}

func NewLandingHandler(deps *Deps) *LandingHandler {
	return &LandingHandler{deps: deps}
}

func (h *LandingHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /landing", h.show)
}

func (h *LandingHandler) show(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, go to dashboard
	if cookie, err := r.Cookie(middleware.CookieName); err == nil {
		if claims, err := h.deps.JWT.ValidateToken(cookie.Value); err == nil {
			if claims.CompanyID != 0 || claims.Role == "super_admin" {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
	}

	// Only serve landing page on atlascloud.app; others go to login
	host := strings.Split(r.Host, ":")[0]
	if host != "atlascloud.app" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	brand := components.BrandFromHost(r.Host)
	h.deps.renderTempl(w, r, pages.LandingPage(brand))
}
