package handler

import (
	"context"
	"net/http"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components/lookup"
	"github.com/brady1408/auto-transport-logistics/internal/store"
)

type termsStore interface {
	List(ctx context.Context) ([]store.TermItem, error)
	GetByID(ctx context.Context, id int) (*store.TermItem, error)
	Create(ctx context.Context, term, description string, days *int) (*store.TermItem, error)
	Delete(ctx context.Context, id int) error
}

type TermsHandler struct {
	store termsStore
	deps  *Deps
}

func NewTermsHandler(s termsStore, deps *Deps) *TermsHandler {
	return &TermsHandler{store: s, deps: deps}
}

func (h *TermsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/terms", h.list)
	mux.HandleFunc("POST /global/terms", h.create)
	mux.HandleFunc("DELETE /global/terms/{id}", h.delete)
}

func (h *TermsHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	if isHTMX(r) {
		h.deps.renderTempl(w, r, lookup.TermsTable(items))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, lookup.TermsListPage(pg, items))
}

func (h *TermsHandler) create(w http.ResponseWriter, r *http.Request) {
	term := formStringRequired(r, "term")
	desc := r.FormValue("description")
	days := formInt(r, "days")

	if term == "" {
		http.Error(w, "Term is required", http.StatusBadRequest)
		return
	}

	item, err := h.store.Create(r.Context(), term, desc, days)
	if err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "terms", item.ID, "INSERT", nil, item)
	h.deps.setFlash(w, "Term created")

	redirect(w, r, "/global/terms")
}

func (h *TermsHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, _ := h.store.GetByID(r.Context(), id)
	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "terms", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Term deleted")

	redirectBack(w, r, "/global/terms")
}

// Tax Codes Handler

type taxCodeStore interface {
	List(ctx context.Context) ([]store.TaxCodeItem, error)
	GetByID(ctx context.Context, id int) (*store.TaxCodeItem, error)
	Create(ctx context.Context, code, description string, rate *string) (*store.TaxCodeItem, error)
	Delete(ctx context.Context, id int) error
}

type TaxCodeHandler struct {
	store taxCodeStore
	deps  *Deps
}

func NewTaxCodeHandler(s taxCodeStore, deps *Deps) *TaxCodeHandler {
	return &TaxCodeHandler{store: s, deps: deps}
}

func (h *TaxCodeHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/tax-codes", h.list)
	mux.HandleFunc("POST /global/tax-codes", h.create)
	mux.HandleFunc("DELETE /global/tax-codes/{id}", h.delete)
}

func (h *TaxCodeHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	if isHTMX(r) {
		h.deps.renderTempl(w, r, lookup.TaxCodesTable(items))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, lookup.TaxCodesListPage(pg, items))
}

func (h *TaxCodeHandler) create(w http.ResponseWriter, r *http.Request) {
	code := formStringRequired(r, "code")
	desc := r.FormValue("description")
	rate := formString(r, "rate")

	if code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	item, err := h.store.Create(r.Context(), code, desc, rate)
	if err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "tax_codes", item.ID, "INSERT", nil, item)
	h.deps.setFlash(w, "Tax code created")

	redirect(w, r, "/global/tax-codes")
}

func (h *TaxCodeHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, _ := h.store.GetByID(r.Context(), id)
	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "tax_codes", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Tax code deleted")

	redirectBack(w, r, "/global/tax-codes")
}

// Items Handler

type itemStore interface {
	List(ctx context.Context) ([]store.ItemRecord, error)
	GetByID(ctx context.Context, id int) (*store.ItemRecord, error)
	Create(ctx context.Context, item, description string, defaultAmount, calcType *string) (*store.ItemRecord, error)
	Delete(ctx context.Context, id int) error
}

type ItemHandler struct {
	store itemStore
	deps  *Deps
}

func NewItemHandler(s itemStore, deps *Deps) *ItemHandler {
	return &ItemHandler{store: s, deps: deps}
}

func (h *ItemHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/items", h.list)
	mux.HandleFunc("POST /global/items", h.create)
	mux.HandleFunc("DELETE /global/items/{id}", h.delete)
}

func (h *ItemHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	if isHTMX(r) {
		h.deps.renderTempl(w, r, lookup.ItemsTable(items))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, lookup.ItemsListPage(pg, items))
}

func (h *ItemHandler) create(w http.ResponseWriter, r *http.Request) {
	item := formStringRequired(r, "item")
	desc := r.FormValue("description")
	amount := formString(r, "default_amount")
	calcType := formString(r, "calc_type")

	if item == "" {
		http.Error(w, "Item code is required", http.StatusBadRequest)
		return
	}

	rec, err := h.store.Create(r.Context(), item, desc, amount, calcType)
	if err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "items", rec.ID, "INSERT", nil, rec)
	h.deps.setFlash(w, "Item created")

	redirect(w, r, "/global/items")
}

func (h *ItemHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, _ := h.store.GetByID(r.Context(), id)
	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "items", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Item deleted")

	redirectBack(w, r, "/global/items")
}
