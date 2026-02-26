# Loadboard Safety & Process Visibility Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add claim milestone tracking (Picked Up / Delivered), vehicle status propagation to both sides, a read-only subhauled loads section on the poster's order page, insurance warnings, and no-show handling.

**Architecture:** All features extend the existing `loadboard_claims` table with two new timestamps. Service methods handle vehicle propagation in the same transaction as the status change. No new tables required. The order show page gains a read-only section that queries loadboard claims by `source_order_id`.

**Tech Stack:** Go 1.22+, pgx/v5, templ, HTMX, PostgreSQL 16, goose migrations.

---

## Conventions (read before any task)

- **Migrations**: Create `internal/database/migrations/022_<name>.sql` with `-- +goose Up` and `-- +goose Down` sections
- **IDs**: Use `BIGSERIAL` / `bigint` (not SERIAL/INTEGER)
- **Soft deletes**: `deleted_at IS NULL` in all queries
- **company_id isolation**: Poster-owned vehicles use poster's company_id; carrier-owned use carrier's
- **templ compile**: After editing `.templ` files run `make generate` then `go build ./...`
- **Test build**: `go build ./...` after every task — catch compile errors early
- **Commit message style**: `feat: <description>` (lowercase, no attribution)

---

## Task 1: Migration — add milestone timestamps

**Files:**
- Create: `internal/database/migrations/022_loadboard_milestones.sql`

**Step 1: Write the migration**

```sql
-- +goose Up
ALTER TABLE loadboard_claims
  ADD COLUMN picked_up_at  TIMESTAMPTZ,
  ADD COLUMN delivered_at  TIMESTAMPTZ;

-- +goose Down
ALTER TABLE loadboard_claims
  DROP COLUMN IF EXISTS picked_up_at,
  DROP COLUMN IF EXISTS delivered_at;
```

**Step 2: Apply and verify**

```bash
make migrate-up
```

Expected: `OK   022_loadboard_milestones.sql`

**Step 3: Commit**

```bash
git add internal/database/migrations/022_loadboard_milestones.sql
git commit -m "feat: add picked_up_at and delivered_at to loadboard_claims"
```

---

## Task 2: Model + store column registration

The new columns must be reflected in the model and in every scan across the store.

**Files:**
- Modify: `internal/models/loadboard.go`
- Modify: `internal/store/loadboard_store.go`

**Step 1: Add fields to LoadboardClaim**

In `internal/models/loadboard.go`, add after `AcceptedAt`:

```go
PickedUpAt  *time.Time `json:"picked_up_at,omitempty"`
DeliveredAt *time.Time `json:"delivered_at,omitempty"`
```

**Step 2: Update claimColumnsAliased**

In `internal/store/loadboard_store.go`, replace the `claimColumnsAliased()` return value to append the two new columns:

```go
func claimColumnsAliased() string {
	return `c.id, c.listing_id, c.carrier_company_id, c.carrier_user_id,
	c.carrier_company_name, c.carrier_scac, c.carrier_mc_number, c.carrier_dot_number, c.carrier_insurance_exp,
	c.carrier_order_id, c.agreed_pay, c.vehicle_count, c.status,
	c.carrier_notes, c.poster_notes,
	c.accepted_at, c.rejected_at, c.cancelled_at, c.completed_at,
	c.picked_up_at, c.delivered_at,
	c.created_at, c.updated_at`
}
```

**Step 3: Update claimStatusDateCol**

Replace the existing `claimStatusDateCol` function to handle the new statuses and remove `Completed`:

```go
func claimStatusDateCol(status string) (string, error) {
	switch status {
	case "Accepted":
		return "accepted_at", nil
	case "Rejected":
		return "rejected_at", nil
	case "Cancelled":
		return "cancelled_at", nil
	case "NoShow":
		return "cancelled_at", nil
	case "PickedUp":
		return "picked_up_at", nil
	case "Delivered":
		return "delivered_at", nil
	default:
		return "", fmt.Errorf("unrecognized claim status: %q", status)
	}
}
```

**Step 4: Update the three Scan call sites**

