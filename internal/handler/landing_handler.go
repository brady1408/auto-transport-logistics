package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/brady1408/auto-transport-logistics/internal/email"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/pages"
	"github.com/brady1408/auto-transport-logistics/internal/middleware"
)

type LandingHandler struct {
	deps     *Deps
	emailSvc *email.Service
}

func NewLandingHandler(deps *Deps, emailSvc *email.Service) *LandingHandler {
	return &LandingHandler{deps: deps, emailSvc: emailSvc}
}

func (h *LandingHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /landing", h.show)
	mux.HandleFunc("POST /contact", h.handleContact)
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

func (h *LandingHandler) handleContact(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	contactEmail := strings.TrimSpace(r.FormValue("email"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	company := strings.TrimSpace(r.FormValue("company"))
	message := strings.TrimSpace(r.FormValue("message"))

	if name == "" || contactEmail == "" {
		http.Error(w, "Name and email are required", http.StatusBadRequest)
		return
	}

	// Send confirmation to the prospect
	if h.emailSvc.Enabled() {
		if err := h.emailSvc.SendContactConfirmation(contactEmail, name); err != nil {
			log.Printf("contact confirmation email error: %v", err)
		}
		// Notify the team
		if err := h.emailSvc.SendContactNotification(name, contactEmail, phone, company, message); err != nil {
			log.Printf("contact notification email error: %v", err)
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="text-center py-8"><span class="material-symbols-outlined text-on-tertiary-container text-4xl mb-4" style="font-size:48px;">check_circle</span><h3 class="text-xl font-bold text-slate-900 mb-2">Thank you!</h3><p class="text-sm text-secondary">We've received your request and will be in touch soon.</p></div>`))
}
