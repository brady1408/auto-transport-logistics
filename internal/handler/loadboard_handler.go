package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/handler/components/loadboard"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/service"
)

type loadboardStoreInterface interface {
	ListAvailable(ctx context.Context, f models.LoadboardFilter, excludeCompanyID int) (*models.LoadboardListResult, error)
	GetByID(ctx context.Context, id int) (*models.LoadboardListing, error)
	GetListingVehicles(ctx context.Context, listingID int) ([]models.LoadboardListingVehicle, error)
	ListMyListings(ctx context.Context, companyID int, f models.LoadboardFilter) (*models.LoadboardListResult, error)
	ListMyClaims(ctx context.Context, companyID int, f models.LoadboardFilter) (*models.LoadboardClaimListResult, error)
	ListClaimsOnListing(ctx context.Context, listingID int) ([]models.LoadboardClaim, error)
	GetClaimByID(ctx context.Context, id int) (*models.LoadboardClaim, error)
}

type loadboardOrderStore interface {
	GetByID(ctx context.Context, id int) (*models.Order, error)
}

type loadboardVehicleStore interface {
	ListByOrder(ctx context.Context, orderID int) ([]models.OrderVehicle, error)
}

type LoadboardHandler struct {
	store        loadboardStoreInterface
	orderStore   loadboardOrderStore
	vehicleStore loadboardVehicleStore
	svc          *service.LoadboardService
	deps         *Deps
}

func NewLoadboardHandler(
	store loadboardStoreInterface,
	orderStore loadboardOrderStore,
	vehicleStore loadboardVehicleStore,
	svc *service.LoadboardService,
	deps *Deps,
) *LoadboardHandler {
	return &LoadboardHandler{
		store:        store,
		orderStore:   orderStore,
		vehicleStore: vehicleStore,
		svc:          svc,
		deps:         deps,
	}
}

func (h *LoadboardHandler) Register(mux *http.ServeMux) {
	// Browse available listings (cross-company)
	mux.HandleFunc("GET /loadboard", h.browse)
	mux.HandleFunc("GET /loadboard/post/{orderID}", h.postForm)
	mux.HandleFunc("POST /loadboard/post/{orderID}", h.postCreate)
	mux.HandleFunc("GET /loadboard/my-listings", h.myListings)
	mux.HandleFunc("GET /loadboard/my-listings/{id}", h.myListingShow)
	mux.HandleFunc("POST /loadboard/my-listings/{id}/cancel", h.cancelListing)
	mux.HandleFunc("GET /loadboard/my-claims", h.myClaims)
	mux.HandleFunc("GET /loadboard/my-claims/{id}", h.myClaimShow)
	mux.HandleFunc("POST /loadboard/my-claims/{id}/complete", h.completeClaim)
	mux.HandleFunc("POST /loadboard/my-claims/{id}/cancel", h.cancelClaim)
	mux.HandleFunc("POST /loadboard/claims/{id}/accept", h.acceptClaim)
	mux.HandleFunc("POST /loadboard/claims/{id}/reject", h.rejectClaim)
	// These must be last to avoid matching the above
	mux.HandleFunc("GET /loadboard/{id}", h.show)
	mux.HandleFunc("POST /loadboard/{id}/claim", h.claim)
}

func (h *LoadboardHandler) browse(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	filter := models.LoadboardFilter{
		Search:      r.URL.Query().Get("search"),
		OriginState: r.URL.Query().Get("origin_state"),
		DestState:   r.URL.Query().Get("dest_state"),
		MinPay:      r.URL.Query().Get("min_pay"),
		MaxPay:      r.URL.Query().Get("max_pay"),
		Page:        intParam(r, "page", 1),
		PageSize:    25,
	}

	result, err := h.store.ListAvailable(r.Context(), filter, user.CompanyID)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, loadboard.Table(*result))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, loadboard.BrowsePage(pg, *result, filter))
}

