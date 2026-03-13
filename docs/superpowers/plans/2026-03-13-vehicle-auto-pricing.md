# Vehicle Auto-Pricing Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Auto-populate vehicle transport_amt from zone pricing when vehicles are added to an order, with zones derived from load/drop contacts.

**Architecture:** Server-side pricing lookup. Order gets origin_zone (from load contact) and destination_zone (from drop contact). Zone pricing table maps zone pairs to rates. When a vehicle is added, the rate auto-fills transport_amt. HTMX handles UI updates.

**Tech Stack:** Go 1.22+, PostgreSQL 16, pgx/v5, templ, HTMX, Connect-RPC (protobuf), MCP

**Spec:** `docs/superpowers/specs/2026-03-13-vehicle-auto-pricing-design.md`

**Note:** The spec lists customer-related files (`customer.go`, `customer_store.go`, `customer_handler.go`, `customers/form.templ`, `customers/show.templ`, `customer.proto`, `customer_server.go`) for adding a `zone` field. This is **already implemented** — `models.Customer` already has `Zone *string`, the store/handler/templates/proto already support it. No customer changes are needed.

---

## Chunk 1: Database Migration & Model Changes

### Task 1: Migration — rename zone, add destination_zone

**Files:**
- Create: `internal/database/migrations/037_order_zones.sql`

- [ ] **Step 1: Write migration**

```sql
-- +goose Up

-- Rename zone to origin_zone on orders
ALTER TABLE orders RENAME COLUMN zone TO origin_zone;

-- Add destination zone
ALTER TABLE orders ADD COLUMN destination_zone VARCHAR(20);

-- Update indexes that reference the old column name
DROP INDEX IF EXISTS idx_orders_zone;
CREATE INDEX idx_orders_origin_zone ON orders (origin_zone);
CREATE INDEX idx_orders_destination_zone ON orders (destination_zone);

-- +goose Down

DROP INDEX IF EXISTS idx_orders_destination_zone;
DROP INDEX IF EXISTS idx_orders_origin_zone;
ALTER TABLE orders DROP COLUMN IF EXISTS destination_zone;
ALTER TABLE orders RENAME COLUMN origin_zone TO zone;
CREATE INDEX idx_orders_zone ON orders (zone, order_number);
```

- [ ] **Step 2: Verify migration locally**

Run: `make migrate-up`
Expected: Migration 037 applies successfully

- [ ] **Step 3: Commit**

```bash
git add internal/database/migrations/037_order_zones.sql
git commit -m "feat: migration 037 — rename orders.zone to origin_zone, add destination_zone"
```

### Task 2: Update Order model and store

**Files:**
- Modify: `internal/models/order.go:10` — rename Zone to OriginZone, add DestinationZone
- Modify: `internal/models/order.go:89-101` — update OrderFilter (Zone → OriginZone, add DestinationZone)
- Modify: `internal/store/order_store.go:22-32` — update orderSortConfig zone mapping
- Modify: `internal/store/order_store.go:34-50` — update orderColumns (zone → origin_zone, add destination_zone)
- Modify: `internal/store/order_store.go:52-86` — update scanOrder (Zone → OriginZone, add DestinationZone)
- Modify: `internal/store/order_store.go:98-100` — update zone filter in List()
- Modify: `internal/store/order_store.go:348-376` — update StatusSummary zone reference

- [ ] **Step 1: Update Order struct in models/order.go**

Change line 10:
```go
// Old: Zone *string
OriginZone      *string
DestinationZone *string
```

- [ ] **Step 2: Update OrderFilter struct**

Change line 91:
```go
// Old: Zone string
OriginZone      string
DestinationZone string
```

- [ ] **Step 3: Update orderColumns in store**

Replace `zone` with `origin_zone, destination_zone` in the column list constant.

- [ ] **Step 4: Update scanOrder()**

Add `&o.OriginZone` and `&o.DestinationZone` in place of `&o.Zone` in the Scan call. Ensure column order matches.

- [ ] **Step 5: Update orderSortConfig**

