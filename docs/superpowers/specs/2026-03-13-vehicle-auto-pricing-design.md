# Vehicle Auto-Pricing from Zone Pricing

**Date**: 2026-03-13
**Status**: Approved
**Related feedback**: #47 [5B-3] Vehicle Auto-Pricing

## Problem

Zone pricing exists as a lookup table (origin zone → destination zone → rate) but is disconnected from orders and vehicles. All pricing is manual entry. Dispatchers must remember rates or look them up separately.

## Solution

Auto-populate vehicle `transport_amt` from zone pricing when adding vehicles to an order. Zones are derived from the load (origin) and drop (destination) contacts, which each have a default zone assigned at the customer level.

## Out of Scope

- Weight/wheelbase-based pricing tiers (future — zone pricing will eventually have weight ranges)
- Fuel surcharge auto-calculation (future — potentially tied to real-time gas prices per zone)
- Renaming bill/load/drop customer terminology to customer/contacts
- Discount, tax, or total_charge auto-calculation

## Data Model Changes

### Customers table — add default zone

```sql
ALTER TABLE customers ADD COLUMN zone VARCHAR(20);
```

Each customer/contact can have a default zone (e.g., "SLC", "DEN"). Optional — not every customer needs one.

### Orders table — origin/destination zones

```sql
ALTER TABLE orders RENAME COLUMN zone TO origin_zone;
ALTER TABLE orders ADD COLUMN destination_zone VARCHAR(20);
```

Existing `zone` values are preserved as `origin_zone`. `destination_zone` starts NULL for existing orders.

### Zone pricing table — no changes

Already has `zone_a` (origin), `zone_b` (destination), `amount`. No schema changes needed.

### Order vehicles — no changes

`transport_amt` already exists on `order_vehicles`. We auto-populate it from zone pricing.

### Relationship flow

```
Load Contact → customer.zone → order.origin_zone
Drop Contact → customer.zone → order.destination_zone
origin_zone + destination_zone → zone_pricing.amount → vehicle.transport_amt
```

## Zone Pricing Store

Add a lookup method:

```go
func (s *ZonePricingStore) GetByZones(ctx context.Context, zoneA, zoneB string) (*models.ZonePricing, error)
```

Returns the zone pricing record matching the origin/destination pair, or `pgx.ErrNoRows` if no match. Company ID is extracted from context automatically (existing pattern — all store methods use `auth.GetCompanyID(ctx)`).

## Order Form UI

### Zone auto-fill from contacts

- When Load Contact is selected → auto-fill Origin Zone from `customer.zone` (editable)
- When Drop Contact is selected → auto-fill Destination Zone from `customer.zone` (editable)
- Both zone fields remain editable — user can override the auto-filled values
- Implemented via HTMX: contact selection triggers server round-trip that returns updated zone fields

### Rate hint display

- When both origin and destination zones are set, the handler looks up zone pricing
- If a match is found, display the rate near the zone fields: "Zone rate: $150.00"
- Informational only — the rate is applied when vehicles are added, not at the order level

### Zone change on existing order with vehicles