func (h *LoadboardHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listing, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Listing not found", http.StatusNotFound)
		return
	}

	vehicles, err := h.store.GetListingVehicles(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}

	canClaim := listing.Status == "Posted" && listing.PosterCompanyID != user.CompanyID

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, loadboard.ShowPage(pg, listing, vehicles, canClaim))
}

func (h *LoadboardHandler) postForm(w http.ResponseWriter, r *http.Request) {
	orderID, err := parsePathID(r, "orderID")
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := h.orderStore.GetByID(r.Context(), orderID)
	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	vehicles, err := h.vehicleStore.ListByOrder(r.Context(), orderID)
	if err != nil {
		serverError(w, err)
		return
	}

	// Count waiting vehicles
	waitingCount := 0
	for _, v := range vehicles {
		if v.Status == "Waiting" {
			waitingCount++
		}
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, loadboard.PostFormPage(pg, order, waitingCount, ""))
}

func (h *LoadboardHandler) postCreate(w http.ResponseWriter, r *http.Request) {
	orderID, err := parsePathID(r, "orderID")
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	opts := service.PostOpts{
		Title:               formStringRequired(r, "title"),
		CarrierPay:          formStringRequired(r, "carrier_pay"),
		PickupDateFrom:      formDate(r, "pickup_date_from"),
		PickupDateTo:        formDate(r, "pickup_date_to"),
		DeliverDateFrom:     formDate(r, "deliver_date_from"),
		DeliverDateTo:       formDate(r, "deliver_date_to"),
		SpecialInstructions: formString(r, "special_instructions"),
		AutoAccept:          formBool(r, "auto_accept"),
		ExpiresAt:           formDateTime(r, "expires_at"),
	}

	if opts.Title == "" {
		opts.Title = "Vehicle Transport"
	}
	if opts.CarrierPay == "" {
		opts.CarrierPay = "0"
	}

	listing, err := h.svc.PostToLoadboard(r.Context(), orderID, opts)
	if err != nil {
		log.Printf("post to loadboard: %v", err)
		order, _ := h.orderStore.GetByID(r.Context(), orderID)
		vehicles, _ := h.vehicleStore.ListByOrder(r.Context(), orderID)
		waitingCount := 0
		for _, v := range vehicles {
			if v.Status == "Waiting" {
				waitingCount++
			}
		}
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, loadboard.PostFormPage(pg, order, waitingCount, err.Error()))
		return
	}

	h.deps.setFlash(w, fmt.Sprintf("Listed as %s on the loadboard", listing.ListingNumber))
	redirect(w, r, "/loadboard/my-listings")
}

func (h *LoadboardHandler) claim(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	carrierNotes := formString(r, "carrier_notes")

	claim, err := h.svc.ClaimListing(r.Context(), id, carrierNotes)
	if err != nil {
		log.Printf("claim listing: %v", err)
		h.deps.setFlash(w, "Failed to claim listing")
		redirect(w, r, fmt.Sprintf("/loadboard/%d", id))
		return
	}

	if claim.Status == "Accepted" {
		h.deps.setFlash(w, "Claim accepted! Order imported to your system.")
	} else {
		h.deps.setFlash(w, "Claim submitted. Waiting for poster approval.")
	}
	redirect(w, r, "/loadboard/my-claims")
}

func (h *LoadboardHandler) myListings(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	filter := models.LoadboardFilter{
		Search:   r.URL.Query().Get("search"),
		Status:   r.URL.Query().Get("status"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.store.ListMyListings(r.Context(), user.CompanyID, filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, loadboard.MyListingsTable(*result))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, loadboard.MyListingsPage(pg, *result, filter))
}

func (h *LoadboardHandler) myListingShow(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listing, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Listing not found", http.StatusNotFound)
		return
	}
	if listing.PosterCompanyID != user.CompanyID {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	vehicles, err := h.store.GetListingVehicles(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}

	claims, err := h.store.ListClaimsOnListing(r.Context(), id)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, loadboard.MyListingShowPage(pg, listing, vehicles, claims))
}

