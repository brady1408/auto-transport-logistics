package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/audit"
	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/geocode"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LoadboardService struct {
	pool           *pgxpool.Pool
	loadboardStore *store.LoadboardStore
	orderStore     *store.OrderStore
	vehicleStore   *store.VehicleStore
	companyStore   *store.CompanyStore
	orderSvc       *OrderService
	audit          *audit.Service
}

func NewLoadboardService(
	pool *pgxpool.Pool,
	loadboardStore *store.LoadboardStore,
	orderStore *store.OrderStore,
	vehicleStore *store.VehicleStore,
	companyStore *store.CompanyStore,
	orderSvc *OrderService,
	audit *audit.Service,
) *LoadboardService {
	return &LoadboardService{
		pool:           pool,
		loadboardStore: loadboardStore,
		orderStore:     orderStore,
		vehicleStore:   vehicleStore,
		companyStore:   companyStore,
		orderSvc:       orderSvc,
		audit:          audit,
	}
}

// PostToLoadboard creates a loadboard listing from an order.
func (s *LoadboardService) PostToLoadboard(ctx context.Context, orderID int, opts PostOpts) (*models.LoadboardListing, error) {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return nil, auth.ErrNoUser
	}

	// Load order (company-scoped)
	order, err := s.orderStore.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("load order: %w", err)
	}

	// Load vehicles for the order
	vehicles, err := s.vehicleStore.ListByOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("load vehicles: %w", err)
	}

	// Filter to waiting vehicles only
	var waitingVehicles []models.OrderVehicle
	for _, v := range vehicles {
		if v.Status == "Waiting" {
			waitingVehicles = append(waitingVehicles, v)
		}
	}
	if len(waitingVehicles) == 0 {
		return nil, fmt.Errorf("no waiting vehicles on order %s", order.OrderNumber)
	}

	// Load poster's company info
	company, err := s.companyStore.GetByID(ctx, user.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("load company: %w", err)
	}

	// Generate listing number
	listingNumber, err := s.loadboardStore.NextListingNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate listing number: %w", err)
	}

	listing := &models.LoadboardListing{
		PosterCompanyID:     user.CompanyID,
		PosterUserID:        user.ID,
		SourceOrderID:       orderID,
		ListingNumber:       listingNumber,
		Title:               opts.Title,
		OriginName:          order.LoadCustomerName,
		OriginCity:          order.LoadCity,
		OriginState:         order.LoadState,
		OriginZip:           order.LoadZip,
		DestName:            order.DropCustomerName,
		DestCity:            order.DropCity,
		DestState:           order.DropState,
		DestZip:             order.DropZip,
		CarrierPay:          opts.CarrierPay,
		PickupDateFrom:      opts.PickupDateFrom,
		PickupDateTo:        opts.PickupDateTo,
		DeliverDateFrom:     opts.DeliverDateFrom,
		DeliverDateTo:       opts.DeliverDateTo,
		VehicleCount:        len(waitingVehicles),
		EquipmentType:       order.EquipmentType,
		SpecialInstructions: opts.SpecialInstructions,
		AutoAccept:          opts.AutoAccept,
		Status:              "Posted",
		ExpiresAt:           opts.ExpiresAt,
		PosterCompanyName:   &company.CompanyName,
		PosterSCAC:          company.SCAC,
		PosterMCNumber:      company.MCNumber,
	}

	// Geocode origin and dest addresses (non-fatal, use city/state/zip only — name is a customer name)
	oLat, oLng, err := geocode.Geocode(ctx, "", derefStr(listing.OriginCity), derefStr(listing.OriginState), derefStr(listing.OriginZip))
	if err != nil {
		log.Printf("geocode origin for listing %s: %v", listing.ListingNumber, err)
	}
	listing.OriginLat, listing.OriginLng = oLat, oLng

	dLat, dLng, err := geocode.Geocode(ctx, "", derefStr(listing.DestCity), derefStr(listing.DestState), derefStr(listing.DestZip))
	if err != nil {
		log.Printf("geocode dest for listing %s: %v", listing.ListingNumber, err)
	}
	listing.DestLat, listing.DestLng = dLat, dLng

	if err := s.loadboardStore.CreateListing(ctx, listing); err != nil {
		return nil, fmt.Errorf("create listing: %w", err)
	}

	// Create listing vehicles (denormalized snapshot)
	listingVehicles := make([]models.LoadboardListingVehicle, len(waitingVehicles))
	for i, v := range waitingVehicles {
		listingVehicles[i] = models.LoadboardListingVehicle{
			ListingID:       listing.ID,
			SourceVehicleID: v.ID,
			VIN:             v.VIN,
			Year:            v.Year,
			Make:            v.Make,
			Model:           v.Model,
			Color:           v.Color,
			Weight:          v.Weight,
			Category:        v.Category,
			BodyStyle:       v.BodyStyle,
			Operable:        v.Operable,
			RunDrive:        v.RunDrive,
		}
	}
	if err := s.loadboardStore.CreateListingVehicles(ctx, listingVehicles); err != nil {
		return nil, fmt.Errorf("create listing vehicles: %w", err)
	}

	s.audit.Log(ctx, "loadboard_listings", listing.ID, "INSERT", nil, listing)

	return listing, nil
}