Change `"zone": "zone"` to `"zone": "origin_zone"` at line 26.

- [ ] **Step 6: Update List() zone filter**

Replace:
```go
if f.Zone != "" {
    qb.Add("zone = ?", f.Zone)
}
```
With:
```go
if f.OriginZone != "" {
    qb.Add("origin_zone = ?", f.OriginZone)
}
if f.DestinationZone != "" {
    qb.Add("destination_zone = ?", f.DestinationZone)
}
```

- [ ] **Step 7: Update StatusSummary()**

Change `zone` references to `origin_zone` in the OrderStatusRow struct and GROUP BY clause.

- [ ] **Step 8: Update all handler references to order.Zone**

Search for `order.Zone`, `o.Zone`, `filter.Zone` in handler files and update to `OriginZone`/`DestinationZone` as appropriate. Key files:
- `internal/handler/order_handler.go:308` — bindOrderForm: `Zone: formString(r, "zone")` → `OriginZone: formString(r, "origin_zone"), DestinationZone: formString(r, "destination_zone")`
- `internal/handler/order_handler.go` — list handler filter binding: `Zone` → `OriginZone`

- [ ] **Step 9: Build and verify**

Run: `go build ./...`
Expected: Build succeeds (templates may fail until Task 3)

- [ ] **Step 10: Commit**

```bash
git add internal/models/order.go internal/store/order_store.go internal/handler/order_handler.go
git commit -m "feat: rename order Zone to OriginZone, add DestinationZone"
```

### Task 3: Update order templates

**Files:**
- Modify: `internal/handler/components/orders/form.templ:52-54` — replace zone field with origin_zone + destination_zone
- Modify: `internal/handler/components/orders/show.templ:48-49` — display both zones
- Modify: `internal/handler/components/orders/list.templ` — update filter if zone is shown
- Modify: `internal/handler/components/orders/table.templ` — update any zone column references

- [ ] **Step 1: Update order form**

Replace the single zone input (lines 52-54) with two fields:
```templ
<div class="form-group">
    <label for="origin_zone">Origin Zone</label>
    <input type="text" id="origin_zone" name="origin_zone" class="form-control"
        value={ components.Deref(order.OriginZone) } maxlength="20"/>
</div>
<div class="form-group">
    <label for="destination_zone">Destination Zone</label>
    <input type="text" id="destination_zone" name="destination_zone" class="form-control"
        value={ components.Deref(order.DestinationZone) } maxlength="20"/>
</div>
```

- [ ] **Step 2: Update order show page**

Replace zone display (lines 48-49) with:
```templ
<span class="detail-label">Origin Zone:</span>
<span>{ components.Deref(order.OriginZone) }</span>
<span class="detail-label">Destination Zone:</span>
<span>{ components.Deref(order.DestinationZone) }</span>
```

- [ ] **Step 3: Update table/list if zone column exists**

Check `table.templ` for zone column display. Update `Deref(o.Zone)` → `Deref(o.OriginZone)` if present.

- [ ] **Step 4: Generate templ and build**

Run: `templ generate && go build ./...`
Expected: Clean build

- [ ] **Step 5: Commit**

```bash
git add internal/handler/components/orders/
git commit -m "feat: update order templates for origin/destination zones"
```

## Chunk 2: Zone Pricing Lookup & Auto-fill

### Task 4: Add GetByZones() to ZonePricingStore

**Files:**
- Modify: `internal/store/zone_store.go` — add GetByZones method after existing CRUD methods

- [ ] **Step 1: Add GetByZones method**

Add after the existing Delete method (~line 211):
```go
// GetByZones returns zone pricing for the given origin/destination zone pair.
func (s *ZonePricingStore) GetByZones(ctx context.Context, zoneA, zoneB string) (*models.ZonePricing, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	var zp models.ZonePricing
	err = s.pool.QueryRow(ctx,
		`SELECT id, company_id, legacy_id, zone_a, zone_b, description, amount, miles, transport_days, ship_to, created_at, updated_at
		 FROM zone_pricing WHERE company_id = $1 AND zone_a = $2 AND zone_b = $3 AND deleted_at IS NULL`,
		companyID, zoneA, zoneB,
	).Scan(&zp.ID, &zp.CompanyID, &zp.LegacyID, &zp.ZoneA, &zp.ZoneB, &zp.Description,
		&zp.Amount, &zp.Miles, &zp.TransportDays, &zp.ShipTo, &zp.CreatedAt, &zp.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get zone pricing by zones %s→%s: %w", zoneA, zoneB, err)
	}
	return &zp, nil
}
```