func (h *LoadboardHandler) cancelListing(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.CancelListing(r.Context(), id); err != nil {
		log.Printf("cancel listing: %v", err)
		h.deps.setFlash(w, "Failed to cancel")
		redirect(w, r, fmt.Sprintf("/loadboard/my-listings/%d", id))
		return
	}

	h.deps.setFlash(w, "Listing cancelled")
	redirect(w, r, "/loadboard/my-listings")
}

func (h *LoadboardHandler) myClaims(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	filter := models.LoadboardFilter{
		Search:   r.URL.Query().Get("search"),
		Status:   r.URL.Query().Get("status"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}

	result, err := h.store.ListMyClaims(r.Context(), user.CompanyID, filter)
	if err != nil {
		serverError(w, err)
		return
	}

	if isHTMX(r) {
		h.deps.renderTempl(w, r, loadboard.MyClaimsTable(*result))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, loadboard.MyClaimsPage(pg, *result, filter))
}

func (h *LoadboardHandler) myClaimShow(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	claim, err := h.store.GetClaimByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Claim not found", http.StatusNotFound)
		return
	}
	if claim.CarrierCompanyID != user.CompanyID {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	listing, err := h.store.GetByID(r.Context(), claim.ListingID)
	if err != nil {
		serverError(w, err)
		return
	}

	vehicles, err := h.store.GetListingVehicles(r.Context(), claim.ListingID)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, loadboard.MyClaimShowPage(pg, claim, listing, vehicles))
}

func (h *LoadboardHandler) completeClaim(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.CompleteClaim(r.Context(), id); err != nil {
		log.Printf("complete claim: %v", err)
		h.deps.setFlash(w, "Failed to complete")
		redirect(w, r, fmt.Sprintf("/loadboard/my-claims/%d", id))
		return
	}

	h.deps.setFlash(w, "Claim marked as completed")
	redirect(w, r, "/loadboard/my-claims")
}

func (h *LoadboardHandler) cancelClaim(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.CancelClaim(r.Context(), id); err != nil {
		log.Printf("cancel claim: %v", err)
		h.deps.setFlash(w, "Failed to cancel")
		redirect(w, r, fmt.Sprintf("/loadboard/my-claims/%d", id))
		return
	}

	h.deps.setFlash(w, "Claim cancelled")
	redirect(w, r, "/loadboard/my-claims")
}

func (h *LoadboardHandler) acceptClaim(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.AcceptClaim(r.Context(), id); err != nil {
		log.Printf("accept claim: %v", err)
		h.deps.setFlash(w, "Failed to accept claim")

		// Redirect back to the listing that this claim belongs to
		claim, claimErr := h.store.GetClaimByID(r.Context(), id)
		if claimErr == nil {
			redirect(w, r, fmt.Sprintf("/loadboard/my-listings/%d", claim.ListingID))
		} else {
			redirect(w, r, "/loadboard/my-listings")
		}
		return
	}

	h.deps.setFlash(w, "Claim accepted and order imported to carrier's system")
	redirect(w, r, "/loadboard/my-listings")
}

func (h *LoadboardHandler) rejectClaim(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	reason := formString(r, "reason")
	if err := h.svc.RejectClaim(r.Context(), id, reason); err != nil {
		log.Printf("reject claim: %v", err)
		h.deps.setFlash(w, "Failed to reject claim")
		redirect(w, r, "/loadboard/my-listings")
		return
	}

	h.deps.setFlash(w, "Claim rejected")

	claim, claimErr := h.store.GetClaimByID(r.Context(), id)
	if claimErr == nil {
		redirect(w, r, fmt.Sprintf("/loadboard/my-listings/%d", claim.ListingID))
	} else {
		redirect(w, r, "/loadboard/my-listings")
	}
}

// formDateTime parses a datetime-local input value.
func formDateTime(r *http.Request, key string) *time.Time {
	v := r.FormValue(key)
	if v == "" {
		return nil
	}
	// datetime-local format: 2006-01-02T15:04
	t, err := time.Parse("2006-01-02T15:04", v)
	if err != nil {
		// Also try date-only
		t, err = time.Parse("2006-01-02", v)
		if err != nil {
			return nil
		}
	}
	return &t
}
