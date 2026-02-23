package handler

import (
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type CreditMemoHandler struct {
	store *store.CreditMemoStore
	deps  *Deps
}

func NewCreditMemoHandler(s *store.CreditMemoStore, deps *Deps) *CreditMemoHandler {
	return &CreditMemoHandler{store: s, deps: deps}
}

func (h *CreditMemoHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /accounting/credit-memos", h.list)
	mux.HandleFunc("GET /accounting/credit-memos/new", h.newForm)
	mux.HandleFunc("POST /accounting/credit-memos", h.create)
	mux.HandleFunc("GET /accounting/credit-memos/{id}", h.show)
	mux.HandleFunc("GET /accounting/credit-memos/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /accounting/credit-memos/{id}", h.update)
	mux.HandleFunc("DELETE /accounting/credit-memos/{id}", h.delete)
}

func (h *CreditMemoHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.CreditMemoFilter{
		Search:     r.URL.Query().Get("search"),
		CustomerID: r.URL.Query().Get("customer_id"),
		Status:     r.URL.Query().Get("status"),
		Page:       intParam(r, "page", 1),
		PageSize:   25,
	}

	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Result": result,
		"Filter": filter,
	}

	if isHTMX(r) {
		h.deps.renderPartial(w, "credit_memo_table", data)
		return
	}
	h.deps.render(w, r, "credit_memo_list.html", data)
}

func (h *CreditMemoHandler) newForm(w http.ResponseWriter, r *http.Request) {
	creditNum, _ := h.store.NextCreditNumber(r.Context())
	now := time.Now()
	status := "Pending"
	h.deps.render(w, r, "credit_memo_form.html", map[string]any{
		"CreditMemo": &models.CreditMemo{
			CreditNumber: creditNum,
			CreditDate:   &now,
			Status:       &status,
		},
		"IsNew": true,
	})
}

func (h *CreditMemoHandler) create(w http.ResponseWriter, r *http.Request) {
	cm := bindCreditMemoForm(r)

	if cm.CreditNumber == "" {
		num, err := h.store.NextCreditNumber(r.Context())
		if err != nil {
			h.deps.render(w, r, "credit_memo_form.html", map[string]any{
				"CreditMemo": cm,
				"IsNew":      true,
				"Error":      "Failed to generate credit number: " + err.Error(),
			})
			return
		}
		cm.CreditNumber = num
	}

	if err := h.store.Create(r.Context(), cm); err != nil {
		h.deps.render(w, r, "credit_memo_form.html", map[string]any{
			"CreditMemo": cm,
			"IsNew":      true,
			"Error":      "Failed to create credit memo: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "credit_memos", cm.ID, "INSERT", nil, cm)
	setFlash(w, "Credit memo created successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/credit-memos")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/credit-memos", http.StatusSeeOther)
}

func (h *CreditMemoHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cm, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Credit memo not found", http.StatusNotFound)
		return
	}

	h.deps.render(w, r, "credit_memo_show.html", map[string]any{
		"CreditMemo": cm,
	})
}

func (h *CreditMemoHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cm, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Credit memo not found", http.StatusNotFound)
		return
	}

	h.deps.render(w, r, "credit_memo_form.html", map[string]any{
		"CreditMemo": cm,
		"IsNew":      false,
	})
}

func (h *CreditMemoHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Credit memo not found", http.StatusNotFound)
		return
	}

	cm := bindCreditMemoForm(r)
	cm.ID = id
	cm.CreditNumber = old.CreditNumber

	if err := h.store.Update(r.Context(), cm); err != nil {
		h.deps.render(w, r, "credit_memo_form.html", map[string]any{
			"CreditMemo": cm,
			"IsNew":      false,
			"Error":      "Failed to update credit memo: " + err.Error(),
		})
		return
	}

	h.deps.Audit.Log(r.Context(), "credit_memos", cm.ID, "UPDATE", old, cm)
	setFlash(w, "Credit memo updated successfully")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/credit-memos")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/credit-memos", http.StatusSeeOther)
}

func (h *CreditMemoHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Credit memo not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete credit memo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.deps.Audit.Log(r.Context(), "credit_memos", id, "DELETE", old, nil)
	setFlash(w, "Credit memo deleted")

	if isHTMX(r) {
		w.Header().Set("HX-Redirect", "/accounting/credit-memos")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/accounting/credit-memos", http.StatusSeeOther)
}

func bindCreditMemoForm(r *http.Request) *models.CreditMemo {
	return &models.CreditMemo{
		CreditNumber:   formStringRequired(r, "credit_number"),
		CustomerID:     formInt(r, "customer_id"),
		CustomerNumber: formString(r, "customer_number"),
		CustomerName:   formString(r, "customer_name"),
		InvoiceID:      formInt(r, "invoice_id"),
		InvoiceNumber:  formString(r, "invoice_number"),
		CreditDate:     formDate(r, "credit_date"),
		Amount:         formString(r, "amount"),
		Reason:         formString(r, "reason"),
		Status:         formString(r, "status"),
		Comments:       formString(r, "comments"),
	}
}