// ClaimListing handles a carrier claiming a listing.
func (s *LoadboardService) ClaimListing(ctx context.Context, listingID int, carrierNotes *string) (*models.LoadboardClaim, error) {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return nil, auth.ErrNoUser
	}

	// Load carrier's company info
	carrierCompany, err := s.companyStore.GetByID(ctx, user.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("load carrier company: %w", err)
	}

	// Begin transaction with row lock
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock the listing row
	listing, err := s.loadboardStore.GetByIDForUpdate(ctx, tx, listingID)
	if err != nil {
		return nil, fmt.Errorf("lock listing: %w", err)
	}

	if listing.Status != "Posted" {
		return nil, fmt.Errorf("listing %s is no longer available (status: %s)", listing.ListingNumber, listing.Status)
	}
	if listing.PosterCompanyID == user.CompanyID {
		return nil, fmt.Errorf("cannot claim your own listing")
	}

	// Determine claim status based on auto_accept
	claimStatus := "Pending"
	var acceptedAt *time.Time
	if listing.AutoAccept {
		claimStatus = "Accepted"
		now := time.Now()
		acceptedAt = &now
	}

	claim := &models.LoadboardClaim{
		ListingID:          listingID,
		CarrierCompanyID:   user.CompanyID,
		CarrierUserID:      user.ID,
		CarrierCompanyName: &carrierCompany.CompanyName,
		CarrierSCAC:        carrierCompany.SCAC,
		CarrierMCNumber:    carrierCompany.MCNumber,
		CarrierDOTNumber:   carrierCompany.DOTNumber,
		CarrierInsuranceExp: carrierCompany.InsuranceExpDate,
		AgreedPay:          listing.CarrierPay,
		VehicleCount:       listing.VehicleCount,
		Status:             claimStatus,
		CarrierNotes:       carrierNotes,
		AcceptedAt:         acceptedAt,
	}

	if err := s.loadboardStore.CreateClaim(ctx, tx, claim); err != nil {
		return nil, fmt.Errorf("create claim: %w", err)
	}

	// If auto-accept, import order into carrier's system and mark listing as Claimed
	if listing.AutoAccept {
		orderID, err := s.importOrderForCarrier(ctx, listing, carrierCompany, claim)
		if err != nil {
			return nil, fmt.Errorf("import order for carrier: %w", err)
		}
		if err := s.loadboardStore.UpdateClaimCarrierOrder(ctx, tx, claim.ID, orderID); err != nil {
			return nil, fmt.Errorf("update claim carrier order: %w", err)
		}
		claim.CarrierOrderID = &orderID

		if err := s.loadboardStore.UpdateListingStatusTx(ctx, tx, listingID, "Claimed"); err != nil {
			return nil, fmt.Errorf("update listing status: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim: %w", err)
	}

	// Sync order counts for the imported order (outside tx, non-critical)
	if claim.CarrierOrderID != nil {
		carrierCtx := auth.SetUser(ctx, auth.ContextUser{
			ID:        user.ID,
			Username:  user.Username,
			Role:      user.Role,
			CompanyID: carrierCompany.ID,
		})
		_ = s.orderSvc.SyncOrderCounts(carrierCtx, *claim.CarrierOrderID)
	}

	s.audit.Log(ctx, "loadboard_claims", claim.ID, "INSERT", nil, claim)

	return claim, nil
}

