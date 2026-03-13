# BIGSERIAL Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Convert all SERIAL (int32) primary keys and their FK columns to BIGSERIAL/BIGINT across all 59 tables, eliminating the ~2.1 billion row ceiling in a multi-tenant environment.

**Architecture:** Single goose migration file (`018_bigserial.sql`) with three sections: extend sequences to bigint range, convert id columns, convert FK columns. No Go code changes needed — on amd64 Linux `int` == `int64`, and pgx v5 scans `BIGINT` into `*int` natively. The migration runs in a single transaction via goose's default behavior.

**Tech Stack:** Go 1.22, PostgreSQL 16, goose, pgx/v5

---

## Context

PostgreSQL `SERIAL` = `INTEGER` sequence (max ~2.1B rows per table). `BIGSERIAL` = `BIGINT` sequence (max ~9.2 quintillion). The conversion is safe:
- Every existing INTEGER value is a valid BIGINT value (no data loss)
- PostgreSQL allows ALTER COLUMN INTEGER → BIGINT within a transaction even across FK constraints
- Go `int` is 64-bit on amd64 — pgx v5 handles BIGINT ↔ int scanning without any model or store changes

Advisory lock registry (existing, for reference):
- 1 = order numbers, 2 = load numbers, 3 = invoice numbers, 4 = credit memo numbers, 5 = claim numbers

---

## Task 1: Create and apply the BIGSERIAL migration

**Files:**
- Create: `internal/database/migrations/018_bigserial.sql`

### Step 1: Create the migration file

Create `internal/database/migrations/018_bigserial.sql` with the full content below.