There are three places in `loadboard_store.go` that scan claim rows. Each needs `&c.PickedUpAt, &c.DeliveredAt` added after `&c.CompletedAt`.

In `ListMyClaims` (around line 252), update the Scan to:
```go
&c.AcceptedAt, &c.RejectedAt, &c.CancelledAt, &c.CompletedAt,
&c.PickedUpAt, &c.DeliveredAt,
&c.CreatedAt, &c.UpdatedAt,
```

In `ListClaimsOnListing` (around line 300), same change.

In `GetClaimByID` (around line 327), same change.

**Step 5: Verify compile**

```bash
go build ./...
```

Expected: no errors.

**Step 6: Commit**

```bash
git add internal/models/loadboard.go internal/store/loadboard_store.go
git commit -m "feat: add PickedUpAt/DeliveredAt to loadboard claim model and scans"
```

---

## Task 3: Store — ListActiveClaimsForOrder

This new method powers the subhauled loads section on the poster's order page.

**Files:**
- Modify: `internal/store/loadboard_store.go`

**Step 1: Add the method at the end of loadboard_store.go**

```go
// ListActiveClaimsForOrder returns accepted/in-progress claims for a given source order.
// Used to display the subhauled loads section on the order show page.
// No company_id scoping: the caller already verified they own the order.
func (s *LoadboardStore) ListActiveClaimsForOrder(ctx context.Context, orderID int) ([]models.LoadboardClaim, error) {
	query := fmt.Sprintf(`SELECT %s, l.listing_number, l.title, l.status
		FROM loadboard_claims c
		JOIN loadboard_listings l ON l.id = c.listing_id
		WHERE l.source_order_id = $1
		  AND c.status IN ('Accepted', 'PickedUp', 'Delivered', 'NoShow')
		  AND c.deleted_at IS NULL
		  AND l.deleted_at IS NULL
		ORDER BY c.created_at DESC`,
		claimColumnsAliased())
	rows, err := s.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("list active claims for order %d: %w", orderID, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.LoadboardClaim, error) {
		var c models.LoadboardClaim
		if err := row.Scan(
			&c.ID, &c.ListingID, &c.CarrierCompanyID, &c.CarrierUserID,
			&c.CarrierCompanyName, &c.CarrierSCAC, &c.CarrierMCNumber, &c.CarrierDOTNumber, &c.CarrierInsuranceExp,
			&c.CarrierOrderID, &c.AgreedPay, &c.VehicleCount, &c.Status,
			&c.CarrierNotes, &c.PosterNotes,
			&c.AcceptedAt, &c.RejectedAt, &c.CancelledAt, &c.CompletedAt,
			&c.PickedUpAt, &c.DeliveredAt,
			&c.CreatedAt, &c.UpdatedAt,
			&c.ListingNumber, &c.ListingTitle, &c.ListingStatus,
		); err != nil {
			return models.LoadboardClaim{}, err
		}
		return c, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan active claim: %w", err)
	}
	return items, nil
}
```