This matches the exact column list and scan order from `GetByID` (line 152-155 of zone_store.go).

- [ ] **Step 2: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 3: Commit**

```bash
git add internal/store/zone_store.go
git commit -m "feat: add ZonePricingStore.GetByZones() lookup method"
```

### Task 5: Zone pricing lookup endpoint

**Files:**
- Modify: `internal/handler/zone_handler.go` — add lookup endpoint
- Modify: `internal/handler/zone_handler.go` — register new route

- [ ] **Step 1: Add lookup handler method**

Add a new method to ZoneHandler:
```go
func (h *ZoneHandler) pricingLookup(w http.ResponseWriter, r *http.Request) {
	origin := r.URL.Query().Get("origin_zone")
	destination := r.URL.Query().Get("destination_zone")
	if origin == "" || destination == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	zp, err := h.pricingStore.GetByZones(r.Context(), origin, destination)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Render a small partial with the rate hint
	h.deps.renderTempl(w, r, zones.RateHint(zp))
}
```

Note: The endpoint uses `origin_zone` and `destination_zone` as query param names to match the form field names, so HTMX `hx-include` works directly without param remapping.

- [ ] **Step 2: Create RateHint templ component**

Create or add to an existing zones component file:
```templ
templ RateHint(zp *models.ZonePricing) {
	if zp != nil && zp.Amount != nil {
		<span class="rate-hint">Zone rate: ${ components.Deref(zp.Amount) }</span>
	}
}
```

- [ ] **Step 3: Register route**

Add to the ZoneHandler Register method:
```go
mux.HandleFunc("GET /api/zone-pricing/lookup", h.pricingLookup)
```

Ensure the ZoneHandler has access to the ZonePricingStore (check constructor).

- [ ] **Step 4: Build and verify**

Run: `templ generate && go build ./...`
Expected: Clean build

- [ ] **Step 5: Commit**

```bash
git add internal/handler/zone_handler.go internal/handler/components/
git commit -m "feat: add zone pricing lookup endpoint with rate hint partial"
```

### Task 6: Wire zone auto-fill on order form

**Files:**
- Modify: `internal/handler/components/orders/form.templ` — add HTMX triggers on zone fields for rate hint
- Modify: `internal/handler/components/orders/form.templ` — add rate hint target div

- [ ] **Step 1: Add rate hint display and HTMX trigger to zone fields**

On the origin_zone and destination_zone inputs, add HTMX attributes to trigger a lookup when both are filled. The lookup endpoint reads `origin_zone` and `destination_zone` query params (matching field names), so `hx-include` works directly:
```templ
<div class="form-group">
    <label for="origin_zone">Origin Zone</label>
    <input type="text" id="origin_zone" name="origin_zone" class="form-control"
        value={ components.Deref(order.OriginZone) } maxlength="20"
        hx-get="/api/zone-pricing/lookup"
        hx-trigger="change delay:300ms"
        hx-target="#rate-hint"
        hx-include="#destination_zone"
    />
</div>
<div class="form-group">
    <label for="destination_zone">Destination Zone</label>
    <input type="text" id="destination_zone" name="destination_zone" class="form-control"
        value={ components.Deref(order.DestinationZone) } maxlength="20"
        hx-get="/api/zone-pricing/lookup"
        hx-trigger="change delay:300ms"
        hx-target="#rate-hint"
        hx-include="#origin_zone"
    />
</div>
<div id="rate-hint"></div>
```

Each field includes the other via `hx-include` so the lookup always receives both `origin_zone` and `destination_zone` as query params.