```sql
-- +goose Up

-- =============================================================================
-- Convert all SERIAL primary keys to BIGSERIAL and FK INTEGER columns to BIGINT.
-- Go code requires no changes: on amd64 Linux, int == int64; pgx v5 handles
-- BIGINT <-> int scanning natively.
-- =============================================================================

-- ---- 1. Extend all sequences to bigint range --------------------------------

ALTER SEQUENCE users_id_seq AS bigint;
ALTER SEQUENCE companies_id_seq AS bigint;
ALTER SEQUENCE customers_id_seq AS bigint;
ALTER SEQUENCE employees_id_seq AS bigint;
ALTER SEQUENCE trucks_id_seq AS bigint;
ALTER SEQUENCE trailers_id_seq AS bigint;
ALTER SEQUENCE zones_id_seq AS bigint;
ALTER SEQUENCE zone_pricing_id_seq AS bigint;
ALTER SEQUENCE vendors_id_seq AS bigint;
ALTER SEQUENCE vendor_groups_id_seq AS bigint;
ALTER SEQUENCE carriers_id_seq AS bigint;
ALTER SEQUENCE regions_id_seq AS bigint;
ALTER SEQUENCE dispatch_codes_id_seq AS bigint;
ALTER SEQUENCE equipment_types_id_seq AS bigint;
ALTER SEQUENCE items_id_seq AS bigint;
ALTER SEQUENCE vehicle_makes_id_seq AS bigint;
ALTER SEQUENCE vin_definitions_id_seq AS bigint;
ALTER SEQUENCE color_codes_id_seq AS bigint;
ALTER SEQUENCE hold_codes_id_seq AS bigint;
ALTER SEQUENCE declination_codes_id_seq AS bigint;
ALTER SEQUENCE field_codes_1_id_seq AS bigint;
ALTER SEQUENCE field_codes_2_id_seq AS bigint;
ALTER SEQUENCE field_codes_3_id_seq AS bigint;
ALTER SEQUENCE field_codes_4_id_seq AS bigint;
ALTER SEQUENCE field_codes_5_id_seq AS bigint;
ALTER SEQUENCE damage_areas_id_seq AS bigint;
ALTER SEQUENCE damage_types_id_seq AS bigint;
ALTER SEQUENCE damage_severities_id_seq AS bigint;
ALTER SEQUENCE terms_id_seq AS bigint;
ALTER SEQUENCE tax_codes_id_seq AS bigint;
ALTER SEQUENCE chart_of_accounts_id_seq AS bigint;
ALTER SEQUENCE orders_id_seq AS bigint;
ALTER SEQUENCE trips_id_seq AS bigint;
ALTER SEQUENCE order_vehicles_id_seq AS bigint;
ALTER SEQUENCE load_details_id_seq AS bigint;
ALTER SEQUENCE order_charges_id_seq AS bigint;
ALTER SEQUENCE vehicle_damage_id_seq AS bigint;
ALTER SEQUENCE damage_details_id_seq AS bigint;
ALTER SEQUENCE vehicle_notes_id_seq AS bigint;
ALTER SEQUENCE trip_fuel_id_seq AS bigint;
ALTER SEQUENCE trip_expenses_id_seq AS bigint;
ALTER SEQUENCE trip_routes_id_seq AS bigint;
ALTER SEQUENCE split_loads_id_seq AS bigint;
ALTER SEQUENCE invoices_id_seq AS bigint;
ALTER SEQUENCE invoice_details_id_seq AS bigint;
ALTER SEQUENCE credit_memos_id_seq AS bigint;
ALTER SEQUENCE payments_id_seq AS bigint;
ALTER SEQUENCE payment_details_id_seq AS bigint;
ALTER SEQUENCE damage_claims_id_seq AS bigint;
ALTER SEQUENCE accounts_payable_id_seq AS bigint;
ALTER SEQUENCE audit_log_id_seq AS bigint;
ALTER SEQUENCE feedback_id_seq AS bigint;
ALTER SEQUENCE password_reset_tokens_id_seq AS bigint;
ALTER SEQUENCE pending_registrations_id_seq AS bigint;
ALTER SEQUENCE feedback_comments_id_seq AS bigint;
ALTER SEQUENCE attachments_id_seq AS bigint;
ALTER SEQUENCE loadboard_listings_id_seq AS bigint;
ALTER SEQUENCE loadboard_listing_vehicles_id_seq AS bigint;
ALTER SEQUENCE loadboard_claims_id_seq AS bigint;
ALTER SEQUENCE loadboard_messages_id_seq AS bigint;

-- ---- 2. Convert id columns to bigint ----------------------------------------
-- (referenced tables first so FK constraints remain valid throughout)

ALTER TABLE companies ALTER COLUMN id TYPE bigint;
ALTER TABLE users ALTER COLUMN id TYPE bigint;
ALTER TABLE customers ALTER COLUMN id TYPE bigint;
ALTER TABLE employees ALTER COLUMN id TYPE bigint;
ALTER TABLE trucks ALTER COLUMN id TYPE bigint;
ALTER TABLE trailers ALTER COLUMN id TYPE bigint;
ALTER TABLE zones ALTER COLUMN id TYPE bigint;
ALTER TABLE zone_pricing ALTER COLUMN id TYPE bigint;
ALTER TABLE vendors ALTER COLUMN id TYPE bigint;
ALTER TABLE vendor_groups ALTER COLUMN id TYPE bigint;
ALTER TABLE carriers ALTER COLUMN id TYPE bigint;
ALTER TABLE regions ALTER COLUMN id TYPE bigint;
ALTER TABLE dispatch_codes ALTER COLUMN id TYPE bigint;
ALTER TABLE equipment_types ALTER COLUMN id TYPE bigint;
ALTER TABLE items ALTER COLUMN id TYPE bigint;
ALTER TABLE vehicle_makes ALTER COLUMN id TYPE bigint;
ALTER TABLE vin_definitions ALTER COLUMN id TYPE bigint;
ALTER TABLE color_codes ALTER COLUMN id TYPE bigint;
ALTER TABLE hold_codes ALTER COLUMN id TYPE bigint;
ALTER TABLE declination_codes ALTER COLUMN id TYPE bigint;
ALTER TABLE field_codes_1 ALTER COLUMN id TYPE bigint;
ALTER TABLE field_codes_2 ALTER COLUMN id TYPE bigint;
ALTER TABLE field_codes_3 ALTER COLUMN id TYPE bigint;
ALTER TABLE field_codes_4 ALTER COLUMN id TYPE bigint;
ALTER TABLE field_codes_5 ALTER COLUMN id TYPE bigint;
ALTER TABLE damage_areas ALTER COLUMN id TYPE bigint;
ALTER TABLE damage_types ALTER COLUMN id TYPE bigint;
ALTER TABLE damage_severities ALTER COLUMN id TYPE bigint;
ALTER TABLE terms ALTER COLUMN id TYPE bigint;
ALTER TABLE tax_codes ALTER COLUMN id TYPE bigint;
ALTER TABLE chart_of_accounts ALTER COLUMN id TYPE bigint;
ALTER TABLE orders ALTER COLUMN id TYPE bigint;
ALTER TABLE trips ALTER COLUMN id TYPE bigint;
ALTER TABLE order_vehicles ALTER COLUMN id TYPE bigint;
ALTER TABLE load_details ALTER COLUMN id TYPE bigint;
ALTER TABLE order_charges ALTER COLUMN id TYPE bigint;
ALTER TABLE vehicle_damage ALTER COLUMN id TYPE bigint;
ALTER TABLE damage_details ALTER COLUMN id TYPE bigint;
ALTER TABLE vehicle_notes ALTER COLUMN id TYPE bigint;
ALTER TABLE trip_fuel ALTER COLUMN id TYPE bigint;
ALTER TABLE trip_expenses ALTER COLUMN id TYPE bigint;
ALTER TABLE trip_routes ALTER COLUMN id TYPE bigint;
ALTER TABLE split_loads ALTER COLUMN id TYPE bigint;
ALTER TABLE invoices ALTER COLUMN id TYPE bigint;
ALTER TABLE invoice_details ALTER COLUMN id TYPE bigint;
ALTER TABLE credit_memos ALTER COLUMN id TYPE bigint;
ALTER TABLE payments ALTER COLUMN id TYPE bigint;
ALTER TABLE payment_details ALTER COLUMN id TYPE bigint;
ALTER TABLE damage_claims ALTER COLUMN id TYPE bigint;
ALTER TABLE accounts_payable ALTER COLUMN id TYPE bigint;
ALTER TABLE audit_log ALTER COLUMN id TYPE bigint;
ALTER TABLE feedback ALTER COLUMN id TYPE bigint;
ALTER TABLE password_reset_tokens ALTER COLUMN id TYPE bigint;
ALTER TABLE pending_registrations ALTER COLUMN id TYPE bigint;
ALTER TABLE feedback_comments ALTER COLUMN id TYPE bigint;
ALTER TABLE attachments ALTER COLUMN id TYPE bigint;
ALTER TABLE loadboard_listings ALTER COLUMN id TYPE bigint;
ALTER TABLE loadboard_listing_vehicles ALTER COLUMN id TYPE bigint;
ALTER TABLE loadboard_claims ALTER COLUMN id TYPE bigint;
ALTER TABLE loadboard_messages ALTER COLUMN id TYPE bigint;

-- ---- 3. Convert FK and reference integer columns to bigint ------------------

-- users
ALTER TABLE users ALTER COLUMN company_id TYPE bigint;

-- customers
ALTER TABLE customers ALTER COLUMN company_id TYPE bigint;

-- employees
ALTER TABLE employees ALTER COLUMN company_id TYPE bigint;

-- trucks
ALTER TABLE trucks ALTER COLUMN company_id TYPE bigint;

-- trailers
ALTER TABLE trailers ALTER COLUMN company_id TYPE bigint;

-- zones
ALTER TABLE zones ALTER COLUMN company_id TYPE bigint;

-- zone_pricing
ALTER TABLE zone_pricing ALTER COLUMN company_id TYPE bigint;

-- vendors
ALTER TABLE vendors ALTER COLUMN company_id TYPE bigint;

-- vendor_groups
ALTER TABLE vendor_groups ALTER COLUMN company_id TYPE bigint;

-- carriers
ALTER TABLE carriers ALTER COLUMN company_id TYPE bigint;

-- regions
ALTER TABLE regions ALTER COLUMN company_id TYPE bigint;

-- dispatch_codes
ALTER TABLE dispatch_codes ALTER COLUMN company_id TYPE bigint;

-- equipment_types
ALTER TABLE equipment_types ALTER COLUMN company_id TYPE bigint;

-- items
ALTER TABLE items ALTER COLUMN company_id TYPE bigint;

-- hold_codes
ALTER TABLE hold_codes ALTER COLUMN company_id TYPE bigint;

-- declination_codes
ALTER TABLE declination_codes ALTER COLUMN company_id TYPE bigint;

-- field_codes_1-5
ALTER TABLE field_codes_1 ALTER COLUMN company_id TYPE bigint;
ALTER TABLE field_codes_2 ALTER COLUMN company_id TYPE bigint;
ALTER TABLE field_codes_3 ALTER COLUMN company_id TYPE bigint;
ALTER TABLE field_codes_4 ALTER COLUMN company_id TYPE bigint;
ALTER TABLE field_codes_5 ALTER COLUMN company_id TYPE bigint;

-- damage_areas / damage_types / damage_severities
ALTER TABLE damage_areas ALTER COLUMN company_id TYPE bigint;
ALTER TABLE damage_types ALTER COLUMN company_id TYPE bigint;
ALTER TABLE damage_severities ALTER COLUMN company_id TYPE bigint;

-- terms
ALTER TABLE terms ALTER COLUMN company_id TYPE bigint;

-- tax_codes
ALTER TABLE tax_codes ALTER COLUMN company_id TYPE bigint;

-- chart_of_accounts
ALTER TABLE chart_of_accounts ALTER COLUMN company_id TYPE bigint;

-- orders
ALTER TABLE orders ALTER COLUMN company_id TYPE bigint;
ALTER TABLE orders ALTER COLUMN bill_customer_id TYPE bigint;
ALTER TABLE orders ALTER COLUMN load_customer_id TYPE bigint;
ALTER TABLE orders ALTER COLUMN drop_customer_id TYPE bigint;

-- trips
ALTER TABLE trips ALTER COLUMN company_id TYPE bigint;
ALTER TABLE trips ALTER COLUMN truck_id TYPE bigint;
ALTER TABLE trips ALTER COLUMN driver1_id TYPE bigint;
ALTER TABLE trips ALTER COLUMN driver2_id TYPE bigint;

-- order_vehicles
ALTER TABLE order_vehicles ALTER COLUMN company_id TYPE bigint;
ALTER TABLE order_vehicles ALTER COLUMN order_id TYPE bigint;
ALTER TABLE order_vehicles ALTER COLUMN trip_id TYPE bigint;

-- load_details
ALTER TABLE load_details ALTER COLUMN company_id TYPE bigint;
ALTER TABLE load_details ALTER COLUMN trip_id TYPE bigint;
ALTER TABLE load_details ALTER COLUMN order_id TYPE bigint;
ALTER TABLE load_details ALTER COLUMN vehicle_id TYPE bigint;

-- order_charges
ALTER TABLE order_charges ALTER COLUMN company_id TYPE bigint;
ALTER TABLE order_charges ALTER COLUMN order_id TYPE bigint;
ALTER TABLE order_charges ALTER COLUMN vehicle_id TYPE bigint;
ALTER TABLE order_charges ALTER COLUMN trip_id TYPE bigint;

-- vehicle_damage
ALTER TABLE vehicle_damage ALTER COLUMN company_id TYPE bigint;
ALTER TABLE vehicle_damage ALTER COLUMN order_id TYPE bigint;
ALTER TABLE vehicle_damage ALTER COLUMN vehicle_id TYPE bigint;
ALTER TABLE vehicle_damage ALTER COLUMN trip_id TYPE bigint;

-- damage_details
ALTER TABLE damage_details ALTER COLUMN company_id TYPE bigint;
ALTER TABLE damage_details ALTER COLUMN vehicle_damage_id TYPE bigint;

-- vehicle_notes
ALTER TABLE vehicle_notes ALTER COLUMN company_id TYPE bigint;
ALTER TABLE vehicle_notes ALTER COLUMN vehicle_id TYPE bigint;

-- trip_fuel
ALTER TABLE trip_fuel ALTER COLUMN company_id TYPE bigint;
ALTER TABLE trip_fuel ALTER COLUMN trip_id TYPE bigint;

-- trip_expenses
ALTER TABLE trip_expenses ALTER COLUMN company_id TYPE bigint;
ALTER TABLE trip_expenses ALTER COLUMN trip_id TYPE bigint;

-- trip_routes
ALTER TABLE trip_routes ALTER COLUMN company_id TYPE bigint;
ALTER TABLE trip_routes ALTER COLUMN trip_id TYPE bigint;
ALTER TABLE trip_routes ALTER COLUMN customer_id TYPE bigint;

-- split_loads
ALTER TABLE split_loads ALTER COLUMN company_id TYPE bigint;
ALTER TABLE split_loads ALTER COLUMN order_id TYPE bigint;
ALTER TABLE split_loads ALTER COLUMN vehicle_id TYPE bigint;
ALTER TABLE split_loads ALTER COLUMN trip_id TYPE bigint;
ALTER TABLE split_loads ALTER COLUMN orig_trip_id TYPE bigint;

-- invoices
ALTER TABLE invoices ALTER COLUMN company_id TYPE bigint;
ALTER TABLE invoices ALTER COLUMN customer_id TYPE bigint;
ALTER TABLE invoices ALTER COLUMN order_id TYPE bigint;

-- invoice_details
ALTER TABLE invoice_details ALTER COLUMN company_id TYPE bigint;
ALTER TABLE invoice_details ALTER COLUMN invoice_id TYPE bigint;
ALTER TABLE invoice_details ALTER COLUMN order_id TYPE bigint;
ALTER TABLE invoice_details ALTER COLUMN vehicle_id TYPE bigint;

-- credit_memos
ALTER TABLE credit_memos ALTER COLUMN company_id TYPE bigint;
ALTER TABLE credit_memos ALTER COLUMN customer_id TYPE bigint;
ALTER TABLE credit_memos ALTER COLUMN invoice_id TYPE bigint;

-- payments
ALTER TABLE payments ALTER COLUMN company_id TYPE bigint;
ALTER TABLE payments ALTER COLUMN customer_id TYPE bigint;

-- payment_details
ALTER TABLE payment_details ALTER COLUMN company_id TYPE bigint;
ALTER TABLE payment_details ALTER COLUMN payment_id TYPE bigint;
ALTER TABLE payment_details ALTER COLUMN invoice_id TYPE bigint;

-- damage_claims
ALTER TABLE damage_claims ALTER COLUMN company_id TYPE bigint;
ALTER TABLE damage_claims ALTER COLUMN order_id TYPE bigint;
ALTER TABLE damage_claims ALTER COLUMN vehicle_id TYPE bigint;
ALTER TABLE damage_claims ALTER COLUMN trip_id TYPE bigint;

-- accounts_payable
ALTER TABLE accounts_payable ALTER COLUMN company_id TYPE bigint;
ALTER TABLE accounts_payable ALTER COLUMN trip_id TYPE bigint;
ALTER TABLE accounts_payable ALTER COLUMN employee_id TYPE bigint;
ALTER TABLE accounts_payable ALTER COLUMN truck_id TYPE bigint;

-- audit_log (user_id FK; record_id is polymorphic — convert both)
ALTER TABLE audit_log ALTER COLUMN user_id TYPE bigint;
ALTER TABLE audit_log ALTER COLUMN record_id TYPE bigint;

-- feedback
ALTER TABLE feedback ALTER COLUMN company_id TYPE bigint;
ALTER TABLE feedback ALTER COLUMN user_id TYPE bigint;

-- password_reset_tokens
ALTER TABLE password_reset_tokens ALTER COLUMN user_id TYPE bigint;

-- feedback_comments
ALTER TABLE feedback_comments ALTER COLUMN feedback_id TYPE bigint;
ALTER TABLE feedback_comments ALTER COLUMN user_id TYPE bigint;
ALTER TABLE feedback_comments ALTER COLUMN company_id TYPE bigint;

-- attachments (entity_id is polymorphic)
ALTER TABLE attachments ALTER COLUMN company_id TYPE bigint;
ALTER TABLE attachments ALTER COLUMN entity_id TYPE bigint;
ALTER TABLE attachments ALTER COLUMN uploaded_by TYPE bigint;

-- loadboard_listings
ALTER TABLE loadboard_listings ALTER COLUMN poster_company_id TYPE bigint;
ALTER TABLE loadboard_listings ALTER COLUMN poster_user_id TYPE bigint;
ALTER TABLE loadboard_listings ALTER COLUMN source_order_id TYPE bigint;

-- loadboard_listing_vehicles
ALTER TABLE loadboard_listing_vehicles ALTER COLUMN listing_id TYPE bigint;
ALTER TABLE loadboard_listing_vehicles ALTER COLUMN source_vehicle_id TYPE bigint;

-- loadboard_claims
ALTER TABLE loadboard_claims ALTER COLUMN listing_id TYPE bigint;
ALTER TABLE loadboard_claims ALTER COLUMN carrier_company_id TYPE bigint;
ALTER TABLE loadboard_claims ALTER COLUMN carrier_user_id TYPE bigint;
ALTER TABLE loadboard_claims ALTER COLUMN carrier_order_id TYPE bigint;

-- loadboard_messages
ALTER TABLE loadboard_messages ALTER COLUMN claim_id TYPE bigint;
ALTER TABLE loadboard_messages ALTER COLUMN sender_company_id TYPE bigint;
ALTER TABLE loadboard_messages ALTER COLUMN sender_user_id TYPE bigint;


-- +goose Down

-- WARNING: Reverting BIGINT → INTEGER is only safe if no ID values exceed 2,147,483,647.
-- On a production system with real data, do NOT run this down migration.

-- ---- Restore FK columns to integer ----
ALTER TABLE users ALTER COLUMN company_id TYPE integer;
ALTER TABLE customers ALTER COLUMN company_id TYPE integer;
ALTER TABLE employees ALTER COLUMN company_id TYPE integer;
ALTER TABLE trucks ALTER COLUMN company_id TYPE integer;
ALTER TABLE trailers ALTER COLUMN company_id TYPE integer;
ALTER TABLE zones ALTER COLUMN company_id TYPE integer;
ALTER TABLE zone_pricing ALTER COLUMN company_id TYPE integer;
ALTER TABLE vendors ALTER COLUMN company_id TYPE integer;
ALTER TABLE vendor_groups ALTER COLUMN company_id TYPE integer;
ALTER TABLE carriers ALTER COLUMN company_id TYPE integer;
ALTER TABLE regions ALTER COLUMN company_id TYPE integer;
ALTER TABLE dispatch_codes ALTER COLUMN company_id TYPE integer;
ALTER TABLE equipment_types ALTER COLUMN company_id TYPE integer;
ALTER TABLE items ALTER COLUMN company_id TYPE integer;
ALTER TABLE hold_codes ALTER COLUMN company_id TYPE integer;
ALTER TABLE declination_codes ALTER COLUMN company_id TYPE integer;
ALTER TABLE field_codes_1 ALTER COLUMN company_id TYPE integer;
ALTER TABLE field_codes_2 ALTER COLUMN company_id TYPE integer;
ALTER TABLE field_codes_3 ALTER COLUMN company_id TYPE integer;
ALTER TABLE field_codes_4 ALTER COLUMN company_id TYPE integer;
ALTER TABLE field_codes_5 ALTER COLUMN company_id TYPE integer;
ALTER TABLE damage_areas ALTER COLUMN company_id TYPE integer;
ALTER TABLE damage_types ALTER COLUMN company_id TYPE integer;
ALTER TABLE damage_severities ALTER COLUMN company_id TYPE integer;
ALTER TABLE terms ALTER COLUMN company_id TYPE integer;
ALTER TABLE tax_codes ALTER COLUMN company_id TYPE integer;
ALTER TABLE chart_of_accounts ALTER COLUMN company_id TYPE integer;
ALTER TABLE orders ALTER COLUMN company_id TYPE integer;
ALTER TABLE orders ALTER COLUMN bill_customer_id TYPE integer;
ALTER TABLE orders ALTER COLUMN load_customer_id TYPE integer;
ALTER TABLE orders ALTER COLUMN drop_customer_id TYPE integer;
ALTER TABLE trips ALTER COLUMN company_id TYPE integer;
ALTER TABLE trips ALTER COLUMN truck_id TYPE integer;
ALTER TABLE trips ALTER COLUMN driver1_id TYPE integer;
ALTER TABLE trips ALTER COLUMN driver2_id TYPE integer;
ALTER TABLE order_vehicles ALTER COLUMN company_id TYPE integer;
ALTER TABLE order_vehicles ALTER COLUMN order_id TYPE integer;
ALTER TABLE order_vehicles ALTER COLUMN trip_id TYPE integer;
ALTER TABLE load_details ALTER COLUMN company_id TYPE integer;
ALTER TABLE load_details ALTER COLUMN trip_id TYPE integer;
ALTER TABLE load_details ALTER COLUMN order_id TYPE integer;
ALTER TABLE load_details ALTER COLUMN vehicle_id TYPE integer;
ALTER TABLE order_charges ALTER COLUMN company_id TYPE integer;
ALTER TABLE order_charges ALTER COLUMN order_id TYPE integer;
ALTER TABLE order_charges ALTER COLUMN vehicle_id TYPE integer;
ALTER TABLE order_charges ALTER COLUMN trip_id TYPE integer;
ALTER TABLE vehicle_damage ALTER COLUMN company_id TYPE integer;
ALTER TABLE vehicle_damage ALTER COLUMN order_id TYPE integer;
ALTER TABLE vehicle_damage ALTER COLUMN vehicle_id TYPE integer;
ALTER TABLE vehicle_damage ALTER COLUMN trip_id TYPE integer;
ALTER TABLE damage_details ALTER COLUMN company_id TYPE integer;
ALTER TABLE damage_details ALTER COLUMN vehicle_damage_id TYPE integer;
ALTER TABLE vehicle_notes ALTER COLUMN company_id TYPE integer;
ALTER TABLE vehicle_notes ALTER COLUMN vehicle_id TYPE integer;
ALTER TABLE trip_fuel ALTER COLUMN company_id TYPE integer;
ALTER TABLE trip_fuel ALTER COLUMN trip_id TYPE integer;
ALTER TABLE trip_expenses ALTER COLUMN company_id TYPE integer;
ALTER TABLE trip_expenses ALTER COLUMN trip_id TYPE integer;
ALTER TABLE trip_routes ALTER COLUMN company_id TYPE integer;
ALTER TABLE trip_routes ALTER COLUMN trip_id TYPE integer;
ALTER TABLE trip_routes ALTER COLUMN customer_id TYPE integer;
ALTER TABLE split_loads ALTER COLUMN company_id TYPE integer;
ALTER TABLE split_loads ALTER COLUMN order_id TYPE integer;
ALTER TABLE split_loads ALTER COLUMN vehicle_id TYPE integer;
ALTER TABLE split_loads ALTER COLUMN trip_id TYPE integer;
ALTER TABLE split_loads ALTER COLUMN orig_trip_id TYPE integer;
ALTER TABLE invoices ALTER COLUMN company_id TYPE integer;
ALTER TABLE invoices ALTER COLUMN customer_id TYPE integer;
ALTER TABLE invoices ALTER COLUMN order_id TYPE integer;
ALTER TABLE invoice_details ALTER COLUMN company_id TYPE integer;
ALTER TABLE invoice_details ALTER COLUMN invoice_id TYPE integer;
ALTER TABLE invoice_details ALTER COLUMN order_id TYPE integer;
ALTER TABLE invoice_details ALTER COLUMN vehicle_id TYPE integer;
ALTER TABLE credit_memos ALTER COLUMN company_id TYPE integer;
ALTER TABLE credit_memos ALTER COLUMN customer_id TYPE integer;
ALTER TABLE credit_memos ALTER COLUMN invoice_id TYPE integer;
ALTER TABLE payments ALTER COLUMN company_id TYPE integer;
ALTER TABLE payments ALTER COLUMN customer_id TYPE integer;
ALTER TABLE payment_details ALTER COLUMN company_id TYPE integer;
ALTER TABLE payment_details ALTER COLUMN payment_id TYPE integer;
ALTER TABLE payment_details ALTER COLUMN invoice_id TYPE integer;
ALTER TABLE damage_claims ALTER COLUMN company_id TYPE integer;
ALTER TABLE damage_claims ALTER COLUMN order_id TYPE integer;
ALTER TABLE damage_claims ALTER COLUMN vehicle_id TYPE integer;
ALTER TABLE damage_claims ALTER COLUMN trip_id TYPE integer;
ALTER TABLE accounts_payable ALTER COLUMN company_id TYPE integer;
ALTER TABLE accounts_payable ALTER COLUMN trip_id TYPE integer;
ALTER TABLE accounts_payable ALTER COLUMN employee_id TYPE integer;
ALTER TABLE accounts_payable ALTER COLUMN truck_id TYPE integer;
ALTER TABLE audit_log ALTER COLUMN user_id TYPE integer;
ALTER TABLE audit_log ALTER COLUMN record_id TYPE integer;
ALTER TABLE feedback ALTER COLUMN company_id TYPE integer;
ALTER TABLE feedback ALTER COLUMN user_id TYPE integer;
ALTER TABLE password_reset_tokens ALTER COLUMN user_id TYPE integer;
ALTER TABLE feedback_comments ALTER COLUMN feedback_id TYPE integer;
ALTER TABLE feedback_comments ALTER COLUMN user_id TYPE integer;
ALTER TABLE feedback_comments ALTER COLUMN company_id TYPE integer;
ALTER TABLE attachments ALTER COLUMN company_id TYPE integer;
ALTER TABLE attachments ALTER COLUMN entity_id TYPE integer;
ALTER TABLE attachments ALTER COLUMN uploaded_by TYPE integer;
ALTER TABLE loadboard_listings ALTER COLUMN poster_company_id TYPE integer;
ALTER TABLE loadboard_listings ALTER COLUMN poster_user_id TYPE integer;
ALTER TABLE loadboard_listings ALTER COLUMN source_order_id TYPE integer;
ALTER TABLE loadboard_listing_vehicles ALTER COLUMN listing_id TYPE integer;
ALTER TABLE loadboard_listing_vehicles ALTER COLUMN source_vehicle_id TYPE integer;
ALTER TABLE loadboard_claims ALTER COLUMN listing_id TYPE integer;
ALTER TABLE loadboard_claims ALTER COLUMN carrier_company_id TYPE integer;
ALTER TABLE loadboard_claims ALTER COLUMN carrier_user_id TYPE integer;
ALTER TABLE loadboard_claims ALTER COLUMN carrier_order_id TYPE integer;
ALTER TABLE loadboard_messages ALTER COLUMN claim_id TYPE integer;
ALTER TABLE loadboard_messages ALTER COLUMN sender_company_id TYPE integer;
ALTER TABLE loadboard_messages ALTER COLUMN sender_user_id TYPE integer;

-- ---- Restore id columns to integer ----
ALTER TABLE loadboard_messages ALTER COLUMN id TYPE integer;
ALTER TABLE loadboard_claims ALTER COLUMN id TYPE integer;
ALTER TABLE loadboard_listing_vehicles ALTER COLUMN id TYPE integer;
ALTER TABLE loadboard_listings ALTER COLUMN id TYPE integer;
ALTER TABLE attachments ALTER COLUMN id TYPE integer;
ALTER TABLE feedback_comments ALTER COLUMN id TYPE integer;
ALTER TABLE pending_registrations ALTER COLUMN id TYPE integer;
ALTER TABLE password_reset_tokens ALTER COLUMN id TYPE integer;
ALTER TABLE feedback ALTER COLUMN id TYPE integer;
ALTER TABLE audit_log ALTER COLUMN id TYPE integer;
ALTER TABLE accounts_payable ALTER COLUMN id TYPE integer;
ALTER TABLE damage_claims ALTER COLUMN id TYPE integer;
ALTER TABLE payment_details ALTER COLUMN id TYPE integer;
ALTER TABLE payments ALTER COLUMN id TYPE integer;
ALTER TABLE credit_memos ALTER COLUMN id TYPE integer;
ALTER TABLE invoice_details ALTER COLUMN id TYPE integer;
ALTER TABLE invoices ALTER COLUMN id TYPE integer;
ALTER TABLE split_loads ALTER COLUMN id TYPE integer;
ALTER TABLE trip_routes ALTER COLUMN id TYPE integer;
ALTER TABLE trip_expenses ALTER COLUMN id TYPE integer;
ALTER TABLE trip_fuel ALTER COLUMN id TYPE integer;
ALTER TABLE vehicle_notes ALTER COLUMN id TYPE integer;
ALTER TABLE damage_details ALTER COLUMN id TYPE integer;
ALTER TABLE vehicle_damage ALTER COLUMN id TYPE integer;
ALTER TABLE order_charges ALTER COLUMN id TYPE integer;
ALTER TABLE load_details ALTER COLUMN id TYPE integer;
ALTER TABLE order_vehicles ALTER COLUMN id TYPE integer;
ALTER TABLE trips ALTER COLUMN id TYPE integer;
ALTER TABLE orders ALTER COLUMN id TYPE integer;
ALTER TABLE chart_of_accounts ALTER COLUMN id TYPE integer;
ALTER TABLE tax_codes ALTER COLUMN id TYPE integer;
ALTER TABLE terms ALTER COLUMN id TYPE integer;
ALTER TABLE damage_severities ALTER COLUMN id TYPE integer;
ALTER TABLE damage_types ALTER COLUMN id TYPE integer;
ALTER TABLE damage_areas ALTER COLUMN id TYPE integer;
ALTER TABLE field_codes_5 ALTER COLUMN id TYPE integer;
ALTER TABLE field_codes_4 ALTER COLUMN id TYPE integer;
ALTER TABLE field_codes_3 ALTER COLUMN id TYPE integer;
ALTER TABLE field_codes_2 ALTER COLUMN id TYPE integer;
ALTER TABLE field_codes_1 ALTER COLUMN id TYPE integer;
ALTER TABLE declination_codes ALTER COLUMN id TYPE integer;
ALTER TABLE hold_codes ALTER COLUMN id TYPE integer;
ALTER TABLE color_codes ALTER COLUMN id TYPE integer;
ALTER TABLE vin_definitions ALTER COLUMN id TYPE integer;
ALTER TABLE vehicle_makes ALTER COLUMN id TYPE integer;
ALTER TABLE items ALTER COLUMN id TYPE integer;
ALTER TABLE equipment_types ALTER COLUMN id TYPE integer;
ALTER TABLE dispatch_codes ALTER COLUMN id TYPE integer;
ALTER TABLE regions ALTER COLUMN id TYPE integer;
ALTER TABLE carriers ALTER COLUMN id TYPE integer;
ALTER TABLE vendor_groups ALTER COLUMN id TYPE integer;
ALTER TABLE vendors ALTER COLUMN id TYPE integer;
ALTER TABLE zone_pricing ALTER COLUMN id TYPE integer;
ALTER TABLE zones ALTER COLUMN id TYPE integer;
ALTER TABLE trailers ALTER COLUMN id TYPE integer;
ALTER TABLE trucks ALTER COLUMN id TYPE integer;
ALTER TABLE employees ALTER COLUMN id TYPE integer;
ALTER TABLE customers ALTER COLUMN id TYPE integer;
ALTER TABLE users ALTER COLUMN id TYPE integer;
ALTER TABLE companies ALTER COLUMN id TYPE integer;

-- ---- Restore sequences to integer range ----
ALTER SEQUENCE loadboard_messages_id_seq AS integer;
ALTER SEQUENCE loadboard_claims_id_seq AS integer;
ALTER SEQUENCE loadboard_listing_vehicles_id_seq AS integer;
ALTER SEQUENCE loadboard_listings_id_seq AS integer;
ALTER SEQUENCE attachments_id_seq AS integer;
ALTER SEQUENCE feedback_comments_id_seq AS integer;
ALTER SEQUENCE pending_registrations_id_seq AS integer;
ALTER SEQUENCE password_reset_tokens_id_seq AS integer;
ALTER SEQUENCE feedback_id_seq AS integer;
ALTER SEQUENCE audit_log_id_seq AS integer;
ALTER SEQUENCE accounts_payable_id_seq AS integer;
ALTER SEQUENCE damage_claims_id_seq AS integer;
ALTER SEQUENCE payment_details_id_seq AS integer;
ALTER SEQUENCE payments_id_seq AS integer;
ALTER SEQUENCE credit_memos_id_seq AS integer;
ALTER SEQUENCE invoice_details_id_seq AS integer;
ALTER SEQUENCE invoices_id_seq AS integer;
ALTER SEQUENCE split_loads_id_seq AS integer;
ALTER SEQUENCE trip_routes_id_seq AS integer;
ALTER SEQUENCE trip_expenses_id_seq AS integer;
ALTER SEQUENCE trip_fuel_id_seq AS integer;
ALTER SEQUENCE vehicle_notes_id_seq AS integer;
ALTER SEQUENCE damage_details_id_seq AS integer;
ALTER SEQUENCE vehicle_damage_id_seq AS integer;
ALTER SEQUENCE order_charges_id_seq AS integer;
ALTER SEQUENCE load_details_id_seq AS integer;
ALTER SEQUENCE order_vehicles_id_seq AS integer;
ALTER SEQUENCE trips_id_seq AS integer;
ALTER SEQUENCE orders_id_seq AS integer;
ALTER SEQUENCE chart_of_accounts_id_seq AS integer;
ALTER SEQUENCE tax_codes_id_seq AS integer;
ALTER SEQUENCE terms_id_seq AS integer;
ALTER SEQUENCE damage_severities_id_seq AS integer;
ALTER SEQUENCE damage_types_id_seq AS integer;
ALTER SEQUENCE damage_areas_id_seq AS integer;
ALTER SEQUENCE field_codes_5_id_seq AS integer;
ALTER SEQUENCE field_codes_4_id_seq AS integer;
ALTER SEQUENCE field_codes_3_id_seq AS integer;
ALTER SEQUENCE field_codes_2_id_seq AS integer;
ALTER SEQUENCE field_codes_1_id_seq AS integer;
ALTER SEQUENCE declination_codes_id_seq AS integer;
ALTER SEQUENCE hold_codes_id_seq AS integer;
ALTER SEQUENCE color_codes_id_seq AS integer;
ALTER SEQUENCE vin_definitions_id_seq AS integer;
ALTER SEQUENCE vehicle_makes_id_seq AS integer;
ALTER SEQUENCE items_id_seq AS integer;
ALTER SEQUENCE equipment_types_id_seq AS integer;
ALTER SEQUENCE dispatch_codes_id_seq AS integer;
ALTER SEQUENCE regions_id_seq AS integer;
ALTER SEQUENCE carriers_id_seq AS integer;
ALTER SEQUENCE vendor_groups_id_seq AS integer;
ALTER SEQUENCE vendors_id_seq AS integer;
ALTER SEQUENCE zone_pricing_id_seq AS integer;
ALTER SEQUENCE zones_id_seq AS integer;
ALTER SEQUENCE trailers_id_seq AS integer;
ALTER SEQUENCE trucks_id_seq AS integer;
ALTER SEQUENCE employees_id_seq AS integer;
ALTER SEQUENCE customers_id_seq AS integer;
ALTER SEQUENCE users_id_seq AS integer;
ALTER SEQUENCE companies_id_seq AS integer;
```