**Step 2: Verify compile**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/store/loadboard_store.go
git commit -m "feat: add ListActiveClaimsForOrder to loadboard store"
```

---

## Task 4: Service — MarkPickedUp, MarkDelivered, ReportNoShow

These three methods handle the status transitions and vehicle propagation on both sides in a single transaction.

**Files:**
- Modify: `internal/service/loadboard_service.go`

**Step 1: Add MarkPickedUp**

Add after `CompleteClaim`:

```go
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

	if err := s.loadboardStore.UpdateClaimStatusTx(ctx, tx, claimID, "PickedUp"); err != nil {
		return fmt.Errorf("update claim status: %w", err)
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
```

**Step 2: Add MarkDelivered**

```go
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

	if err := s.loadboardStore.UpdateClaimStatusTx(ctx, tx, claimID, "Delivered"); err != nil {
		return fmt.Errorf("update claim status: %w", err)
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
```

**Step 3: Add ReportNoShow**

```go
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
	if _, err := tx.Exec(ctx,
		`UPDATE loadboard_claims SET status = 'NoShow', cancelled_at = NOW(),
		 poster_notes = 'No-show reported by poster', updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL`,
		claimID); err != nil {
		return fmt.Errorf("mark no-show: %w", err)
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
```

**Step 4: Verify compile**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add internal/service/loadboard_service.go
git commit -m "feat: add MarkPickedUp, MarkDelivered, ReportNoShow to loadboard service"
```

---

## Task 5: Handler — new routes, remove completeClaim, add companyStore

**Files:**
- Modify: `internal/handler/loadboard_handler.go`

**Step 1: Add loadboardCompanyStoreInterface**

Add a new interface at the top of `loadboard_handler.go` (after the existing interfaces):

```go
type loadboardCompanyStoreInterface interface {
	GetByID(ctx context.Context, id int) (*models.Company, error)
}
```

**Step 2: Add companyStore field to LoadboardHandler**

Update the struct and constructor:

```go
type LoadboardHandler struct {
	store        loadboardStoreInterface
	orderStore   loadboardOrderStore
	vehicleStore loadboardVehicleStore
	companyStore loadboardCompanyStoreInterface
	svc          *service.LoadboardService
	deps         *Deps
}

func NewLoadboardHandler(
	store loadboardStoreInterface,
	orderStore loadboardOrderStore,
	vehicleStore loadboardVehicleStore,
	companyStore loadboardCompanyStoreInterface,
	svc *service.LoadboardService,
	deps *Deps,
) *LoadboardHandler {
	return &LoadboardHandler{
		store:        store,
		orderStore:   orderStore,
		vehicleStore: vehicleStore,
		companyStore: companyStore,
		svc:          svc,
		deps:         deps,
	}
}
```

**Step 3: Update Register — replace completeClaim, add new routes**

In `Register`, remove:
```go
mux.HandleFunc("POST /loadboard/my-claims/{id}/complete", h.completeClaim)
```

Add:
```go
mux.HandleFunc("POST /loadboard/my-claims/{id}/pickup", h.pickupClaim)
mux.HandleFunc("POST /loadboard/my-claims/{id}/deliver", h.deliverClaim)
mux.HandleFunc("POST /loadboard/claims/{id}/no-show", h.noShowClaim)
```

**Step 4: Add the three new handlers, remove completeClaim**

Remove the existing `completeClaim` function entirely. Add in its place:

```go
func (h *LoadboardHandler) pickupClaim(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if err := h.svc.MarkPickedUp(r.Context(), id); err != nil {
		log.Printf("mark pickup: %v", err)
		h.deps.setFlash(w, "Failed to mark pickup")
		redirect(w, r, fmt.Sprintf("/loadboard/my-claims/%d", id))
		return
	}
	h.deps.setFlash(w, "Pickup confirmed — vehicles are now in transit")
	redirect(w, r, fmt.Sprintf("/loadboard/my-claims/%d", id))
}

func (h *LoadboardHandler) deliverClaim(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if err := h.svc.MarkDelivered(r.Context(), id); err != nil {
		log.Printf("mark delivered: %v", err)
		h.deps.setFlash(w, "Failed to mark delivery")
		redirect(w, r, fmt.Sprintf("/loadboard/my-claims/%d", id))
		return
	}
	h.deps.setFlash(w, "Delivery confirmed — load complete")
	redirect(w, r, "/loadboard/my-claims")
}

func (h *LoadboardHandler) noShowClaim(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	claim, err := h.store.GetClaimByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Claim not found", http.StatusNotFound)
		return
	}
	if err := h.svc.ReportNoShow(r.Context(), id); err != nil {
		log.Printf("report no-show: %v", err)
		h.deps.setFlash(w, "Failed to report no-show")
		redirect(w, r, fmt.Sprintf("/loadboard/my-listings/%d", claim.ListingID))
		return
	}
	h.deps.setFlash(w, "No-show reported — load has been relisted")
	redirect(w, r, fmt.Sprintf("/loadboard/my-listings/%d", claim.ListingID))
}
```

**Step 5: Update show handler to pass insurance info**

The `show` handler needs to load the viewer's company and compute insurance warning fields. Update the handler signature's template call:

```go
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

	// Load viewer's company for insurance warning (non-fatal)
	var viewerInsuranceExp *time.Time
	if company, err := h.companyStore.GetByID(r.Context(), user.CompanyID); err == nil {
		viewerInsuranceExp = company.InsuranceExpDate
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, loadboard.ShowPage(pg, listing, vehicles, canClaim, viewerInsuranceExp))
}
```

**Step 6: Wire companyStore in main.go**

Find the `NewLoadboardHandler` call in `cmd/server/main.go` and add `companyStore` as an argument. The `companyStore` variable is already declared earlier in main.go (it's used by other handlers). Pass it as the fourth argument.

**Step 7: Verify compile**

```bash
go build ./...
```

**Step 8: Commit**

```bash
git add internal/handler/loadboard_handler.go cmd/server/main.go
git commit -m "feat: add pickup/deliver/no-show handlers and insurance store to loadboard handler"
```

---

## Task 6: UI — my_claim_show.templ (carrier milestone buttons)

**Files:**
- Modify: `internal/handler/components/loadboard/my_claim_show.templ`

**Step 1: Replace Complete button with milestone buttons**

In the `btn-group` section, replace:
```templ
if claim.Status == "Accepted" {
    <button
        class="btn btn-success"
        hx-post={ fmt.Sprintf("/loadboard/my-claims/%d/complete", claim.ID) }
        hx-confirm="Mark this claim as completed?"
    >Mark Completed</button>
}
```

With:
```templ
if claim.Status == "Accepted" {
    <button
        class="btn btn-primary"
        hx-post={ fmt.Sprintf("/loadboard/my-claims/%d/pickup", claim.ID) }
        hx-confirm="Confirm all vehicles have been picked up?"
    >Mark Picked Up</button>
}
if claim.Status == "PickedUp" {
    <button
        class="btn btn-success"
        hx-post={ fmt.Sprintf("/loadboard/my-claims/%d/deliver", claim.ID) }
        hx-confirm="Confirm all vehicles have been delivered?"
    >Mark Delivered</button>
}
```

**Step 2: Add milestone timestamps to the Claim Details card**

After the `AcceptedAt` block, add:

```templ
if claim.PickedUpAt != nil {
    <div class="detail-row">
        <span class="detail-label">Picked Up:</span>
        <span>{ claim.PickedUpAt.Format("01/02/2006 3:04 PM") }</span>
    </div>
}
if claim.DeliveredAt != nil {
    <div class="detail-row">
        <span class="detail-label">Delivered:</span>
        <span>{ claim.DeliveredAt.Format("01/02/2006 3:04 PM") }</span>
    </div>
}
```

**Step 3: Compile templates**

```bash
make generate && go build ./...
```

**Step 4: Commit**

```bash
git add internal/handler/components/loadboard/my_claim_show.templ \
        internal/handler/components/loadboard/my_claim_show_templ.go
git commit -m "feat: replace Complete button with Picked Up / Delivered milestone buttons"
```

---

## Task 7: UI — my_listing_show.templ (no-show button + insurance badge)

**Files:**
- Modify: `internal/handler/components/loadboard/my_listing_show.templ`

**Step 1: Add insurance badge helper**

Add this template function at the bottom of the file (before the closing of the package):

```templ
templ insuranceBadge(exp *time.Time) {
	if exp == nil {
		<span class="badge badge-warning" title="No insurance date on file">⚠ None on file</span>
	} else if exp.Before(time.Now()) {
		<span class="badge badge-inactive" title={ "Expired: " + exp.Format("01/02/2006") }>✗ Expired</span>
	} else if exp.Before(time.Now().AddDate(0, 0, 30)) {
		<span class="badge badge-warning" title={ "Expiring: " + exp.Format("01/02/2006") }>⚠ Exp { exp.Format("01/02/2006") }</span>
	} else {
		<span class="badge badge-active" title={ "Valid until: " + exp.Format("01/02/2006") }>✓ Valid</span>
	}
}
```

Add `"time"` to the imports at the top.

**Step 2: Add insurance badge and No-Show button to each claim card**

In the claim card header span, add the insurance badge after the DOT# line:

```templ
<div>
    <span class="detail-label">Insurance:</span>
    @insuranceBadge(c.CarrierInsuranceExp)
</div>
```

In the button group, add the No-Show button after the Reject button:

```templ
if c.Status == "Accepted" {
    <button
        class="btn btn-sm btn-warning"
        hx-post={ fmt.Sprintf("/loadboard/claims/%d/no-show", c.ID) }
        hx-confirm="Report this carrier as a no-show? The load will be relisted and the claim will be closed."
    >Report No-Show</button>
}
```

**Step 3: Compile templates**

```bash
make generate && go build ./...
```

**Step 4: Commit**

```bash
git add internal/handler/components/loadboard/my_listing_show.templ \
        internal/handler/components/loadboard/my_listing_show_templ.go
git commit -m "feat: add no-show button and insurance badge to listing claims"
```

---

## Task 8: UI — show.templ (insurance warning banner for carrier)

**Files:**
- Modify: `internal/handler/components/loadboard/show.templ`

**Step 1: Update ShowPage signature**

Change:
```go
templ ShowPage(pg components.PageContext, listing *models.LoadboardListing, vehicles []models.LoadboardListingVehicle, canClaim bool)
```

To:
```go
templ ShowPage(pg components.PageContext, listing *models.LoadboardListing, vehicles []models.LoadboardListingVehicle, canClaim bool, viewerInsuranceExp *time.Time)
```

Add `"time"` to imports.

**Step 2: Add insurance warning banner**

Add this block just before the `if canClaim` Claim button (inside the page-header div):

```templ
if canClaim {
    @insuranceWarningBanner(viewerInsuranceExp)
}
```

Add the helper template at the bottom of the file:

```templ
templ insuranceWarningBanner(exp *time.Time) {
	if exp == nil {
		<div class="alert alert-warning" style="margin-bottom:1rem;">
			⚠ No insurance information on file for your company.
			<a href="/settings/company">Update your company profile</a> before claiming loads.
		</div>
	} else if exp.Before(time.Now()) {
		<div class="alert alert-danger" style="margin-bottom:1rem;">
			⚠ Your company's insurance on file expired { exp.Format("01/02/2006") }.
			<a href="/settings/company">Update your company profile</a> before claiming loads.
		</div>
	} else if exp.Before(time.Now().AddDate(0, 0, 30)) {
		<div class="alert alert-warning" style="margin-bottom:1rem;">
			⚠ Your company's insurance on file expires { exp.Format("01/02/2006") }.
			Consider <a href="/settings/company">updating your company profile</a>.
		</div>
	}
}
```

**Step 3: Compile and build**

```bash
make generate && go build ./...
```

**Step 4: Commit**

```bash
git add internal/handler/components/loadboard/show.templ \
        internal/handler/components/loadboard/show_templ.go
git commit -m "feat: add insurance warning banner to loadboard listing detail"
```

---

## Task 9: Order show page — Subhauled Loads section

**Files:**
- Modify: `internal/handler/order_handler.go`
- Modify: `internal/handler/components/orders/show.templ`

**Step 1: Add loadboardSubhaulStore interface to order_handler.go**

Near the top of `order_handler.go` where the other interfaces are defined, add:

```go
type loadboardSubhaulStore interface {
	ListActiveClaimsForOrder(ctx context.Context, orderID int) ([]models.LoadboardClaim, error)
}
```

**Step 2: Add field to OrderHandler**

Find the `OrderHandler` struct and add:
```go
loadboardStore loadboardSubhaulStore
```

Find `NewOrderHandler` and add the parameter and assignment. Check `main.go` to confirm the constructor signature — add `loadboardStore loadboardSubhaulStore` as a parameter and `loadboardStore: loadboardStore` in the struct literal.

**Step 3: Update show handler to fetch subhauled claims**

In `order_handler.go`, update the `show` function to fetch subhauled claims after fetching attachments:

```go
subhauledClaims, err := h.loadboardStore.ListActiveClaimsForOrder(r.Context(), id)
if err != nil {
    log.Printf("list subhauled claims for order %d: %v", id, err)
    subhauledClaims = nil
}

pg := h.deps.pageContext(w, r)
h.deps.renderTempl(w, r, orders.ShowPage(pg, o, atts, subhauledClaims))
```

**Step 4: Update ShowPage signature**

In `internal/handler/components/orders/show.templ`, change:

```go
templ ShowPage(pg components.PageContext, order *models.Order, atts []models.Attachment)
```

To:

```go
templ ShowPage(pg components.PageContext, order *models.Order, atts []models.Attachment, subhauledClaims []models.LoadboardClaim)
```

Add imports for `"fmt"` (already present) and `"github.com/brady1408/atlinks/internal/models"` if not already imported.

**Step 5: Add Subhauled Loads section to the template**

Add this section after the Attachments section and before the closing brace of the Layout block:

```templ
if len(subhauledClaims) > 0 {
    <h2 class="section-title">Subhauled Loads</h2>
    <div class="table-container">
        <table>
            <thead>
                <tr>
                    <th>Listing</th>
                    <th>Carrier</th>
                    <th>MC#</th>
                    <th>DOT#</th>
                    <th>Agreed Pay</th>
                    <th>Status</th>
                    <th>Picked Up</th>
                    <th>Delivered</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                for _, c := range subhauledClaims {
                    <tr>
                        <td>{ c.ListingNumber }</td>
                        <td>{ components.Deref(c.CarrierCompanyName) }</td>
                        <td>{ components.Deref(c.CarrierMCNumber) }</td>
                        <td>{ components.Deref(c.CarrierDOTNumber) }</td>
                        <td>${ c.AgreedPay }</td>
                        <td>@subhaulStatusBadge(c.Status)</td>
                        <td>
                            if c.PickedUpAt != nil {
                                { c.PickedUpAt.Format("01/02/2006") }
                            }
                        </td>
                        <td>
                            if c.DeliveredAt != nil {
                                { c.DeliveredAt.Format("01/02/2006") }
                            }
                        </td>
                        <td>
                            <a href={ templ.SafeURL(fmt.Sprintf("/loadboard/my-listings/%d", c.ListingID)) } class="btn btn-sm">View</a>
                        </td>
                    </tr>
                }
            </tbody>
        </table>
    </div>
}
```

Add the badge helper at the bottom of the file:

```templ
templ subhaulStatusBadge(status string) {
	switch status {
		case "Accepted":
			<span class="badge badge-waiting">Accepted</span>
		case "PickedUp":
			<span class="badge badge-loaded">In Transit</span>
		case "Delivered":
			<span class="badge badge-confirmed">Delivered</span>
		case "NoShow":
			<span class="badge badge-inactive">No-Show</span>
		default:
			<span class="badge">{ status }</span>
	}
}
```

**Step 6: Wire loadboardStore in main.go**

Find `NewOrderHandler(...)` in `cmd/server/main.go` and pass `loadboardStore` as the new argument. The `loadboardStore` variable is already declared earlier in main when setting up the loadboard handler.

**Step 7: Compile and build**

```bash
make generate && go build ./...
```

**Step 8: Commit**

```bash
git add internal/handler/order_handler.go \
        internal/handler/components/orders/show.templ \
        internal/handler/components/orders/show_templ.go \
        cmd/server/main.go
git commit -m "feat: add subhauled loads section to order show page"
```

---

## Final Verification

**Step 1: Full build**

```bash
go build ./...
```

**Step 2: Smoke test the happy path**

1. Log in as Company A — create an order, post it to the loadboard
2. Log in as Company B — browse the loadboard, claim the listing
3. Company A accepts the claim
4. Verify Company A's order show page now has a "Subhauled Loads" section showing Company B's claim as Accepted
5. Company B marks Picked Up — verify claim shows PickedUp, Company A's vehicles show Scheduled, Company B's vehicles show Loaded
6. Company B marks Delivered — verify claim shows Delivered, both sides' vehicles show Delivered
7. Re-run with a no-show: Company A reports No-Show on an Accepted claim — verify listing returns to Posted, vehicles revert to Waiting
8. Verify insurance warning banner shows on listing detail when Company B has no insurance on file

**Step 3: Deploy**

```bash
./scripts/deploy.sh
```