// AcceptClaim allows a poster to accept a pending claim.
func (s *LoadboardService) AcceptClaim(ctx context.Context, claimID int) error {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return auth.ErrNoUser
	}

	// Begin transaction first, then read with row lock to prevent races
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Read claim within tx
	claim, err := s.loadboardStore.GetClaimByID(ctx, claimID)
	if err != nil {
		return fmt.Errorf("get claim: %w", err)
	}
	if claim.Status != "Pending" {
		return fmt.Errorf("claim is not pending")
	}

	// Lock listing row to prevent concurrent accept/cancel
	listing, err := s.loadboardStore.GetByIDForUpdate(ctx, tx, claim.ListingID)
	if err != nil {
		return fmt.Errorf("get listing: %w", err)
	}
	if listing.PosterCompanyID != user.CompanyID {
		return fmt.Errorf("only the poster can accept claims")
	}

	// Load carrier company info
	carrierCompany, err := s.companyStore.GetByID(ctx, claim.CarrierCompanyID)
	if err != nil {
		return fmt.Errorf("load carrier company: %w", err)
	}

	// Import order into carrier's system (uses pool, not tx)
	orderID, err := s.importOrderForCarrier(ctx, listing, carrierCompany, claim)
	if err != nil {
		return fmt.Errorf("import order for carrier: %w", err)
	}

	if err := s.loadboardStore.UpdateClaimCarrierOrder(ctx, tx, claimID, orderID); err != nil {
		return fmt.Errorf("update claim carrier order: %w", err)
	}

	if err := s.loadboardStore.UpdateClaimStatusTx(ctx, tx, claimID, "Accepted"); err != nil {
		return fmt.Errorf("update claim status: %w", err)
	}

	if err := s.loadboardStore.UpdateListingStatusTx(ctx, tx, claim.ListingID, "Claimed"); err != nil {
		return fmt.Errorf("update listing status: %w", err)
	}

	// Reject all other pending claims on this listing (bulk update within tx)
	if _, err := tx.Exec(ctx,
		"UPDATE loadboard_claims SET status = 'Rejected', rejected_at = NOW(), poster_notes = 'Another claim accepted', updated_at = NOW() WHERE listing_id = $1 AND id != $2 AND status = 'Pending'",
		claim.ListingID, claimID); err != nil {
		return fmt.Errorf("reject other claims: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit accept: %w", err)
	}

	// Sync order counts for the imported order
	carrierCtx := auth.SetUser(ctx, auth.ContextUser{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CompanyID: carrierCompany.ID,
	})
	_ = s.orderSvc.SyncOrderCounts(carrierCtx, orderID)

	s.audit.Log(ctx, "loadboard_claims", claimID, "UPDATE", nil, map[string]string{"action": "accept"})

	return nil
}

// RejectClaim allows a poster to reject a pending claim.
func (s *LoadboardService) RejectClaim(ctx context.Context, claimID int, reason *string) error {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return auth.ErrNoUser
	}

	claim, err := s.loadboardStore.GetClaimByID(ctx, claimID)
	if err != nil {
		return fmt.Errorf("get claim: %w", err)
	}
	if claim.Status != "Pending" {
		return fmt.Errorf("claim is not pending (status: %s)", claim.Status)
	}

	listing, err := s.loadboardStore.GetByID(ctx, claim.ListingID)
	if err != nil {
		return fmt.Errorf("get listing: %w", err)
	}
	if listing.PosterCompanyID != user.CompanyID {
		return fmt.Errorf("only the poster can reject claims")
	}

	if err := s.loadboardStore.UpdateClaimStatus(ctx, claimID, "Rejected"); err != nil {
		return fmt.Errorf("reject claim: %w", err)
	}

	// If reason provided, store as poster_notes
	if reason != nil {
		if _, err := s.pool.Exec(ctx, "UPDATE loadboard_claims SET poster_notes = $1 WHERE id = $2", *reason, claimID); err != nil {
			log.Printf("store reject reason for claim %d: %v", claimID, err)
		}
	}

	s.audit.Log(ctx, "loadboard_claims", claimID, "UPDATE", nil, map[string]string{"action": "reject"})

	return nil
}

