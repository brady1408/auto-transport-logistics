package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/models"
)

type apiCustomerStore interface {
	List(ctx context.Context, f models.CustomerFilter) (*models.CustomerListResult, error)
}

type apiVehicleStore interface {
	SearchUnassigned(ctx context.Context, search string, limit int) ([]models.OrderVehicle, error)
}

type APIHandler struct {
	custStore apiCustomerStore
	vehStore  apiVehicleStore
	deps      *Deps
}

func NewAPIHandler(custStore apiCustomerStore, vehStore apiVehicleStore, deps *Deps) *APIHandler {
	return &APIHandler{custStore: custStore, vehStore: vehStore, deps: deps}
}

func (h *APIHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/customers/search", h.customerSearch)
	mux.HandleFunc("GET /api/vehicles/search", h.vehicleSearch)
	mux.HandleFunc("GET /api/vin/decode", h.vinDecode)
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
		serverError(w, err)
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
		serverError(w, err)
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

// vinDecode calls the NHTSA vPIC API to decode a VIN and returns
// year, make, model, body style, and weight.
func (h *APIHandler) vinDecode(w http.ResponseWriter, r *http.Request) {
	vin := strings.TrimSpace(r.URL.Query().Get("vin"))
	if len(vin) != 17 {
		writeJSON(w, map[string]string{"error": "VIN must be 17 characters"})
		return
	}

	url := fmt.Sprintf("https://vpic.nhtsa.dot.gov/api/vehicles/decodevin/%s?format=json", vin)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		writeJSON(w, map[string]string{"error": "Failed to reach NHTSA API"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeJSON(w, map[string]string{"error": "Failed to read NHTSA response"})
		return
	}

	var nhtsa nhtsaResponse
	if err := json.Unmarshal(body, &nhtsa); err != nil {
		writeJSON(w, map[string]string{"error": "Failed to parse NHTSA response"})
		return
	}

	result := vinDecodeResult{}
	for _, item := range nhtsa.Results {
		val := strings.TrimSpace(item.Value)
		if val == "" || val == "Not Applicable" {
			continue
		}
		switch item.VariableID {
		case 29: // ModelYear
			result.Year = val
		case 26: // Make
			result.Make = val
		case 28: // Model
			result.Model = val
		case 5: // BodyClass
			result.BodyStyle = val
		case 27: // PlantCity (skip)
		case 25: // GrossVehicleWeightRating (GVWR)
			result.Weight = val
		}
	}

	writeJSON(w, result)
}

type nhtsaResponse struct {
	Results []nhtsaResult `json:"Results"`
}

type nhtsaResult struct {
	Value      string `json:"Value"`
	ValueID    string `json:"ValueId"`
	Variable   string `json:"Variable"`
	VariableID int    `json:"VariableId"`
}

type vinDecodeResult struct {
	Year      string `json:"year"`
	Make      string `json:"make"`
	Model     string `json:"model"`
	BodyStyle string `json:"body_style"`
	Weight    string `json:"weight"`
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
