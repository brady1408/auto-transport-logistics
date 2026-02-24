package handler

import (
	"encoding/json"
	"net/http"

	"github.com/brady1408/atlinks/internal/handler/components/pages"
	"github.com/brady1408/atlinks/internal/store"
)

type VinSearchHandler struct {
	vehicleStore *store.VehicleStore
	deps         *Deps
}

func NewVinSearchHandler(vehicleStore *store.VehicleStore, deps *Deps) *VinSearchHandler {
	return &VinSearchHandler{vehicleStore: vehicleStore, deps: deps}
}

func (h *VinSearchHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /search/vin", h.page)
	mux.HandleFunc("GET /api/vehicles/search-global", h.searchAPI)
}

func (h *VinSearchHandler) page(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	var results []store.GlobalSearchResult

	if query != "" {
		var err error
		results, err = h.vehicleStore.SearchGlobal(r.Context(), query, 100)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, pages.VinSearchResults(query, results))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, pages.VinSearchPage(pg, query, results))
}

func (h *VinSearchHandler) searchAPI(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}

	results, err := h.vehicleStore.SearchGlobal(r.Context(), query, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
