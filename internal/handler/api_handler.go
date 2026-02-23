package handler

import (
	"encoding/json"
	"net/http"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type APIHandler struct {
	custStore *store.CustomerStore
	vehStore  *store.VehicleStore
	deps      *Deps
}

func NewAPIHandler(custStore *store.CustomerStore, vehStore *store.VehicleStore, deps *Deps) *APIHandler {
	return &APIHandler{custStore: custStore, vehStore: vehStore, deps: deps}
}

func (h *APIHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/customers/search", h.customerSearch)
	mux.HandleFunc("GET /api/vehicles/search", h.vehicleSearch)
}

type customerSearchResult struct {
	ID      int     `json:"id"`
	Number  *string `json:"number"`
	Name    string  `json:"name"`
	Address *string `json:"address"`
	City    *string `json:"city"`
	State   *string `json:"state"`
	Zip     *string `json:"zip"`
	Contact *string `json:"contact"`
	Phone   *string `json:"phone"`
}

func (h *APIHandler) customerSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, []customerSearchResult{})
		return
	}

	// Use customer store list with search filter, limit to 10
	result, err := h.custStore.List(r.Context(), models.CustomerFilter{Search: q, PageSize: 10, Page: 1})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var results []customerSearchResult
	for _, c := range result.Items {
		results = append(results, customerSearchResult{
			ID:      c.ID,
			Number:  c.Number,
			Name:    c.Name,
			Address: c.Address,
			City:    c.City,
			State:   c.State,
			Zip:     c.Zip,
			Contact: c.Contact,
			Phone:   c.Phone,
		})
	}

	writeJSON(w, results)
}

type vehicleSearchResult struct {
	ID          int     `json:"id"`
	OrderID     int     `json:"order_id"`
	VIN         *string `json:"vin"`
	Year        *string `json:"year"`
	Make        *string `json:"make"`
	Model       *string `json:"model"`
	Color       *string `json:"color"`
	Status      string  `json:"status"`
	OrderNumber string  `json:"order_number"`
}

func (h *APIHandler) vehicleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, []vehicleSearchResult{})
		return
	}

	vehicles, err := h.vehStore.SearchUnassigned(r.Context(), q, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var results []vehicleSearchResult
	for _, v := range vehicles {
		results = append(results, vehicleSearchResult{
			ID:      v.ID,
			OrderID: v.OrderID,
			VIN:     v.VIN,
			Year:    v.Year,
			Make:    v.Make,
			Model:   v.Model,
			Color:   v.Color,
			Status:  v.Status,
		})
	}

	writeJSON(w, results)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
