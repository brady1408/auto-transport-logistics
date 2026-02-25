# Soft Deletes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `deleted_at TIMESTAMPTZ` to ~27 tables so all user-initiated deletes become soft deletes hidden from queries rather than hard `DELETE` statements.

**Architecture:** One migration adds `deleted_at DEFAULT NULL` to every affected table. Each store gets two mechanical changes: (1) `DELETE FROM` → `UPDATE SET deleted_at = NOW()`, (2) all SELECT queries gain `AND deleted_at IS NULL`. No model struct changes, no handler changes, no frontend changes. The `hx-confirm` dialogs stay in place.

**Tech Stack:** PostgreSQL 16, goose migrations, pgx/v5, Go 1.22+. No ORM — raw SQL only.

---

## Key Patterns (Read Before Starting)

**Every DELETE becomes:**
```sql
-- Before
DELETE FROM customers WHERE id = $1 AND company_id = $2

-- After
UPDATE customers SET deleted_at = NOW() WHERE id = $1 AND company_id = $2
```

**Every queryBuilder-based List gets one extra filter:**
```go
// Add this line to every List() function that uses a queryBuilder, alongside the existing qb.Add() calls:
qb.Add("deleted_at IS NULL")
```

**Every direct SELECT gets AND deleted_at IS NULL:**
```sql
-- Before
SELECT ... FROM order_vehicles WHERE order_id = $1 AND company_id = $2

-- After
SELECT ... FROM order_vehicles WHERE order_id = $1 AND company_id = $2 AND deleted_at IS NULL
```

**Verify after every task:**
```bash
go build ./...
```
Expected: no compile errors (SQL is a string — compile only catches Go syntax errors).

---

### Task 1: Migration — add deleted_at to all tables

**Files:**
- Create: `internal/database/migrations/017_soft_deletes.sql`

**Step 1: Create the migration file**

```sql
-- +goose Up
-- Core entities
ALTER TABLE customers ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE employees ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE trucks ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE zones ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE zone_pricing ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- Dispatch
ALTER TABLE orders ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE order_vehicles ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE order_charges ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE vehicle_damage ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE vehicle_notes ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE trips ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE load_details ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE trip_fuel ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE trip_expenses ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE trip_routes ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- Accounting
ALTER TABLE invoices ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE invoice_details ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE payments ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE payment_details ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE credit_memos ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE damage_claims ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE accounts_payable ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- Lookups
ALTER TABLE items ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE tax_codes ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE terms ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE dispatch_codes ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE equipment_types ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE regions ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE damage_areas ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE damage_types ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE damage_severities ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE hold_codes ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE declination_codes ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- Other
ALTER TABLE attachments ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE loadboard_listings ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE loadboard_claims ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;

-- +goose Down
ALTER TABLE customers DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE employees DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE trucks DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE zones DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE zone_pricing DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE orders DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE order_vehicles DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE order_charges DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE vehicle_damage DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE vehicle_notes DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE trips DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE load_details DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE trip_fuel DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE trip_expenses DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE trip_routes DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE invoices DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE invoice_details DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE payments DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE payment_details DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE credit_memos DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE damage_claims DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE accounts_payable DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE items DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE tax_codes DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE terms DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE dispatch_codes DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE equipment_types DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE regions DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE damage_areas DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE damage_types DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE damage_severities DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE hold_codes DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE declination_codes DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE attachments DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE loadboard_listings DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE loadboard_claims DROP COLUMN IF EXISTS deleted_at;
```

**Step 2: Run migration against local DB**

```bash
make migrate-up
```
Expected: `OK   017_soft_deletes.sql`

**Step 3: Commit**

```bash
git add internal/database/migrations/017_soft_deletes.sql
git commit -m "Add deleted_at column to all soft-deletable tables"
```

---

### Task 2: Core entity stores

**Files:**
- Modify: `internal/store/customer_store.go`
- Modify: `internal/store/employee_store.go`
- Modify: `internal/store/truck_store.go`
- Modify: `internal/store/zone_store.go`

