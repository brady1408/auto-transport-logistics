package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components/trailers"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

type trailerStore interface {
	List(ctx context.Context, f models.TrailerFilter) (*models.TrailerListResult, error)
	GetByID(ctx context.Context, id int) (*models.Trailer, error)
	Create(ctx context.Context, t *models.Trailer) error
	Update(ctx context.Context, t *models.Trailer) error
	Delete(ctx context.Context, id int) error
}

type TrailerHandler struct {
	store trailerStore
	deps  *Deps
}

func NewTrailerHandler(store trailerStore, deps *Deps) *TrailerHandler {
	return &TrailerHandler{store: store, deps: deps}
}

func (h *TrailerHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/trailers", h.list)
	mux.HandleFunc("GET /global/trailers/new", h.newForm)
	mux.HandleFunc("POST /global/trailers", h.create)
	mux.HandleFunc("GET /global/trailers/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /global/trailers/{id}", h.update)
	mux.HandleFunc("DELETE /global/trailers/{id}", h.delete)
}

func (h *TrailerHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.TrailerFilter{
		Search:   r.URL.Query().Get("search"),
		Active:   r.URL.Query().Get("active"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, trailers.Table(*result, filter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, trailers.ListPage(pg, *result, filter))
}

func (h *TrailerHandler) newForm(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, trailers.FormPage(pg, &models.Trailer{Active: true}, true, ""))
}

func (h *TrailerHandler) create(w http.ResponseWriter, r *http.Request) {
	t := bindTrailerForm(r)

	if t.TrailerNumber == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, trailers.FormPage(pg, t, true, "Trailer Number is required"))
		return
	}

	if err := h.store.Create(r.Context(), t); err != nil {
		log.Printf("create trailer: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, trailers.FormPage(pg, t, true, "Failed to create trailer"))
		return
	}

	h.deps.Audit.Log(r.Context(), "trailers", t.ID, "INSERT", nil, t)
	h.deps.setFlash(w, "Trailer created successfully")

	redirect(w, r, "/global/trailers")
}

func (h *TrailerHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	t, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trailer not found", http.StatusNotFound)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, trailers.FormPage(pg, t, false, ""))
}

func (h *TrailerHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trailer not found", http.StatusNotFound)
		return
	}

	t := bindTrailerForm(r)
	t.ID = id

	if t.TrailerNumber == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, trailers.FormPage(pg, t, false, "Trailer Number is required"))
		return
	}

	if err := h.store.Update(r.Context(), t); err != nil {
		log.Printf("update trailer: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, trailers.FormPage(pg, t, false, "Failed to update trailer"))
		return
	}

	h.deps.Audit.Log(r.Context(), "trailers", t.ID, "UPDATE", old, t)
	h.deps.setFlash(w, "Trailer updated successfully")

	redirect(w, r, "/global/trailers")
}

func (h *TrailerHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Trailer not found", http.StatusNotFound)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "trailers", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Trailer deleted")

	redirectBack(w, r, "/global/trailers")
}

func bindTrailerForm(r *http.Request) *models.Trailer {
	t := &models.Trailer{
		TrailerNumber: formStringRequired(r, "trailer_number"),
		Make:          formString(r, "make"),
		Model:         formString(r, "model"),
		Year:          formString(r, "year"),
		SerialNumber:  formString(r, "serial_number"),
		TypeCode:      formString(r, "type_code"),
		License:       formString(r, "license"),
		TareWeight:    formInt(r, "tare_weight"),
		Capacity:      formInt(r, "capacity"),
		LengthFt:      formString(r, "length_ft"),
		WidthFt:       formString(r, "width_ft"),
		HeightFt:      formString(r, "height_ft"),
		PurchasedFrom: formString(r, "purchased_from"),
		Cost:          formString(r, "cost"),
		Comments:      formString(r, "comments"),
		Active:        formBool(r, "active"),
	}

	for _, df := range []struct {
		field string
		dest  **time.Time
	}{
		{"manufacture_date", &t.ManufactureDate},
		{"license_exp", &t.LicenseExp},
		{"safety_inspection", &t.SafetyInspection},
		{"purchase_date", &t.PurchaseDate},
	} {
		if v := r.FormValue(df.field); v != "" {
			if parsed, err := time.Parse("2006-01-02", v); err == nil {
				*df.dest = &parsed
			}
		}
	}

	return t
}
