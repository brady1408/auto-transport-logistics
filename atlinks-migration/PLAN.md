# ATLinks Migration Plan: Clarion 6 to Go Web Application

## Context

ATLinks is a vehicle transport/logistics management system built in Clarion 6 (~2009), currently running as a multi-DLL Windows desktop app backed by MSSQL. The application manages the full lifecycle of vehicle transport: order creation, VIN tracking, trip/load dispatch, invoicing, payments, damage claims, and EDI communication with OEMs (Ford, GM, Chrysler, Honda). It has ~77 database tables across 7 modules (Dispatch, Accounting, EDI, Global Masters, Maintenance, Reporting, Utilities).

The goal is to build an MVP web application that replaces the desktop client for daily operations, using Go, PostgreSQL, REST, and HTMX.

---

## Tech Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| **Backend** | Go (1.22+ with new ServeMux) | User preference, excellent for this workload |
| **Database** | PostgreSQL | Multi-user concurrency, triggers for audit, free, handles 77 tables and 437MB dataset easily |
| **API** | REST/JSON | CRUD-heavy app, browser frontend, simplest tooling. No gRPC overhead for MVP |
| **Frontend** | HTMX + Go html/template | 80% of screens are browse-list + detail-form, which is exactly HTMX's sweet spot. No JS build pipeline. Complex widgets (order form with nested vehicles) can use Alpine.js or small Svelte islands |
| **Auth** | JWT + bcrypt | Replace the cleartext TOPSPEED U00 passwords |
| **Migrations** | goose | SQL migration files, up/down |
| **DB Driver** | pgx/v5 | Fastest pure Go PostgreSQL driver, no ORM |

---

## Key Architectural Decisions

### 1. Table Naming: Rename to readable names
The Clarion shorthand (D00, D10, G00, A00) becomes `orders`, `order_vehicles`, `customers`, `invoices`, etc. A `legacy_id` column preserves the original IDs for migration validation.

### 2. Audit Trail: Go middleware, not DB triggers
The current system uses MSSQL triggers that copy full rows to `archive_*` tables. The new system uses a single `audit_log` table with JSONB `old_values`/`new_values` columns, populated by Go middleware that has access to the authenticated user context.

### 3. Auth: Fresh accounts, no password migration
Current U00 passwords are stored as cleartext uppercase strings in an encrypted TPS file. All users get fresh accounts with bcrypt-hashed passwords.

---

## MVP Scope

### IN (must-have for daily operations):

**Foundation:** Auth, company setup, customer/employee/truck/zone CRUD, lookup tables

**Dispatch Core:** Orders (D00), vehicles/VINs (D10) with VIN decoding, trips/loads (D20), load details (D30), vehicle status tracking (Waiting->Scheduled->Loaded->Delivered->Confirmed), damage logging, trip fuel/mileage, other charges

**Accounting MVP:** Invoice generation from orders, payment recording, credit memos, basic AP, basic damage claims

**Reporting:** Top 10 most-used reports, CSV export, dashboard, global VIN search

### OUT (post-MVP):

- EDI integrations (COPAC, VISTA, ACES, Honda) - each is a 2-4 week sub-project
- QuickBooks sync (use CSV bridge initially)
- Rand McNally mileage (use free distance API or manual entry)
- DriverTech/Qualcomm telematics
- General Ledger (A80/A82)
- Driver/Truck earnings calculations (A75/A76)
- Fuel card imports (A70/A72)
- PDF delivery receipts with fancy formatting (start with printable HTML)
- Signature capture, document imaging
- 80+ reports beyond the top 10

---

## Project Structure

```
atlinks/
  cmd/server/main.go
  internal/
    config/config.go
    auth/                    # JWT, bcrypt, middleware
    audit/audit.go           # Audit logging service
    database/
      postgres.go            # Connection pool
      migrate.go             # Migration runner
    models/                  # Go structs (domain types)
      company.go, customer.go, employee.go, truck.go,
      zone.go, order.go, vehicle.go, trip.go,
      load_detail.go, invoice.go, payment.go, ...
    store/                   # Repository layer (raw SQL + pgx)
      customer_store.go, order_store.go, trip_store.go, ...
    service/                 # Business logic
      order_service.go, dispatch_service.go,
      invoice_service.go, vin_decoder.go, pricing_service.go
    handler/                 # HTTP handlers
      auth_handler.go, customer_handler.go,
      order_handler.go, trip_handler.go, ...
    templates/               # Go HTML templates
      layout/base.html, nav.html
      pages/login.html, dashboard.html,
        customers/{list,form}.html
        orders/{list,form,vehicles}.html
        trips/{list,form,load_details}.html
        invoices/{list,form}.html
    static/css/, js/htmx.min.js
  migrations/                # SQL files (goose)
    001_initial_schema.up.sql
    002_seed_lookups.up.sql
  scripts/
    migrate_mssql.go         # One-time data migration from MSSQL->PG
  go.mod, Makefile
```

No ORM. Raw SQL with pgx. Go 1.22+ `net/http` router (or chi if preferred).

---

## Migration Phases