**Pattern for each file:**

1. Find the `List()` function. It will have a `queryBuilder` with `qb.Add("company_id = ?", companyID)`. Add this line alongside the other `qb.Add` calls:
```go
qb.Add("deleted_at IS NULL")
```

2. Find `GetByID()`. It will have a direct WHERE clause. Add `AND deleted_at IS NULL`:
```sql
-- customer example
SELECT {cols} FROM customers WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL
```

3. Find `Delete()`. Change:
```go
// Before
result, err := s.pool.Exec(ctx, "DELETE FROM customers WHERE id = $1 AND company_id = $2", id, companyID)

// After
result, err := s.pool.Exec(ctx, "UPDATE customers SET deleted_at = NOW() WHERE id = $1 AND company_id = $2", id, companyID)
```

Apply the same three changes to `employees`, `trucks`, `zones`.

For `zone_store.go` also update `zone_pricing` — find its List/Get/Delete and apply the same pattern. Zone pricing may use `zone_a` and `zone_b` as identifiers rather than a single `id` — check the actual query and adapt accordingly.

**Step 4: Build**
```bash
go build ./...
```
Expected: no errors.

**Step 5: Commit**
```bash
git add internal/store/customer_store.go internal/store/employee_store.go \
        internal/store/truck_store.go internal/store/zone_store.go
git commit -m "Soft delete: core entity stores (customer, employee, truck, zone)"
```

---

### Task 3: Order stores

**Files:**
- Modify: `internal/store/order_store.go`
- Modify: `internal/store/vehicle_store.go`
- Modify: `internal/store/charge_store.go`
- Modify: `internal/store/damage_store.go` (handles `vehicle_damage` AND `vehicle_notes`)

**order_store.go:**
- `List()`: add `qb.Add("deleted_at IS NULL")`
- `GetByID()`: add `AND deleted_at IS NULL`
- `Delete()`: `DELETE FROM orders` → `UPDATE orders SET deleted_at = NOW()`

**vehicle_store.go:**
- `ListByOrder()`: add `AND deleted_at IS NULL` to direct WHERE
- `GetByID()`: add `AND deleted_at IS NULL`
- `Delete()`: `DELETE FROM order_vehicles` → `UPDATE order_vehicles SET deleted_at = NOW()`
- `DeleteTx()`: same change but on the transaction variant

**charge_store.go:**
- Find List/Get queries and add `AND deleted_at IS NULL`
- `Delete()`: `DELETE FROM order_charges` → `UPDATE order_charges SET deleted_at = NOW()`

**damage_store.go** (handles both `vehicle_damage` and `vehicle_notes`):
- Find all List/Get queries for both tables — add `AND deleted_at IS NULL` to each
- `Delete()` for vehicle_damage: `DELETE FROM vehicle_damage` → `UPDATE vehicle_damage SET deleted_at = NOW()`
- `Delete()` for vehicle_notes: `DELETE FROM vehicle_notes` → `UPDATE vehicle_notes SET deleted_at = NOW()`

**Step N: Build**
```bash
go build ./...
```

**Step N+1: Commit**
```bash
git add internal/store/order_store.go internal/store/vehicle_store.go \
        internal/store/charge_store.go internal/store/damage_store.go
git commit -m "Soft delete: order-related stores"
```

---

### Task 4: Trip stores

**Files:**
- Modify: `internal/store/trip_store.go`
- Modify: `internal/store/load_detail_store.go`
- Modify: `internal/store/trip_fuel_store.go`
- Modify: `internal/store/trip_expense_store.go`
- Modify: `internal/store/trip_route_store.go`

**trip_store.go:**
- `List()`: add `qb.Add("deleted_at IS NULL")`
- `GetByID()` and `GetByIDTx()`: add `AND deleted_at IS NULL`
- `Delete()` for trips: `DELETE FROM trips` → `UPDATE trips SET deleted_at = NOW()`
- NOTE: `DashboardCounts()`, `TripSummaryReport()`, `DriverSettlement()` also query trips — add `AND deleted_at IS NULL` to their WHERE clauses too

