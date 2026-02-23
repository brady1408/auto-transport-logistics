package handler

import "net/http"

type DashboardHandler struct {
	deps *Deps
}

func NewDashboardHandler(deps *Deps) *DashboardHandler {
	return &DashboardHandler{deps: deps}
}

func (h *DashboardHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.show)
}

func (h *DashboardHandler) show(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	h.deps.render(w, r, "dashboard.html", nil)
}