### Step 2: Apply the migration locally

```bash
make migrate-up
```

Expected output: one line ending in `OK` for migration `018_bigserial.sql`. No errors.

If it fails with "sequence not found": a sequence name differs from `tablename_id_seq`. Run `\ds` in psql to list sequences and correct the name.

### Step 3: Verify the schema

```bash
psql $DATABASE_URL -c "\d users" | grep " id "
```

Expected: `id | bigint | ...`

### Step 4: Verify Go build

```bash
go build ./...
```

Expected: no errors. No Go code changes are needed — `int` is 64-bit on amd64, pgx v5 handles BIGINT ↔ int scanning.

### Step 5: Run all tests

```bash
go test ./internal/... -v 2>&1 | tail -30
```

Expected: all tests PASS (store tests needing DB will skip cleanly).

### Step 6: Commit

```bash
git add internal/database/migrations/018_bigserial.sql
git commit -m "feat: convert all primary keys and FK columns from SERIAL/INTEGER to BIGSERIAL/BIGINT"
```

### Step 7: Deploy

```bash
./scripts/deploy.sh
```

Expected: migration runs on production DB, server restarts cleanly.

---

## Notes

- **No Go model changes**: All model fields use `int` which is 64-bit on amd64. pgx v5 scans `BIGINT` into `*int` without modification.
- **No store changes**: `strconv.Atoi` and `parseID` helpers return `int` — still fine.
- **JWT**: `UserID` and `CompanyID` in claims are `int` — still fine.
- **auth.GetCompanyID**: returns `int` — still fine.
- **Advisory locks**: `pg_advisory_xact_lock($1, N)` takes `int8` (bigint) — passing Go `int` already works.
- **Down migration**: only safe on a dev DB with no IDs exceeding 2,147,483,647. Production should never run it.