- [ ] **Step 2: Build and test manually**

Run: `templ generate && go build ./...`
Test: Start the server, navigate to an order form, enter zones, verify the rate hint appears.

- [ ] **Step 3: Commit**

```bash
git add internal/handler/components/orders/form.templ
git commit -m "feat: wire HTMX zone pricing lookup on order form"
```

### Task 7: Auto-populate vehicle transport_amt

**Files:**
- Modify: `internal/handler/vehicle_handler.go` — in the create/newForm method, look up zone pricing and pre-fill transport_amt
- Modify: `internal/handler/components/orders/vehicle_form.templ` — ensure transport_amt field shows pre-filled value

- [ ] **Step 1: Update vehicle new form handler**

In the vehicle handler's `newForm` method, after loading the parent order, look up zone pricing:
```go
// Look up zone pricing for pre-fill
var defaultTransportAmt *string
order, err := h.orderStore.GetByID(r.Context(), orderID)
if err == nil && order.OriginZone != nil && order.DestinationZone != nil {
    zp, err := h.zonePricingStore.GetByZones(r.Context(), *order.OriginZone, *order.DestinationZone)
    if err == nil && zp.Amount != nil {
        defaultTransportAmt = zp.Amount
    }
}
```

Pass `defaultTransportAmt` to the vehicle form template (set on vehicle.TransportAmt if it's a new vehicle).

- [ ] **Step 2: Ensure vehicle handler has ZonePricingStore dependency**

Add `zonePricingStore` to the VehicleHandler struct and constructor. Update `cmd/server/main.go` to pass it.

- [ ] **Step 3: Ensure vehicle handler has access to order store**

The vehicle handler needs to read the parent order's zones. Check if it already has an orderStore dependency; if not, add it.

- [ ] **Step 4: Build and test**

Run: `go build ./...`
Test: Create an order with zones that match a zone pricing entry, then add a vehicle — transport_amt should be pre-filled.

- [ ] **Step 5: Commit**

```bash
git add internal/handler/vehicle_handler.go cmd/server/main.go
git commit -m "feat: auto-populate vehicle transport_amt from zone pricing"
```

## Chunk 3: Zone Auto-fill from Contacts & Update Prompt

### Task 8: Auto-fill zones from load/drop contacts on order form

**Files:**
- Modify: `internal/handler/order_handler.go` — add endpoint or modify create to auto-fill zones from customer
- Modify: `internal/handler/components/orders/form.templ` — add HTMX on load/drop customer selects to fetch zone

- [ ] **Step 1: Add customer zone lookup endpoint**

Add to order handler or customer handler:
```go
func (h *OrderHandler) customerZone(w http.ResponseWriter, r *http.Request) {
    customerID, err := strconv.Atoi(r.URL.Query().Get("customer_id"))
    if err != nil {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    customer, err := h.customerStore.GetByID(r.Context(), customerID)
    if err != nil || customer.Zone == nil {
        w.WriteHeader(http.StatusNoContent)
        return
    }
    // Return the zone value as plain text for HTMX to swap into the input
    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte(*customer.Zone))
}
```

Register: `mux.HandleFunc("GET /api/customer-zone", h.customerZone)`

- [ ] **Step 2: Wire HTMX on load/drop customer fields**

When the load customer changes, fetch their zone and put it in the origin_zone input. When drop customer changes, fetch zone into destination_zone. This likely requires `hx-get` on the customer select/input with `hx-target` pointing to the zone input.

The exact implementation depends on how customer selection currently works (text input with autocomplete vs. dropdown). Inspect the current customer selection pattern and adapt.

- [ ] **Step 3: Build and test**

Run: `templ generate && go build ./...`
Test: Select a load customer with a zone → origin_zone auto-fills. Select a drop customer → destination_zone auto-fills.

- [ ] **Step 4: Commit**

```bash
git add internal/handler/order_handler.go internal/handler/components/orders/form.templ
git commit -m "feat: auto-fill order zones from load/drop contact default zones"
```

### Task 9: Zone change prompt for existing vehicles

**Files:**
- Modify: `internal/handler/order_handler.go` — in update method, detect zone change and prompt
- Create or modify: templ component for the confirmation prompt

- [ ] **Step 1: Detect zone change in order update handler**

In the order update handler, after binding the form into the `order` struct, compare old vs new zones:
```go
old, _ := h.store.GetByID(r.Context(), id)
// ... bind form into 'order' ...
zoneChanged := derefStr(old.OriginZone) != derefStr(order.OriginZone) ||
               derefStr(old.DestinationZone) != derefStr(order.DestinationZone)
```

If zones changed and vehicles exist, check for the `update_vehicles` form param:
- **Not present**: Save the order first (zones are updated), then re-render the form with the confirmation banner. Pass the **saved** order to the form template so all fields are populated correctly.
- **"yes"**: Update vehicles with matching old rate to new rate.
- **"no"**: Skip vehicle updates, redirect to show page.

This means the order is always saved on the first submit — the confirmation only controls whether vehicle prices are also updated.

- [ ] **Step 2: Add vehicle bulk update for zone rate change**

```go
// Update vehicles that still have the old rate
if r.FormValue("update_vehicles") == "yes" {
    oldZP, _ := h.zonePricingStore.GetByZones(r.Context(), derefStr(old.OriginZone), derefStr(old.DestinationZone))
    newZP, _ := h.zonePricingStore.GetByZones(r.Context(), derefStr(order.OriginZone), derefStr(order.DestinationZone))
    if oldZP != nil && newZP != nil {
        // Update vehicles where transport_amt matches old rate
        h.vehicleStore.UpdateTransportAmtByRate(r.Context(), id, oldZP.Amount, newZP.Amount)
    }
}
```

- [ ] **Step 3: Add UpdateTransportAmtByRate to vehicle store**

```go
func (s *VehicleStore) UpdateTransportAmtByRate(ctx context.Context, orderID int, oldAmt, newAmt *string) error {
    companyID, _ := auth.GetCompanyID(ctx)
    _, err := s.pool.Exec(ctx,
        `UPDATE order_vehicles SET transport_amt = $1
         WHERE order_id = $2 AND company_id = $3 AND transport_amt = $4 AND deleted_at IS NULL`,
        newAmt, orderID, companyID, oldAmt)
    return err
}
```

- [ ] **Step 4: Create confirmation UI**

Use a two-pass approach in the order update handler:

1. First submit: handler detects zone change + vehicles exist + no `update_vehicles` param → re-render the form with an alert banner and two hidden fields:
   - `<input type="hidden" name="update_vehicles" value="yes"/>` on the "Yes, update" button
   - `<input type="hidden" name="update_vehicles" value="no"/>` on the "No, keep existing" button

```templ
templ ZoneChangeConfirm(vehicleCount int, newRate string) {
    <div class="alert alert-warning">
        <p>Zone rate changed to ${ newRate }. Update { strconv.Itoa(vehicleCount) } existing vehicles?</p>
        <button type="submit" name="update_vehicles" value="yes" class="btn btn-primary btn-sm">Yes, update</button>
        <button type="submit" name="update_vehicles" value="no" class="btn btn-secondary btn-sm">No, keep existing</button>
    </div>
}
```

2. Second submit: handler reads `update_vehicles` param and proceeds accordingly (update matching vehicles or skip).

- [ ] **Step 5: Build and test**

Run: `templ generate && go build ./...`
Test: Edit an order with vehicles, change zones, verify prompt appears, verify vehicle amounts update on confirmation.

- [ ] **Step 6: Commit**

```bash
git add internal/handler/order_handler.go internal/store/vehicle_store.go internal/handler/components/
git commit -m "feat: prompt to update vehicle pricing when order zones change"
```

## Chunk 4: Proto / Connect-RPC / MCP Updates

### Task 10: Update proto definitions

**Files:**
- Modify: `proto/atlinks/v1/order.proto:21` — rename zone to origin_zone, add destination_zone
- Modify: `proto/atlinks/v1/order.proto:90` — update ListOrdersRequest zone filter
- Modify: `proto/atlinks/v1/order.proto:114` — update CreateOrderRequest
- Modify: `proto/atlinks/v1/order.proto` — update UpdateOrderRequest

- [ ] **Step 1: Update Order message**

```protobuf
// Old: optional string zone = 4;
optional string origin_zone = 4;     // reuse existing field number
optional string destination_zone = 68; // next after updated_at = 67
```

- [ ] **Step 2: Update ListOrdersRequest**

```protobuf
// Old: optional string zone = 3;
optional string origin_zone = 3;      // reuse existing field number
optional string destination_zone = 9; // next after date_to = 8
```

- [ ] **Step 3: Update CreateOrderRequest**

```protobuf
// Old: optional string zone = 3;
optional string origin_zone = 3;       // reuse existing field number
optional string destination_zone = 56; // next after dim_weight = 55
```

- [ ] **Step 4: Update UpdateOrderRequest**

```protobuf
// Old: optional string zone = 3;
optional string origin_zone = 3;       // reuse existing field number
optional string destination_zone = 56; // next after dim_weight = 55
```

- [ ] **Step 5: Regenerate protobuf code**

Run: `make proto`
Expected: Generated code in `internal/gen/atlinks/v1/` updates cleanly

- [ ] **Step 6: Commit**

```bash
git add proto/atlinks/v1/order.proto internal/gen/
git commit -m "feat: update order proto for origin/destination zones"
```

### Task 11: Update Connect-RPC converters

**Files:**
- Modify: `internal/connectrpc/order_server.go` — update all zone references in converters

- [ ] **Step 1: Update order-to-proto converter**

Find where `Zone` is mapped and change to `OriginZone` / `DestinationZone`. Follow the existing converter pattern (likely uses `sp()` helper for string pointers).

- [ ] **Step 2: Update proto-to-order converter**

Update the reverse mapping in Create/Update handlers.

- [ ] **Step 3: Update list filter mapping**

Map `OriginZone` and `DestinationZone` from the ListOrdersRequest.

- [ ] **Step 4: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 5: Commit**

```bash
git add internal/connectrpc/order_server.go
git commit -m "feat: update Connect-RPC order converters for zone fields"
```

### Task 12: Update MCP tools

**Files:**
- Modify: `cmd/atlinks-mcp/tools_order.go` — update zone parameter names and descriptions

- [ ] **Step 1: Update ListOrders tool**

Change `zone` parameter to `origin_zone`, add `destination_zone` parameter. Update descriptions.

- [ ] **Step 2: Update CreateOrder tool**

Replace `zone` with `origin_zone` and `destination_zone` parameters and request mappings.

- [ ] **Step 3: Update UpdateOrder tool**

Same changes as CreateOrder.

- [ ] **Step 4: Update GetOrder tool display**

If the get/list display formats zone info, update to show both zones.

- [ ] **Step 5: Build and verify**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 6: Commit**

```bash
git add cmd/atlinks-mcp/tools_order.go
git commit -m "feat: update MCP order tools for origin/destination zones"
```

## Chunk 5: Final Integration & Deploy

### Task 13: End-to-end verification

- [ ] **Step 1: Run full build**

Run: `go build ./...`
Expected: Clean build with no errors

- [ ] **Step 2: Run migrations on local dev**

Run: `make migrate-up`
Expected: Migration 037 applies

- [ ] **Step 3: Manual smoke test**

1. Create a zone pricing entry (SLC → DEN, $150)
2. Set up two customers with zones (load customer = SLC, drop customer = DEN)
3. Create a new order, select those customers → zones auto-fill
4. Verify rate hint shows "$150.00"
5. Add a vehicle → transport_amt pre-fills with $150
6. Edit order, change destination zone → prompt to update vehicles
7. Verify MCP tools work with new zone fields

- [ ] **Step 4: Deploy**

Run: `./scripts/deploy.sh`
Expected: Migration applies on production, app starts healthy

- [ ] **Step 5: Commit any final fixes**

```bash
git add -A
git commit -m "fix: final adjustments from smoke testing"
```