// CancelListing cancels a listing and auto-rejects any pending claims.
func (s *LoadboardService) CancelListing(ctx context.Context, listingID int) error {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return auth.ErrNoUser
	}

	// Use transaction + row lock to prevent races with concurrent claim acceptance
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	listing, err := s.loadboardStore.GetByIDForUpdate(ctx, tx, listingID)
	if err != nil {
		return fmt.Errorf("get listing: %w", err)
	}
	if listing.PosterCompanyID != user.CompanyID {
		return fmt.Errorf("only the poster can cancel listings")
	}
	if listing.Status != "Posted" {
		return fmt.Errorf("listing is not in Posted status")
	}

	// Check for accepted claims
	claims, err := s.loadboardStore.ListClaimsOnListing(ctx, listingID)
	if err != nil {
		return fmt.Errorf("list claims: %w", err)
	}
	for _, c := range claims {
		if c.Status == "Accepted" {
			return fmt.Errorf("cannot cancel listing with accepted claims")
		}
	}

	// Bulk-reject pending claims within tx
	if _, err := tx.Exec(ctx,
		"UPDATE loadboard_claims SET status = 'Rejected', rejected_at = NOW(), poster_notes = 'Listing cancelled', updated_at = NOW() WHERE listing_id = $1 AND status = 'Pending'",
		listingID); err != nil {
		return fmt.Errorf("reject pending claims: %w", err)
	}

	if err := s.loadboardStore.UpdateListingStatusTx(ctx, tx, listingID, "Cancelled"); err != nil {
		return fmt.Errorf("cancel listing: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit cancel: %w", err)
	}

	s.audit.Log(ctx, "loadboard_listings", listingID, "UPDATE", nil, map[string]string{"action": "cancel"})

	return nil
}

// CancelClaim allows a carrier to cancel their pending claim.
func (s *LoadboardService) CancelClaim(ctx context.Context, claimID int) error {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return auth.ErrNoUser
	}

	claim, err := s.loadboardStore.GetClaimByID(ctx, claimID)
	if err != nil {
		return fmt.Errorf("get claim: %w", err)
	}
	if claim.CarrierCompanyID != user.CompanyID {
		return fmt.Errorf("only the carrier can cancel their claim")
	}
	if claim.Status != "Pending" {
		return fmt.Errorf("can only cancel pending claims (status: %s)", claim.Status)
	}

	if err := s.loadboardStore.UpdateClaimStatus(ctx, claimID, "Cancelled"); err != nil {
		return fmt.Errorf("cancel claim: %w", err)
	}

	s.audit.Log(ctx, "loadboard_claims", claimID, "UPDATE", nil, map[string]string{"action": "cancel"})

	return nil
}

