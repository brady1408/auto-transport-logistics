# Soft Deletes — ATLinks

**Date:** 2026-02-25
**Status:** Approved
**Scope:** UX-2 (MEDIUM) — Replace hard deletes with soft deletes. Undo toast is a follow-on.

## Context

All delete operations in ATLinks are hard `DELETE FROM` SQL statements. This is risky for a logistics app where accidental deletes could lose critical order/payment history. Moving to soft deletes adds a `deleted_at` column — records are never physically removed, just hidden from all queries.

The `hx-confirm` dialogs stay in place for now (no frontend changes). The undo toast UI will replace them in a follow-on pass.

No partial indexes added yet — will add when query performance becomes a real concern.

## Approach

**Schema:** One migration (`017_soft_deletes`) adds `deleted_at TIMESTAMPTZ DEFAULT NULL` to all affected tables.

**Store layer:** Two mechanical changes per table:
1. All `SELECT` queries gain `AND deleted_at IS NULL` (or `WHERE deleted_at IS NULL`)
2. All `DELETE FROM ... WHERE id = $1` become `UPDATE ... SET deleted_at = NOW() WHERE id = $1`

No signature changes — callers see no difference.

## Tables Receiving `deleted_at`

### Core entities
- `customers`
- `employees`
- `trucks`
- `zones`
- `zone_pricing`

### Dispatch
- `orders`
- `order_vehicles`
- `order_charges`
- `vehicle_damage`
- `vehicle_notes`
- `trips`
- `load_details`
- `trip_fuel`
- `trip_expenses`
- `trip_routes`

### Accounting
- `invoices`
- `invoice_details` (individual line item deletes only — cascade void deletes remain hard)
- `payments`
- `payment_details`
- `credit_memos`
- `damage_claims`
- `accounts_payable`

### Lookups
- `items`
- `tax_codes`
- `terms`
- Generic lookup table (via `LookupStore` — uses dynamic table name)

### Other
- `attachments`
- `loadboard_listings`
- `loadboard_claims`

## Exclusions

- `password_reset_tokens` — system cleanup, not user data
- `pending_registrations` — system cleanup
- `feedback` — internal tool, hard delete is fine
- `invoice_details WHERE invoice_id = $1` — cascade when voiding an invoice, keep hard delete
- `backups` — actual file deletion

## Store Files to Change

| Store file | Tables |
|---|---|
| `customer_store.go` | `customers` |
| `employee_store.go` | `employees` |
| `truck_store.go` | `trucks` |
| `zone_store.go` | `zones`, `zone_pricing` |
| `order_store.go` | `orders` |
| `vehicle_store.go` | `order_vehicles` |
| `charge_store.go` | `order_charges` |
| `damage_store.go` | `vehicle_damage` |
| `trip_store.go` | `trips`, `load_details` |
| `load_detail_store.go` | `load_details` |
| `trip_fuel_store.go` | `trip_fuel` |
| `trip_expense_store.go` | `trip_expenses` |
| `trip_route_store.go` | `trip_routes` |
| `invoice_store.go` | `invoices` |
| `invoice_detail_store.go` | `invoice_details` (individual only) |
| `payment_store.go` | `payments` |
| `payment_detail_store.go` | `payment_details` |
| `credit_memo_store.go` | `credit_memos` |
| `damage_claim_store.go` | `damage_claims` |
| `accounts_payable_store.go` | `accounts_payable` |
| `lookup_store.go` | dynamic table (items, tax_codes, terms, lookup_values) |
| `attachment_store.go` | `attachments` |
| `loadboard_store.go` | `loadboard_listings`, `loadboard_claims` |

Also: `vehicle_notes` — check which store handles `vehicle_notes` deletes.