**load_detail_store.go:**
- Find `ListByTrip()` or equivalent — add `AND deleted_at IS NULL`
- `Delete()`: `DELETE FROM load_details` → `UPDATE load_details SET deleted_at = NOW()`
- `DeleteTx()` (if present): same change

**trip_fuel_store.go:**
- List queries: add `AND deleted_at IS NULL`
- `Delete()`: `DELETE FROM trip_fuel` → `UPDATE trip_fuel SET deleted_at = NOW()`

**trip_expense_store.go:**
- List queries: add `AND deleted_at IS NULL`
- `Delete()`: `DELETE FROM trip_expenses` → `UPDATE trip_expenses SET deleted_at = NOW()`

**trip_route_store.go:**
- List queries: add `AND deleted_at IS NULL`
- `Delete()`: `DELETE FROM trip_routes` → `UPDATE trip_routes SET deleted_at = NOW()`

**Build and commit:**
```bash
go build ./...
git add internal/store/trip_store.go internal/store/load_detail_store.go \
        internal/store/trip_fuel_store.go internal/store/trip_expense_store.go \
        internal/store/trip_route_store.go
git commit -m "Soft delete: trip-related stores"
```

---

### Task 5: Accounting stores

**Files:**
- Modify: `internal/store/invoice_store.go`
- Modify: `internal/store/invoice_detail_store.go`
- Modify: `internal/store/payment_store.go`
- Modify: `internal/store/payment_detail_store.go`
- Modify: `internal/store/credit_memo_store.go`
- Modify: `internal/store/damage_claim_store.go`
- Modify: `internal/store/accounts_payable_store.go`

Apply the same pattern to each:
1. `List()` → add `qb.Add("deleted_at IS NULL")`
2. `GetByID()` → add `AND deleted_at IS NULL`
3. `Delete()` → `DELETE FROM {table}` → `UPDATE {table} SET deleted_at = NOW()`

**invoice_detail_store.go — IMPORTANT EXCEPTION:**

This store has TWO delete methods:
1. `Delete(id, companyID)` — deletes a single line item → **soft delete this one**
2. `DeleteByInvoice(invoiceID, companyID)` — cascade-deletes all details when voiding an invoice → **leave this as hard DELETE** (the whole invoice is being voided, not a user-initiated record delete)

Make sure only the single-item `Delete()` is changed.

**payment_detail_store.go:**
- `Delete()` removes a payment application → **soft delete**
- Any cascade deletes if present → leave as hard DELETE

**Build and commit:**
```bash
go build ./...
git add internal/store/invoice_store.go internal/store/invoice_detail_store.go \
        internal/store/payment_store.go internal/store/payment_detail_store.go \
        internal/store/credit_memo_store.go internal/store/damage_claim_store.go \
        internal/store/accounts_payable_store.go
git commit -m "Soft delete: accounting stores"
```

---

### Task 6: Lookup, attachment, and loadboard stores

**Files:**
- Modify: `internal/store/lookup_store.go`
- Modify: `internal/store/attachment_store.go`
- Modify: `internal/store/loadboard_store.go`

**lookup_store.go:**

The `List()` method uses a dynamic table name via `fmt.Sprintf`. Find the SELECT query and add `AND deleted_at IS NULL`. It likely looks like:
```go
fmt.Sprintf("SELECT ... FROM %s WHERE company_id = $1 ORDER BY ...", s.tableName)
```
Change to:
```go
fmt.Sprintf("SELECT ... FROM %s WHERE company_id = $1 AND deleted_at IS NULL ORDER BY ...", s.tableName)
```

The `Delete()` method:
```go
// Before
fmt.Sprintf("DELETE FROM %s WHERE id = $1 AND company_id = $2", s.tableName)

// After
fmt.Sprintf("UPDATE %s SET deleted_at = NOW() WHERE id = $1 AND company_id = $2", s.tableName)
```