// MarkPickedUp is called by the carrier to confirm vehicle pickup.
// Updates claim → PickedUp, carrier's vehicles → Loaded, poster's vehicles → Scheduled.
func (s *LoadboardService) MarkPickedUp(ctx context.Context, claimID int) error {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return auth.ErrNoUser
	}

	claim, err := s.loadboardStore.GetClaimByID(ctx, claimID)
	if err != nil {
		return fmt.Errorf("get claim: %w", err)
	}
	if claim.CarrierCompanyID != user.CompanyID {
		return fmt.Errorf("only the carrier can mark pickup")
	}
	if claim.Status != "Accepted" {
		return fmt.Errorf("can only mark pickup on accepted claims (status: %s)", claim.Status)
	}
	if claim.CarrierOrderID == nil {
		return fmt.Errorf("claim has no carrier order")
	}

	listing, err := s.loadboardStore.GetByID(ctx, claim.ListingID)
	if err != nil {
		return fmt.Errorf("get listing: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`UPDATE loadboard_claims SET status = 'PickedUp', picked_up_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND status = 'Accepted' AND deleted_at IS NULL`,
		claimID)
	if err != nil {
		return fmt.Errorf("update claim status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("claim %d is no longer in Accepted status", claimID)
	}

	// Carrier's order vehicles: Waiting → Loaded
	if _, err := tx.Exec(ctx,
		`UPDATE order_vehicles SET status = 'Loaded', updated_at = NOW()
		 WHERE order_id = $1 AND company_id = $2 AND status = 'Waiting' AND deleted_at IS NULL`,
		*claim.CarrierOrderID, claim.CarrierCompanyID); err != nil {
		return fmt.Errorf("update carrier vehicles: %w", err)
	}

	// Poster's source vehicles: Waiting → Scheduled
	if _, err := tx.Exec(ctx,
		`UPDATE order_vehicles ov SET status = 'Scheduled', updated_at = NOW()
		 FROM loadboard_listing_vehicles llv
		 WHERE llv.listing_id = $1 AND ov.id = llv.source_vehicle_id
		   AND ov.company_id = $2 AND ov.status = 'Waiting' AND ov.deleted_at IS NULL`,
		listing.ID, listing.PosterCompanyID); err != nil {
		return fmt.Errorf("update poster vehicles: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit pickup: %w", err)
	}

	s.audit.Log(ctx, "loadboard_claims", claimID, "UPDATE", nil, map[string]string{"action": "pickup"})
	return nil
}

// MarkDelivered is called by the carrier to confirm delivery.
// Updates claim → Delivered, listing → Completed, both sides' vehicles → Delivered.
func (s *LoadboardService) MarkDelivered(ctx context.Context, claimID int) error {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return auth.ErrNoUser
	}

	claim, err := s.loadboardStore.GetClaimByID(ctx, claimID)
	if err != nil {
		return fmt.Errorf("get claim: %w", err)
	}
	if claim.CarrierCompanyID != user.CompanyID {
		return fmt.Errorf("only the carrier can mark delivery")
	}
	if claim.Status != "PickedUp" {
		return fmt.Errorf("can only mark delivery on picked-up claims (status: %s)", claim.Status)
	}
	if claim.CarrierOrderID == nil {
		return fmt.Errorf("claim has no carrier order")
	}

	listing, err := s.loadboardStore.GetByID(ctx, claim.ListingID)
	if err != nil {
		return fmt.Errorf("get listing: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`UPDATE loadboard_claims SET status = 'Delivered', delivered_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND status = 'PickedUp' AND deleted_at IS NULL`,
		claimID)
	if err != nil {
		return fmt.Errorf("update claim status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("claim %d is no longer in PickedUp status", claimID)
	}

	if err := s.loadboardStore.UpdateListingStatusTx(ctx, tx, claim.ListingID, "Completed"); err != nil {
		return fmt.Errorf("update listing status: %w", err)
	}

	// Carrier's order vehicles: Loaded → Delivered
	if _, err := tx.Exec(ctx,
		`UPDATE order_vehicles SET status = 'Delivered', updated_at = NOW()
		 WHERE order_id = $1 AND company_id = $2 AND status = 'Loaded' AND deleted_at IS NULL`,
		*claim.CarrierOrderID, claim.CarrierCompanyID); err != nil {
		return fmt.Errorf("update carrier vehicles: %w", err)
	}

	// Poster's source vehicles: Scheduled → Delivered
	if _, err := tx.Exec(ctx,
		`UPDATE order_vehicles ov SET status = 'Delivered', updated_at = NOW()
		 FROM loadboard_listing_vehicles llv
		 WHERE llv.listing_id = $1 AND ov.id = llv.source_vehicle_id
		   AND ov.company_id = $2 AND ov.status = 'Scheduled' AND ov.deleted_at IS NULL`,
		listing.ID, listing.PosterCompanyID); err != nil {
		return fmt.Errorf("update poster vehicles: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delivery: %w", err)
	}

	s.audit.Log(ctx, "loadboard_claims", claimID, "UPDATE", nil, map[string]string{"action": "delivered"})
	return nil
}

// ReportNoShow is called by the poster when the carrier fails to pick up.
// Reverts claim → NoShow, listing → Posted, poster's vehicles → Waiting.
func (s *LoadboardService) ReportNoShow(ctx context.Context, claimID int) error {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return auth.ErrNoUser
	}

	claim, err := s.loadboardStore.GetClaimByID(ctx, claimID)
	if err != nil {
		return fmt.Errorf("get claim: %w", err)
	}
	if claim.Status != "Accepted" {
		return fmt.Errorf("can only report no-show on accepted claims (status: %s)", claim.Status)
	}

	listing, err := s.loadboardStore.GetByID(ctx, claim.ListingID)
	if err != nil {
		return fmt.Errorf("get listing: %w", err)
	}
	if listing.PosterCompanyID != user.CompanyID {
		return fmt.Errorf("only the poster can report no-show")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Mark claim as NoShow with reason in poster_notes
	result, err := tx.Exec(ctx,
		`UPDATE loadboard_claims SET status = 'NoShow', cancelled_at = NOW(),
		 poster_notes = 'No-show reported by poster', updated_at = NOW()
		 WHERE id = $1 AND status = 'Accepted' AND deleted_at IS NULL`,
		claimID)
	if err != nil {
		return fmt.Errorf("mark no-show: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("claim %d is no longer in Accepted status", claimID)
	}

	if err := s.loadboardStore.UpdateListingStatusTx(ctx, tx, claim.ListingID, "Posted"); err != nil {
		return fmt.Errorf("relist: %w", err)
	}

	// Revert poster's source vehicles: Scheduled → Waiting
	if _, err := tx.Exec(ctx,
		`UPDATE order_vehicles ov SET status = 'Waiting', updated_at = NOW()
		 FROM loadboard_listing_vehicles llv
		 WHERE llv.listing_id = $1 AND ov.id = llv.source_vehicle_id
		   AND ov.company_id = $2 AND ov.status = 'Scheduled' AND ov.deleted_at IS NULL`,
		listing.ID, listing.PosterCompanyID); err != nil {
		return fmt.Errorf("revert poster vehicles: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit no-show: %w", err)
	}

	s.audit.Log(ctx, "loadboard_claims", claimID, "UPDATE", nil, map[string]string{"action": "no-show"})
	return nil
}

// ExpireListings runs the expiration check.
func (s *LoadboardService) ExpireListings(ctx context.Context) (int, error) {
	return s.loadboardStore.ExpireListings(ctx)
}

// importOrderForCarrier creates an order and vehicles in the carrier's system.
// Must be called within a transaction.
func (s *LoadboardService) importOrderForCarrier(ctx context.Context, listing *models.LoadboardListing, carrierCompany *models.Company, claim *models.LoadboardClaim) (int, error) {
	// Create a context with the carrier's company ID
	carrierCtx := auth.SetUser(ctx, auth.ContextUser{
		ID:        claim.CarrierUserID,
		Username:  "loadboard-import",
		Role:      "user",
		CompanyID: carrierCompany.ID,
	})

	// Generate order number in carrier's system
	orderNumber, err := s.orderStore.NextOrderNumber(carrierCtx)
	if err != nil {
		return 0, fmt.Errorf("next order number: %w", err)
	}

	// Build the order using listing data
	now := time.Now()
	comment := fmt.Sprintf("Imported from loadboard listing %s", listing.ListingNumber)
	order := &models.Order{
		OrderNumber:      orderNumber,
		Active:           true,
		BillCustomerName: listing.PosterCompanyName,
		LoadCustomerName: listing.OriginName,
		LoadCity:         listing.OriginCity,
		LoadState:        listing.OriginState,
		LoadZip:          listing.OriginZip,
		DropCustomerName: listing.DestName,
		DropCity:         listing.DestCity,
		DropState:        listing.DestState,
		DropZip:          listing.DestZip,
		Comments:         &comment,
		TotalCharge:      &listing.CarrierPay,
		CreateDate:       &now,
		EstPickupDate:    listing.PickupDateFrom,
		EstDeliverDate:   listing.DeliverDateFrom,
		EquipmentType:    listing.EquipmentType,
	}

	// Create order using pool (the store uses auth context for company_id)
	if err := s.orderStore.Create(carrierCtx, order); err != nil {
		return 0, fmt.Errorf("create carrier order: %w", err)
	}

	// Load listing vehicles and create in carrier's system
	listingVehicles, err := s.loadboardStore.GetListingVehicles(ctx, listing.ID)
	if err != nil {
		return 0, fmt.Errorf("get listing vehicles: %w", err)
	}

	for _, lv := range listingVehicles {
		vehicle := &models.OrderVehicle{
			OrderID:   order.ID,
			Active:    true,
			VIN:       lv.VIN,
			Year:      lv.Year,
			Make:      lv.Make,
			Model:     lv.Model,
			Color:     lv.Color,
			Weight:    lv.Weight,
			Category:  lv.Category,
			BodyStyle: lv.BodyStyle,
			Operable:  lv.Operable,
			RunDrive:  lv.RunDrive,
			Status:    "Waiting",
		}
		if err := s.vehicleStore.Create(carrierCtx, vehicle); err != nil {
			return 0, fmt.Errorf("create carrier vehicle: %w", err)
		}
	}

	return order.ID, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// PostOpts contains options for posting to the loadboard.
type PostOpts struct {
	Title               string
	CarrierPay          string
	PickupDateFrom      *time.Time
	PickupDateTo        *time.Time
	DeliverDateFrom     *time.Time
	DeliverDateTo       *time.Time
	SpecialInstructions *string
	AutoAccept          bool
	ExpiresAt           *time.Time
}

