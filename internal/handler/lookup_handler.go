package handler

import (
	"net/http"

	"github.com/brady1408/atlinks/internal/handler/components/lookup"
	"github.com/brady1408/atlinks/internal/store"
)

type LookupHandler struct {
	store    *store.LookupStore
	deps     *Deps
	basePath string
	title    string
}

func NewLookupHandler(deps *Deps, lookupStore *store.LookupStore, basePath, title string) *LookupHandler {
	return &LookupHandler{
		store:    lookupStore,
		deps:     deps,
		basePath: basePath,
		title:    title,
	}
}

func (h *LookupHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET "+h.basePath, h.list)
	mux.HandleFunc("POST "+h.basePath, h.create)
	mux.HandleFunc("PUT "+h.basePath+"/{id}", h.update)
	mux.HandleFunc("DELETE "+h.basePath+"/{id}", h.delete)
}

func (h *LookupHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, lookup.Table(items, h.basePath))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, lookup.ListPage(pg, items, h.title, h.basePath))
}

func (h *LookupHandler) create(w http.ResponseWriter, r *http.Request) {
	code := formStringRequired(r, "code")
	desc := r.FormValue("description")

	if code == "" {
		http.Error(w, "Code is required", http.StatusBadRequest)
		return
	}

	item, err := h.store.Create(r.Context(), code, desc)
	if err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), h.store.TableName(), item.ID, "INSERT", nil, item)
	setFlash(w, h.title+" entry created")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", h.basePath)
		return
	}
	http.Redirect(w, r, h.basePath, http.StatusSeeOther)
}

func (h *LookupHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	code := formStringRequired(r, "code")
	desc := r.FormValue("description")

	if err := h.store.Update(r.Context(), id, code, desc); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), h.store.TableName(), id, "UPDATE", old, map[string]any{"code": code, "description": desc})
	setFlash(w, h.title+" entry updated")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", h.basePath)
		return
	}
	http.Redirect(w, r, h.basePath, http.StatusSeeOther)
}

func (h *LookupHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), h.store.TableName(), id, "DELETE", old, nil)
	setFlash(w, h.title+" entry deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", h.basePath)
		return
	}
	http.Redirect(w, r, h.basePath, http.StatusSeeOther)
}