**attachment_store.go:**

The `ListByEntity()` uses a query builder — add `qb.Add("deleted_at IS NULL")`.

The `DeleteByEntity()` builds a dynamic DELETE that also returns `storage_key` values for disk cleanup:
```go
query := fmt.Sprintf("DELETE FROM attachments %s RETURNING storage_key", qb.Where())
```
Change to a soft delete — but note we can no longer RETURNING the storage key since we're doing an UPDATE, so the physical files will be orphaned (acceptable for now). Change to:
```go
query := fmt.Sprintf("UPDATE attachments SET deleted_at = NOW() %s", strings.Replace(qb.Where(), "WHERE", "WHERE", 1))
```
Wait — actually the query builder's `Where()` returns `WHERE col = $1 AND col2 = $2`. An UPDATE needs `SET deleted_at = NOW() WHERE ...`. The cleanest approach:
```go
// Build the where clause normally, then use it in an UPDATE
query := fmt.Sprintf("UPDATE attachments SET deleted_at = NOW() WHERE %s", qb.WhereClause())
```
Check if `queryBuilder` has a method that returns just the conditions without the `WHERE` keyword, or adjust the string accordingly. If `qb.Where()` returns `WHERE entity_type = $1 AND entity_id = $2`, you can do:
```go
whereClause := qb.Where() // "WHERE entity_type = $1 AND entity_id = $2"
query := "UPDATE attachments SET deleted_at = NOW() " + whereClause
```
The `DeleteByEntity()` currently returns `[]string` (storage keys). With soft delete it no longer needs to return keys — change the return to nil/empty or keep the signature and return `[]string{}`. Check if callers use the returned keys; if they do, those physical-file-delete callers will simply skip deletion (files are orphaned, which is acceptable for now).

**loadboard_store.go:**

Find all `DELETE FROM loadboard_listings` and `DELETE FROM loadboard_claims` statements. Change each to soft delete:
```sql
UPDATE loadboard_listings SET deleted_at = NOW() WHERE id = $1 AND company_id = $2
UPDATE loadboard_claims SET deleted_at = NOW() WHERE id = $1 AND company_id = $2
```

Find all SELECT queries that list listings or claims and add `AND deleted_at IS NULL`.

Watch for cross-company queries in `ListAvailable()` — they also need `AND deleted_at IS NULL` since you don't want to show deleted listings to carriers.

**Build and commit:**
```bash
go build ./...
git add internal/store/lookup_store.go internal/store/attachment_store.go \
        internal/store/loadboard_store.go
git commit -m "Soft delete: lookup, attachment, and loadboard stores"
```

---

### Task 7: Integration verification

**Step 1: Apply migration to local DB**
```bash
make migrate-up
```
Expected: `OK   017_soft_deletes.sql`

**Step 2: Start server**
```bash
make run
```
Server at http://localhost:8080.

**Step 3: Smoke test**

Log in as admin and verify soft deletes work end-to-end:

1. **Create a customer**: Global → Customers → New Customer → fill in a test name → Save
2. **Verify it appears** in the customers list
3. **Delete it**: click Delete button → confirm the native dialog
4. **Verify it disappears** from the customers list
5. **Check the database directly** — confirm `deleted_at` is NOT NULL:
```bash
docker exec atlinks-pg psql -U atlinks -d atlinks -c \
  "SELECT id, name, deleted_at FROM customers WHERE deleted_at IS NOT NULL ORDER BY deleted_at DESC LIMIT 5;"
```
Expected: the deleted test customer appears with a timestamp.

6. Repeat for one trip and one invoice detail to cover the trip and accounting paths.

**Step 4: Commit**

No code changes in this task — it's verification only. If you found and fixed bugs, commit them with:
```bash
git add <affected files>
git commit -m "Fix soft delete query missed in <store name>"
```

---

### Task 8: Reply to feedback and close

After all tasks pass verification, post a reply to feedback item #26 via the atlinks-feedback MCP and close it.