### Phase 0: Infrastructure (Week 1)
- [ ] Init Git repo, Go module, project structure
- [ ] PostgreSQL via Docker for local dev
- [ ] Write initial schema migration (all MVP tables) from ATLinks.TXD
- [ ] Restore MSSQL .bak to Docker MSSQL for reference
- [ ] Write Go migration script: MSSQL -> PostgreSQL (table mapping, type conversions)
- [ ] Run migration, validate row counts + spot-check records

### Phase 1: Foundation / Global Masters (Weeks 2-4)
- [ ] Auth system: login page, JWT, bcrypt, middleware, audit logging
- [ ] Base HTMX layout with navigation matching Clarion menu structure
- [ ] Company CRUD (C00 -> companies)
- [ ] User management (admin only)
- [ ] Customer CRUD with browse/search (G00 - most complex master)
- [ ] Employee/Driver CRUD (G10)
- [ ] Truck + Trailer CRUD (G20, G22)
- [ ] Zone + Zone Pricing CRUD (G30, G32)
- [ ] All lookup tables: equipment types, damage codes, dispatch codes, VIN defs, items, classes, terms, tax codes

### Phase 2: Core Dispatch (Weeks 5-9)
- [ ] Order CRUD (D00) - complex form with 3 customer refs (Bill/Load/Drop)
- [ ] Customer lookup/typeahead within order form
- [ ] Order browse with filters (active/inactive, customer, date, zone)
- [ ] Vehicle/VIN management within orders (D10) - nested list
- [ ] VIN decoder using G43/G75-G78 lookup tables
- [ ] Vehicle status workflow: Waiting->Scheduled->Loaded->Delivered->Confirmed
- [ ] Other charges (D13)
- [ ] Trip/Load creation (D20) - assign truck, drivers, dates, rates
- [ ] Load details (D30) - assign vehicles to trips
- [ ] Split loads (D40)
- [ ] Trip fuel/mileage (D23), expenses (D24), routing (D26)
- [ ] Damage logging (D33, D34) with area/type/severity codes

### Phase 3: Accounting MVP (Weeks 10-13)
- [ ] Invoice generation from completed orders (A00, A02)
- [ ] Invoice browse/search/edit
- [ ] Printable invoice view (HTML)
- [ ] Payment recording (A20, A30) - apply payments to invoices
- [ ] Credit memos (A10)
- [ ] Basic AP (A50)
- [ ] Basic damage claims (A40)

### Phase 4: Reporting + Polish (Weeks 14-16)
- [ ] Dashboard: active orders by status, trips in progress, aging AR
- [ ] Global VIN search across orders/vehicles
- [ ] Top 10 reports: delivery receipt, invoice, AR aging, driver settlement, revenue by customer, trip summary
- [ ] CSV export for all report data
- [ ] Performance tuning (indexes matching original KEY definitions)
- [ ] Bug fixes, UAT

---

## Data Migration Strategy

1. **Restore MSSQL backup** via Docker: `mcr.microsoft.com/mssql/server:2019-latest`
2. **Go migration script** connects to both MSSQL and PostgreSQL:
   - Maps table/column names (D00 -> orders, BillG00id -> bill_customer_id)
   - Converts Clarion DATE (days since 1800-12-28) -> PostgreSQL date
   - Converts Clarion TIME (centiseconds since midnight) -> PostgreSQL time
   - Trims CSTRING trailing nulls
   - Maps BYTE 0/1 -> boolean
   - Preserves original IDs in `legacy_id` columns
3. **Migration order** respects FK dependencies: companies -> zones -> customers -> employees -> trucks -> orders -> vehicles -> trips -> load_details -> invoices -> payments
4. **U00/S00 users**: Create fresh accounts manually (likely <20 users)

---

## Core Table Mapping (Key Tables Only)

| Clarion | PostgreSQL | Purpose |
|---------|-----------|---------|
| C00 | `companies` | Company master |
| G00 | `customers` | Customer master (3 refs per order) |
| G10 | `employees` | Employees/drivers |
| G20 | `trucks` | Truck fleet |
| G30 | `zones` | Geographic zones |
| G32 | `zone_pricing` | Zone-to-zone rates |
| D00 | `orders` | Order master (bill/load/drop customers) |
| D10 | `order_vehicles` | VINs within an order |
| D20 | `trips` | Loads/trips (truck + driver assignment) |
| D30 | `load_details` | Vehicle-to-trip assignment |
| D13 | `order_charges` | Additional charges |
| D33 | `vehicle_damage` | Damage records |
| A00 | `invoices` | Invoice headers |
| A02 | `invoice_details` | Invoice line items |
| A20 | `payments` | Payment headers |
| A30 | `payment_details` | Payment-to-invoice application |
| A10 | `credit_memos` | Credit memos |
| A40 | `damage_claims` | Damage claims |
| A50 | `accounts_payable` | AP records |
| U00 | `users` | Auth (rebuilt from scratch) |

---

## Verification Plan

1. **Data migration**: Row counts match between MSSQL and PostgreSQL for every table
2. **Auth**: Login, logout, JWT expiry, role-based access (admin vs user)
3. **CRUD smoke test**: Create/read/update/delete for each entity type
4. **Order workflow**: Create order -> add VINs -> create trip -> assign vehicles -> mark delivered -> generate invoice -> record payment
5. **Audit trail**: Verify audit_log entries for all mutations
6. **Reports**: Compare output of top 10 reports against Clarion originals
7. **Concurrent access**: Multiple users editing orders simultaneously