- If user changes origin/destination zone and a new zone pricing match is found:
  - Prompt: "Zone rate changed to $X. Update N existing vehicles?"
  - **Yes**: Update `transport_amt` on vehicles that still match the old rate (don't overwrite manually customized amounts)
  - **No**: Leave existing vehicles as-is; only new vehicles get the new rate

## Order Show Page

- Display Origin Zone and Destination Zone in the order header/details section
- Show the current zone pricing rate if a match exists

## Vehicle Add Flow

- When adding a vehicle to an order, auto-populate `transport_amt` from zone pricing lookup (order.origin_zone → order.destination_zone)
- Field is pre-filled but fully editable
- If no zone pricing match exists, field is left blank for manual entry

## Customer Form

- Add a Zone dropdown/text field to the customer create/edit form
- Populated from existing zones table
- Optional field
- Display zone on customer detail and list pages

## API / Proto Changes

### order.proto

- Rename `zone` field to `origin_zone` on the Order message
- Add `destination_zone` string field
- Update CreateOrderRequest and UpdateOrderRequest
- Update ListOrdersRequest: existing `zone` filter becomes `origin_zone` filter; add optional `destination_zone` filter

### customer.proto

- Add `zone` string field to Customer message (if not already present)

### Connect-RPC converters

- Update `internal/connectrpc/order_server.go` converters for renamed/new zone fields
- Update `internal/connectrpc/customer_server.go` if zone field is new

### MCP tools

- Update `cmd/atlinks-mcp/tools_order.go` field mappings and descriptions for `origin_zone` / `destination_zone`

### New endpoint

- `GET /api/zone-pricing/lookup?origin_zone={zone_a}&destination_zone={zone_b}`
- Returns a rate hint HTML partial, or 204 No Content if no match
- Used by HTMX on the order form for the rate hint
- Optionally exposed as a Connect-RPC method for MCP

## Migration

Single migration file:

1. `ALTER TABLE customers ADD COLUMN zone VARCHAR(20)`
2. `ALTER TABLE orders RENAME COLUMN zone TO origin_zone`
3. `ALTER TABLE orders ADD COLUMN destination_zone VARCHAR(20)`
4. Update any indexes that reference `orders.zone`

## Implementation Approach

Server-side (Approach A). All pricing logic lives in Go handlers. HTMX handles the UI updates. No new client-side JS beyond what HTMX provides. Follows existing app patterns.

### Multi-tenancy

All lookups are company-scoped via middleware context (`auth.GetCompanyID`). This is the existing pattern for every store method — no special handling needed.

### Validation

- `origin_zone` and `destination_zone` are optional on orders (not all orders have zone-based pricing)
- Zone values are free-text, not validated against the zones table (dispatchers may enter custom zones)
- Zone pricing lookup is best-effort: no match = no auto-fill, no error

### Filter behavior

- `OrderFilter.Zone` field is renamed to `OriginZone` and filters by `origin_zone`
- A new `DestinationZone` filter is added
- `orderSortConfig` is updated: `"zone"` key maps to `"origin_zone"` column

### Rate hint trigger

The zone pricing lookup is triggered via HTMX `hx-trigger="change"` on the zone input fields (debounced). Returns a partial with the rate hint text.

## Files to Modify

| File | Change |
|------|--------|
| Migration 037 | Schema changes (customers.zone, orders zone rename/add) |
| `internal/models/order.go` | Rename Zone → OriginZone, add DestinationZone |
| `internal/models/customer.go` | Add Zone field |
| `internal/store/order_store.go` | Update columns, queries, orderSortConfig (zone→origin_zone), StatusSummary() |
| `internal/store/customer_store.go` | Add zone to columns, queries |
| `internal/store/zone_store.go` | Add `GetByZones()` lookup method |
| `internal/handler/order_handler.go` | Zone auto-fill logic, pricing lookup, vehicle update prompt |
| `internal/handler/customer_handler.go` | Zone field in form binding |
| `internal/handler/zone_handler.go` | New lookup endpoint |
| `internal/handler/components/orders/form.templ` | Origin/destination zone fields, rate hint |
| `internal/handler/components/orders/show.templ` | Display both zones |
| `internal/handler/components/customers/form.templ` | Zone field |
| `internal/handler/components/customers/show.templ` | Display zone |
| `internal/handler/components/orders/vehicle_form.templ` | Auto-populated transport_amt |
| `proto/atlinks/v1/order.proto` | Rename zone, add destination_zone |
| `proto/atlinks/v1/customer.proto` | Add zone field |
| `internal/connectrpc/order_server.go` | Update converters |
| `internal/connectrpc/customer_server.go` | Update converters |
| `cmd/atlinks-mcp/tools_order.go` | Update field mappings |
