package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/handler/components/creditmemos"
	"github.com/brady1408/auto-transport-logistics/internal/models"
)

type creditMemoStore interface {
	List(ctx context.Context, f models.CreditMemoFilter) (*models.CreditMemoListResult, error)
	GetByID(ctx context.Context, id int) (*models.CreditMemo, error)
	Create(ctx context.Context, cm *models.CreditMemo) error
	Update(ctx context.Context, cm *models.CreditMemo) error
	Delete(ctx context.Context, id int) error
	NextCreditNumber(ctx context.Context) (string, error)
}

type CreditMemoHandler struct {
	store creditMemoStore
	deps  *Deps
}

func NewCreditMemoHandler(s creditMemoStore, deps *Deps) *CreditMemoHandler {
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
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, creditmemos.Table(*result))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, creditmemos.ListPage(pg, *result, filter))
}

func (h *CreditMemoHandler) newForm(w http.ResponseWriter, r *http.Request) {
	creditNum, _ := h.store.NextCreditNumber(r.Context())
	now := time.Now()
	status := "Pending"
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, creditmemos.FormPage(pg, &models.CreditMemo{
		CreditNumber: creditNum,
		CreditDate:   &now,
		Status:       &status,
	}, true, ""))
}

func (h *CreditMemoHandler) create(w http.ResponseWriter, r *http.Request) {
	cm := bindCreditMemoForm(r)

	if cm.CreditNumber == "" {
		num, err := h.store.NextCreditNumber(r.Context())
		if err != nil {
			log.Printf("generate credit number: %v", err)
			pg := h.deps.pageContext(w, r)
			h.deps.renderTempl(w, r, creditmemos.FormPage(pg, cm, true, "Failed to generate credit number"))
			return
		}
		cm.CreditNumber = num
	}

	if err := h.store.Create(r.Context(), cm); err != nil {
		log.Printf("create credit memo: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, creditmemos.FormPage(pg, cm, true, "Failed to create credit memo"))
		return
	}

	h.deps.Audit.Log(r.Context(), "credit_memos", cm.ID, "INSERT", nil, cm)
	h.deps.setFlash(w, "Credit memo created successfully")

	redirect(w, r, "/accounting/credit-memos")
}

func (h *CreditMemoHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	cm, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Credit memo not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, creditmemos.ShowPage(pg, cm))
}

func (h *CreditMemoHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	cm, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Credit memo not found", http.StatusNotFound)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, creditmemos.FormPage(pg, cm, false, ""))
}

func (h *CreditMemoHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
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
		log.Printf("update credit memo: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, creditmemos.FormPage(pg, cm, false, "Failed to update credit memo"))
		return
	}

	h.deps.Audit.Log(r.Context(), "credit_memos", cm.ID, "UPDATE", old, cm)
	h.deps.setFlash(w, "Credit memo updated successfully")

	redirect(w, r, "/accounting/credit-memos")
}

func (h *CreditMemoHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Credit memo not found", http.StatusNotFound)
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}

	h.deps.Audit.Log(r.Context(), "credit_memos", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Credit memo deleted")

	redirectBack(w, r, "/accounting/credit-memos")
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
