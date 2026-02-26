# Loadboard Safety & Process Visibility Design

**Date:** 2026-02-26
**Status:** Approved
**Scope:** Approach A — process visibility, claim milestones, vehicle status propagation, insurance warnings, no-show handling

## Background

The ATLinks loadboard has a solid foundation: posting from orders, claim/accept/reject flows, per-claim messaging, map view, and auto-import of orders into the carrier's system on acceptance. The primary gaps identified are:

1. After a claim is accepted, the poster's order and the carrier's order diverge with no visibility between them.
2. There is no mechanism to handle a carrier that accepts and then ghosts (no-show).
3. Insurance information is on the honor system — no warnings surface when a carrier has no insurance on file or their coverage is expired.

The loadboard is deployed as a **closed network** (known parties only) initially, with a plan to move to semi-open. This design targets the closed-network phase but is built to extend cleanly.

## Goals

- Give the poster real-time visibility into the subhauled load's progress without requiring manual coordination.
- Give the carrier clear milestones to communicate pickup and delivery.
- Surface insurance gaps at claim time so posters can make informed decisions.
- Give the poster a recovery path when a carrier no-shows.

## Non-Goals

- MC/DOT number verification against FMCSA (post-MVP, needed for semi-open)
- Blocking claims based on insurance status (warning only, not a gate)
- Poster confirmation of delivery (carrier's delivered mark is final)
- Ratings/reviews (post-MVP)
- Email/SMS notifications (post-MVP)

---

## Design

### 1. Claim Status Milestones

**Current flow:** `Pending → Accepted → Completed`

**New flow:**
```
Pending → Accepted → PickedUp → Delivered  (terminal, success)
                  ↘ NoShow                 (terminal, failure — poster-initiated)
     ↘ Rejected                            (unchanged)
     ↘ Cancelled                           (unchanged, carrier-initiated)
```

The `Completed` status is replaced by `Delivered`. Delivery is completion — no separate poster confirmation step.

**Schema change** — add two nullable timestamps to `loadboard_claims`:
- `picked_up_at TIMESTAMPTZ` — set when carrier marks pickup
- `delivered_at TIMESTAMPTZ` — set when carrier marks delivery

**New carrier actions** on `my-claims/{id}`:
- **Mark Picked Up** — available when claim status is `Accepted`; transitions to `PickedUp`
- **Mark Delivered** — available when claim status is `PickedUp`; transitions to `Delivered`

**Existing action removed:**
- The manual **Complete** button is removed. Delivery replaces it.

**New poster action** on `my-listings/{id}`:
- **Report No-Show** — available when claim status is `Accepted`; transitions to `NoShow`

No time-gating on the no-show button for now.

---

### 2. Subhauled Load Visibility on Poster's Order

The poster's order show page gets a **"Subhauled Loads"** section. It queries `loadboard_claims` joined with `loadboard_listings` where `listings.source_order_id` matches the order. The section is hidden when no accepted/active claims exist.

Each row is read-only and shows:
- Listing number + link to claim detail
- Carrier company name, MC#, DOT#
- Agreed pay
- Current status (with badge)
- Timestamps: picked up at, delivered at (when set)

No edit actions are available from the poster's side. The carrier company info is the snapshot captured at claim time.

If an order has multiple accepted claims (partial loads split across carriers), each appears as a separate row.

---

### 3. Vehicle Status Propagation

Vehicle status updates are triggered by the carrier's milestone actions. All updates run in the same transaction as the claim status change.

| Carrier action | Carrier's vehicles | Poster's source vehicles |
|---|---|---|
| Mark Picked Up | `Waiting → Loaded` | `Waiting → Scheduled` |
| Mark Delivered | `Loaded → Delivered` | `Scheduled → Delivered` |
| No-Show (poster) | unchanged | `Scheduled → Waiting` |

Poster's source vehicles are identified via `loadboard_listing_vehicles.source_vehicle_id`, which links back to the original `order_vehicles` record.

The `LoadboardService` already has access to both `vehicleStore` and `orderStore`. Vehicle updates use a system context scoped to the appropriate company (poster's company for poster's vehicles, carrier's company for carrier's vehicles).

---

### 4. Insurance Expiration Warning

Insurance data is on the honor system — companies may not have filled in their profile. The warning surfaces in two places.

**Carrier side — before claiming:**

On the listing detail page (`/loadboard/{id}`), if the viewing company's `insurance_exp_date` meets any of these conditions, an alert banner appears above the Claim button:

| Condition | Message |
|---|---|
| No date on file | ⚠ No insurance information on file for your company. Add it to your company profile before claiming loads. |
| Expired | ⚠ Your company's insurance on file expired [date]. Update your company profile before claiming loads. |
| Expires within 30 days | ⚠ Your company's insurance on file expires [date]. Consider updating your company profile. |

The Claim button is **not blocked** — this is informational only.

**Poster side — reviewing claimants:**

On the `my-listings/{id}` page, the claims table includes an insurance status column using a badge:

| State | Badge |
|---|---|
| No date on file | ⚠ None on file |
| Expired | ✗ Expired |
| Expiring ≤ 30 days | ⚠ Exp [date] |
| Valid | ✓ Valid |

Badge colors follow existing CSS conventions: warning (yellow), danger (red), active (green).

---

### 5. No-Show Handling

The **Report No-Show** button appears on the poster's `my-listings/{id}` page for any claim in `Accepted` status.

**What it does (in a single transaction):**
1. Marks claim → `NoShow` (sets `cancelled_at`, stores reason `"No-show reported by poster"` in `poster_notes`)
2. Returns listing → `Posted`
3. Reverts poster's source vehicles → `Waiting`
4. Carrier's imported order is left untouched (they own it in their system)

**Why no time-gate for now:** The closed-network assumption means posters know their carriers. Abuse of the no-show button is unlikely and can be tracked via the audit log. Time-gating (e.g., only available after pickup window has passed) can be added when moving to semi-open.

---

## Data Model Changes

```sql
-- Migration: add milestone timestamps to loadboard_claims
ALTER TABLE loadboard_claims
  ADD COLUMN picked_up_at  TIMESTAMPTZ,
  ADD COLUMN delivered_at  TIMESTAMPTZ;
```

No new tables required. The `NoShow` and `Delivered` values are new entries in the existing status string field (checked via Go constants, not a DB enum).

---

## File Surface

| File | Change |
|---|---|
| `migrations/020_loadboard_milestones.sql` | Add `picked_up_at`, `delivered_at` to `loadboard_claims` |
| `internal/models/loadboard.go` | Add `PickedUpAt`, `DeliveredAt` fields to `LoadboardClaim` |
| `internal/store/loadboard_store.go` | Add `MarkPickedUp`, `MarkDelivered`, `MarkNoShow` methods |
| `internal/service/loadboard_service.go` | Add `MarkPickedUp`, `MarkDelivered`, `ReportNoShow` methods with vehicle propagation |
| `internal/handler/loadboard_handler.go` | Add `pickupClaim`, `deliverClaim`, `noShowClaim` HTTP handlers; remove `completeClaim` |
| `internal/handler/components/loadboard/my_claim_show.templ` | Replace Complete button with Picked Up / Delivered buttons (conditional on status) |
| `internal/handler/components/loadboard/my_listing_show.templ` | Add No-Show button per accepted claim; add insurance badge column |
| `internal/handler/components/loadboard/show.templ` | Add insurance warning banner for carrier |
| `internal/handler/components/orders/show.templ` | Add Subhauled Loads section |
| `internal/store/loadboard_store.go` | Add `ListActiveClaimsForOrder` query |
