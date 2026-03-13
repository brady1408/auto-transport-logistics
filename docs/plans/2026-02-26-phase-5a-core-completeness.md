# Phase 5A: Core Completeness Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the 10 Phase 5A features that complete daily operational gaps: Vendor CRUD, Driver/Truck earnings adjustments, Inbound vehicle status, Waiting Grid dispatch board, Truck expiration warnings, Attachments browse, Batch invoice recalc UI, Customer statements, and Invoice/Payment posting.

**Architecture:** All features follow the established pattern: migration → model → store → handler → templ components → main.go wiring. No new frameworks — pgx/v5, templ, HTMX, raw SQL. The `Inbound` status change is additive (no schema change needed). Earnings adjustments (A75/A76) require new migration + full CRUD stack. Features with existing service layer (recalc, posting) need only a UI layer.

**Tech Stack:** Go 1.22+, pgx/v5, templ, HTMX, Alpine.js, PostgreSQL 16, goose migrations.

---

## Conventions (read before any task)

- **Migrations**: Create `internal/database/migrations/019_<name>.sql` with `-- +goose Up` and `-- +goose Down` sections
- **IDs**: Use `BIGSERIAL` / `bigint` (not SERIAL/INTEGER — see migration 018)
- **Soft deletes**: `deleted_at TIMESTAMPTZ` column + `deleted_at IS NULL` in all queries
- **company_id isolation**: Every store method calls `auth.GetCompanyID(ctx)` and adds `AND company_id = $N`
- **templ compile**: After editing `.templ` files run `make generate` then `go build ./...`
- **Test build**: `go build ./...` after every task — catch compile errors early
- **Commit message style**: `feat: <description>` (lowercase, no Claude attribution)

---

## Task 1: Vendor Master File (G50)

**Context:** The `vendors` table exists in the schema (migration 001) but has no model, store, handler, or UI. The vendors table has: id, legacy_id, name, address, address2, city, state, zip, phone, fax, contact, terms, tax_id. No company_id column — need to add one via migration.

**Files:**
- Create: `internal/database/migrations/019_vendor_company_id.sql`
- Create: `internal/models/vendor.go`
- Create: `internal/store/vendor_store.go`
- Create: `internal/handler/vendor_handler.go`
- Create: `internal/handler/components/vendors/list.templ`
- Create: `internal/handler/components/vendors/table.templ`
- Create: `internal/handler/components/vendors/form.templ`
- Modify: `internal/handler/components/nav.templ` (add Vendors to Global dropdown)
- Modify: `cmd/server/main.go` (wire store + handler)

**Step 1: Write migration**

```sql
-- internal/database/migrations/019_vendor_company_id.sql
-- +goose Up
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS company_id bigint REFERENCES companies(id);
ALTER TABLE vendors ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
UPDATE vendors SET company_id = 1 WHERE company_id IS NULL;
ALTER TABLE vendors ALTER COLUMN company_id SET NOT NULL;
CREATE INDEX idx_vendors_company ON vendors (company_id);
CREATE TRIGGER trg_vendors_updated_at_fix BEFORE UPDATE ON vendors
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- +goose Down
ALTER TABLE vendors DROP COLUMN IF EXISTS company_id;
ALTER TABLE vendors DROP COLUMN IF EXISTS deleted_at;
```

**Step 2: Run migration**

```bash
make migrate-up
```

Expected: "OK   019_vendor_company_id.sql"

**Step 3: Write model**

```go
// internal/models/vendor.go
package models

import "time"

type Vendor struct {
	ID        int        `json:"id"`
	CompanyID int        `json:"company_id"`
	LegacyID  *int       `json:"legacy_id,omitempty"`
	Name      string     `json:"name"`
	Address   *string    `json:"address,omitempty"`
	Address2  *string    `json:"address2,omitempty"`
	City      *string    `json:"city,omitempty"`
	State     *string    `json:"state,omitempty"`
	Zip       *string    `json:"zip,omitempty"`
	Phone     *string    `json:"phone,omitempty"`
	Fax       *string    `json:"fax,omitempty"`
	Contact   *string    `json:"contact,omitempty"`
	Terms     *string    `json:"terms,omitempty"`
	TaxID     *string    `json:"tax_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type VendorFilter struct {
	Search   string
	Page     int
	PageSize int
}

type VendorListResult struct {
	Items      []Vendor
	TotalCount int
	Page       int
	PageSize   int
}
```

**Step 4: Write store**

```go
// internal/store/vendor_store.go
package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VendorStore struct {
	pool *pgxpool.Pool
}

func NewVendorStore(pool *pgxpool.Pool) *VendorStore {
	return &VendorStore{pool: pool}
}

const vendorColumns = `id, company_id, legacy_id, name, address, address2, city, state, zip,
	phone, fax, contact, terms, tax_id, created_at, updated_at`

func scanVendor(row interface{ Scan(dest ...any) error }) (*models.Vendor, error) {
	var v models.Vendor
	err := row.Scan(
		&v.ID, &v.CompanyID, &v.LegacyID, &v.Name, &v.Address, &v.Address2,
		&v.City, &v.State, &v.Zip, &v.Phone, &v.Fax, &v.Contact,
		&v.Terms, &v.TaxID, &v.CreatedAt, &v.UpdatedAt,
	)
	return &v, err
}

func (s *VendorStore) List(ctx context.Context, f models.VendorFilter) (*models.VendorListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	qb := newQueryBuilder()
	qb.Add("company_id = ?", companyID)
	qb.AddRaw("deleted_at IS NULL")
	if f.Search != "" {
		qb.Add("(name ILIKE ? OR contact ILIKE ?)", "%"+f.Search+"%", "%"+f.Search+"%")
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM vendors "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count vendors: %w", err)
	}

	query := fmt.Sprintf("SELECT %s FROM vendors %s ORDER BY name %s",
		vendorColumns, qb.Where(), qb.Paginate(f.PageSize, f.Page))
	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list vendors: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Vendor, error) {
		v, err := scanVendor(row)
		if err != nil {
			return models.Vendor{}, err
		}
		return *v, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan vendors: %w", err)
	}
	return &models.VendorListResult{Items: items, TotalCount: total, Page: f.Page, PageSize: f.PageSize}, nil
}

func (s *VendorStore) GetByID(ctx context.Context, id int) (*models.Vendor, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT %s FROM vendors WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL", vendorColumns)
	v, err := scanVendor(s.pool.QueryRow(ctx, query, id, companyID))
	if err != nil {
		return nil, fmt.Errorf("get vendor %d: %w", id, err)
	}
	return v, nil
}

func (s *VendorStore) Create(ctx context.Context, v *models.Vendor) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	v.CompanyID = companyID
	return s.pool.QueryRow(ctx,
		`INSERT INTO vendors (company_id, name, address, address2, city, state, zip, phone, fax, contact, terms, tax_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, created_at, updated_at`,
		v.CompanyID, v.Name, v.Address, v.Address2, v.City, v.State, v.Zip,
		v.Phone, v.Fax, v.Contact, v.Terms, v.TaxID,
	).Scan(&v.ID, &v.CreatedAt, &v.UpdatedAt)
}

func (s *VendorStore) Update(ctx context.Context, v *models.Vendor) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE vendors SET name=$1, address=$2, address2=$3, city=$4, state=$5, zip=$6,
		 phone=$7, fax=$8, contact=$9, terms=$10, tax_id=$11
		 WHERE id=$12 AND company_id=$13 AND deleted_at IS NULL`,
		v.Name, v.Address, v.Address2, v.City, v.State, v.Zip,
		v.Phone, v.Fax, v.Contact, v.Terms, v.TaxID,
		v.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update vendor %d: %w", v.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("vendor %d not found", v.ID)
	}
	return nil
}

func (s *VendorStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		"UPDATE vendors SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL",
		id, companyID,
	)
	if err != nil {
		return fmt.Errorf("delete vendor %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("vendor %d not found", id)
	}
	return nil
}
```

**Step 5: Write handler**

```go
// internal/handler/vendor_handler.go
package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/brady1408/atlinks/internal/handler/components/vendors"
	"github.com/brady1408/atlinks/internal/models"
)

type vendorStore interface {
	List(ctx context.Context, f models.VendorFilter) (*models.VendorListResult, error)
	GetByID(ctx context.Context, id int) (*models.Vendor, error)
	Create(ctx context.Context, v *models.Vendor) error
	Update(ctx context.Context, v *models.Vendor) error
	Delete(ctx context.Context, id int) error
}

type VendorHandler struct {
	store vendorStore
	deps  *Deps
}

func NewVendorHandler(store vendorStore, deps *Deps) *VendorHandler {
	return &VendorHandler{store: store, deps: deps}
}

func (h *VendorHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /global/vendors", h.list)
	mux.HandleFunc("GET /global/vendors/new", h.newForm)
	mux.HandleFunc("POST /global/vendors", h.create)
	mux.HandleFunc("GET /global/vendors/{id}/edit", h.editForm)
	mux.HandleFunc("PUT /global/vendors/{id}", h.update)
	mux.HandleFunc("DELETE /global/vendors/{id}", h.delete)
}

func (h *VendorHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := models.VendorFilter{
		Search:   r.URL.Query().Get("search"),
		Page:     intParam(r, "page", 1),
		PageSize: 25,
	}
	result, err := h.store.List(r.Context(), filter)
	if err != nil {
		serverError(w, err)
		return
	}
	if isHTMX(r) {
		h.deps.renderTempl(w, r, vendors.Table(*result, filter))
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, vendors.ListPage(pg, *result, filter))
}

func (h *VendorHandler) newForm(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, vendors.FormPage(pg, &models.Vendor{}, true, ""))
}

func (h *VendorHandler) create(w http.ResponseWriter, r *http.Request) {
	v := bindVendorForm(r)
	if v.Name == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, vendors.FormPage(pg, v, true, "Name is required"))
		return
	}
	if err := h.store.Create(r.Context(), v); err != nil {
		log.Printf("create vendor: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, vendors.FormPage(pg, v, true, "Failed to create vendor"))
		return
	}
	h.deps.Audit.Log(r.Context(), "vendors", v.ID, "INSERT", nil, v)
	h.deps.setFlash(w, "Vendor created successfully")
	redirect(w, r, "/global/vendors")
}

func (h *VendorHandler) editForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	v, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Vendor not found", http.StatusNotFound)
		return
	}
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, vendors.FormPage(pg, v, false, ""))
}

func (h *VendorHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Vendor not found", http.StatusNotFound)
		return
	}
	v := bindVendorForm(r)
	v.ID = id
	if v.Name == "" {
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, vendors.FormPage(pg, v, false, "Name is required"))
		return
	}
	if err := h.store.Update(r.Context(), v); err != nil {
		log.Printf("update vendor: %v", err)
		pg := h.deps.pageContext(w, r)
		h.deps.renderTempl(w, r, vendors.FormPage(pg, v, false, "Failed to update vendor"))
		return
	}
	h.deps.Audit.Log(r.Context(), "vendors", v.ID, "UPDATE", old, v)
	h.deps.setFlash(w, "Vendor updated successfully")
	redirect(w, r, "/global/vendors")
}

func (h *VendorHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	old, err := h.store.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Vendor not found", http.StatusNotFound)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}
	h.deps.Audit.Log(r.Context(), "vendors", id, "DELETE", old, nil)
	h.deps.setFlash(w, "Vendor deleted")
	redirect(w, r, "/global/vendors")
}

func bindVendorForm(r *http.Request) *models.Vendor {
	return &models.Vendor{
		Name:     formStringRequired(r, "name"),
		Address:  formString(r, "address"),
		Address2: formString(r, "address2"),
		City:     formString(r, "city"),
		State:    formString(r, "state"),
		Zip:      formString(r, "zip"),
		Phone:    formString(r, "phone"),
		Fax:      formString(r, "fax"),
		Contact:  formString(r, "contact"),
		Terms:    formString(r, "terms"),
		TaxID:    formString(r, "tax_id"),
	}
}
```

**Step 6: Write templ components**

Create `internal/handler/components/vendors/list.templ`:
```go
package vendors

import (
	"github.com/brady1408/atlinks/internal/handler/components"
	"github.com/brady1408/atlinks/internal/models"
)

templ ListPage(pg components.PageContext, result models.VendorListResult, filter models.VendorFilter) {
	@components.Layout(pg, "Vendors - ATLinks") {
		<div class="page-header">
			<h1>Vendors</h1>
			<a href="/global/vendors/new" class="btn btn-primary">New Vendor</a>
		</div>
		<div class="filter-bar">
			<div class="form-group">
				<label for="search">Search</label>
				<input
					type="text" id="search" name="search" class="form-control search-input"
					placeholder="Name or contact..."
					value={ filter.Search }
					hx-get="/global/vendors"
					hx-trigger="keyup changed delay:300ms"
					hx-target="#vendor-table"
					hx-include=".filter-bar [name]"
					hx-push-url="true"
				/>
			</div>
		</div>
		<div id="vendor-table">
			@Table(result, filter)
		</div>
	}
}
```

Create `internal/handler/components/vendors/table.templ`:
```go
package vendors

import (
	"fmt"
	"github.com/brady1408/atlinks/internal/handler/components"
	"github.com/brady1408/atlinks/internal/models"
)

templ Table(result models.VendorListResult, filter models.VendorFilter) {
	<div class="table-container">
		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Contact</th>
					<th>City</th>
					<th>State</th>
					<th>Phone</th>
					<th>Terms</th>
					<th class="text-right">Actions</th>
				</tr>
			</thead>
			<tbody>
				if len(result.Items) > 0 {
					for _, v := range result.Items {
						<tr>
							<td><a href={ templ.SafeURL(fmt.Sprintf("/global/vendors/%d/edit", v.ID)) }>{ v.Name }</a></td>
							<td>{ components.Deref(v.Contact) }</td>
							<td>{ components.Deref(v.City) }</td>
							<td>{ components.Deref(v.State) }</td>
							<td>{ components.FormatPhone(v.Phone) }</td>
							<td>{ components.Deref(v.Terms) }</td>
							<td class="text-right">
								<div class="btn-group">
									<a href={ templ.SafeURL(fmt.Sprintf("/global/vendors/%d/edit", v.ID)) } class="btn btn-sm">Edit</a>
									<button
										class="btn btn-sm btn-danger"
										hx-delete={ fmt.Sprintf("/global/vendors/%d", v.ID) }
										hx-confirm={ fmt.Sprintf("Delete vendor %s?", v.Name) }
									>Delete</button>
								</div>
							</td>
						</tr>
					}
				} else {
					<tr><td colspan="7" class="text-center text-muted">No vendors found</td></tr>
				}
			</tbody>
		</table>
	</div>
	if len(result.Items) > 0 {
		@components.Pagination(components.NewPaginationData(
			result.Page, result.PageSize, result.TotalCount,
			"/global/vendors", "#vendor-table",
		))
	}
}
```

Create `internal/handler/components/vendors/form.templ`:
```go
package vendors

import (
	"fmt"
	"github.com/brady1408/atlinks/internal/handler/components"
	"github.com/brady1408/atlinks/internal/models"
)

templ FormPage(pg components.PageContext, vendor *models.Vendor, isNew bool, errMsg string) {
	@components.Layout(pg, vendorFormTitle(isNew)+" - ATLinks") {
		<div class="page-header">
			<h1>
				if isNew {
					New Vendor
				} else {
					Edit Vendor: { vendor.Name }
				}
			</h1>
			<a href="/global/vendors" class="btn">Back to List</a>
		</div>
		if errMsg != "" {
			<div class="alert alert-danger">{ errMsg }</div>
		}
		<form
			method="POST"
			if isNew {
				action="/global/vendors"
				hx-post="/global/vendors"
			} else {
				action={ templ.SafeURL(fmt.Sprintf("/global/vendors/%d/edit", vendor.ID)) }
				hx-put={ fmt.Sprintf("/global/vendors/%d", vendor.ID) }
			}
		>
			@components.CSRFField(pg.CSRFToken)
			<fieldset>
				<legend>Vendor Information</legend>
				<div class="form-row">
					<div class="form-group col-span-2">
						<label for="name">Name *</label>
						<input type="text" id="name" name="name" class="form-control" value={ vendor.Name } maxlength="30" required/>
					</div>
					<div class="form-group">
						<label for="contact">Contact</label>
						<input type="text" id="contact" name="contact" class="form-control" value={ components.Deref(vendor.Contact) } maxlength="20"/>
					</div>
					<div class="form-group">
						<label for="terms">Terms</label>
						<input type="text" id="terms" name="terms" class="form-control" value={ components.Deref(vendor.Terms) } maxlength="20"/>
					</div>
				</div>
				<div class="form-row">
					<div class="form-group">
						<label for="tax_id">Tax ID</label>
						<input type="text" id="tax_id" name="tax_id" class="form-control" value={ components.Deref(vendor.TaxID) } maxlength="15"/>
					</div>
					<div class="form-group">
						<label for="phone">Phone</label>
						<input type="text" id="phone" name="phone" class="form-control" value={ components.Deref(vendor.Phone) } maxlength="10"/>
					</div>
					<div class="form-group">
						<label for="fax">Fax</label>
						<input type="text" id="fax" name="fax" class="form-control" value={ components.Deref(vendor.Fax) } maxlength="10"/>
					</div>
				</div>
			</fieldset>
			<fieldset>
				<legend>Address</legend>
				<div class="form-row">
					<div class="form-group col-span-2">
						<label for="address">Address</label>
						<input type="text" id="address" name="address" class="form-control" value={ components.Deref(vendor.Address) } maxlength="30"/>
					</div>
				</div>
				<div class="form-row">
					<div class="form-group col-span-2">
						<label for="address2">Address 2</label>
						<input type="text" id="address2" name="address2" class="form-control" value={ components.Deref(vendor.Address2) } maxlength="30"/>
					</div>
				</div>
				<div class="form-row">
					<div class="form-group">
						<label for="city">City</label>
						<input type="text" id="city" name="city" class="form-control" value={ components.Deref(vendor.City) } maxlength="25"/>
					</div>
					<div class="form-group">
						<label for="state">State</label>
						<input type="text" id="state" name="state" class="form-control" value={ components.Deref(vendor.State) } maxlength="2" style="text-transform:uppercase"/>
					</div>
					<div class="form-group">
						<label for="zip">Zip</label>
						<input type="text" id="zip" name="zip" class="form-control" value={ components.Deref(vendor.Zip) } maxlength="10"/>
					</div>
				</div>
			</fieldset>
			<div class="form-actions">
				<button type="submit" class="btn btn-primary">
					if isNew { Create Vendor } else { Save Changes }
				</button>
				<a href="/global/vendors" class="btn">Cancel</a>
			</div>
		</form>
	}
}

func vendorFormTitle(isNew bool) string {
	if isNew {
		return "New Vendor"
	}
	return "Edit Vendor"
}
```

**Step 7: Wire in main.go**

In `initRoutes`, after the `handler.NewTruckHandler(...)` line, add:
```go
handler.NewVendorHandler(store.NewVendorStore(pool), deps).Register(protectedMux)
```

**Step 8: Add to nav**

In `internal/handler/components/nav.templ`, inside the Global dropdown after `<a href="/global/trucks"...>`, add:
```html
<a href="/global/vendors" class="dropdown-item">Vendors</a>
```
And in the mobile menu, after `<a href="/global/trucks">Trucks</a>`, add:
```html
<a href="/global/vendors">Vendors</a>
```

**Step 9: Generate templ and verify build**

```bash
make generate
go build ./...
```
Expected: no errors.

**Step 10: Test manually**

```bash
make run
# Navigate to http://localhost:8080/global/vendors
# Create a vendor, edit it, delete it
```

**Step 11: Commit**

```bash
git add internal/database/migrations/019_vendor_company_id.sql \
        internal/models/vendor.go \
        internal/store/vendor_store.go \
        internal/handler/vendor_handler.go \
        internal/handler/components/vendors/ \
        internal/handler/components/nav.templ \
        internal/handler/components/nav_templ.go \
        cmd/server/main.go
git commit -m "feat: add vendor master file (G50) CRUD"
```

---

## Task 2: Driver Earnings Add/Deduct (A75) and Truck Earnings Add/Deduct (A76)

**Context:** These are manual adjustment records for driver and truck pay — e.g., a bonus, fuel advance deduction, or damage charge against settlement. Both follow identical patterns. Build them together. They attach to employees (A75) and trucks (A76) respectively.

**Files:**
- Create: `internal/database/migrations/020_earnings_adjustments.sql`
- Create: `internal/models/earnings_adjustment.go`
- Create: `internal/store/earnings_adjustment_store.go`
- Create: `internal/handler/earnings_adjustment_handler.go`
- Create: `internal/handler/components/earnings/` (list.templ, table.templ, form.templ)
- Modify: `internal/handler/components/nav.templ` (add to Accounting dropdown)
- Modify: `cmd/server/main.go`

**Step 1: Write migration**

```sql
-- internal/database/migrations/020_earnings_adjustments.sql
-- +goose Up
CREATE TABLE driver_earnings_adjustments (
    id          BIGSERIAL PRIMARY KEY,
    company_id  bigint NOT NULL REFERENCES companies(id),
    employee_id bigint NOT NULL REFERENCES employees(id),
    adj_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    description VARCHAR(50) NOT NULL,
    adj_type    VARCHAR(3) NOT NULL DEFAULT 'Add' CHECK (adj_type IN ('Add','Ded')),
    amount      NUMERIC(10,2) NOT NULL DEFAULT 0,
    reference   VARCHAR(20),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);
CREATE TRIGGER trg_driver_earnings_updated_at BEFORE UPDATE ON driver_earnings_adjustments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE INDEX idx_driver_earnings_employee ON driver_earnings_adjustments (company_id, employee_id, adj_date);

CREATE TABLE truck_earnings_adjustments (
    id          BIGSERIAL PRIMARY KEY,
    company_id  bigint NOT NULL REFERENCES companies(id),
    truck_id    bigint NOT NULL REFERENCES trucks(id),
    adj_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    description VARCHAR(50) NOT NULL,
    adj_type    VARCHAR(3) NOT NULL DEFAULT 'Add' CHECK (adj_type IN ('Add','Ded')),
    amount      NUMERIC(10,2) NOT NULL DEFAULT 0,
    reference   VARCHAR(20),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);
CREATE TRIGGER trg_truck_earnings_updated_at BEFORE UPDATE ON truck_earnings_adjustments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE INDEX idx_truck_earnings_truck ON truck_earnings_adjustments (company_id, truck_id, adj_date);

-- +goose Down
DROP TABLE IF EXISTS truck_earnings_adjustments;
DROP TABLE IF EXISTS driver_earnings_adjustments;
```

**Step 2: Run migration**

```bash
make migrate-up
```

**Step 3: Write model**

```go
// internal/models/earnings_adjustment.go
package models

import "time"

type DriverEarningsAdj struct {
	ID          int        `json:"id"`
	CompanyID   int        `json:"company_id"`
	EmployeeID  int        `json:"employee_id"`
	EmployeeName string    `json:"employee_name,omitempty"` // joined
	AdjDate     time.Time  `json:"adj_date"`
	Description string     `json:"description"`
	AdjType     string     `json:"adj_type"` // "Add" or "Ded"
	Amount      string     `json:"amount"`
	Reference   *string    `json:"reference,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TruckEarningsAdj struct {
	ID           int        `json:"id"`
	CompanyID    int        `json:"company_id"`
	TruckID      int        `json:"truck_id"`
	TruckNumber  string     `json:"truck_number,omitempty"` // joined
	AdjDate      time.Time  `json:"adj_date"`
	Description  string     `json:"description"`
	AdjType      string     `json:"adj_type"` // "Add" or "Ded"
	Amount       string     `json:"amount"`
	Reference    *string    `json:"reference,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type EarningsAdjFilter struct {
	EntityID int    // employee_id or truck_id (0 = all)
	DateFrom string
	DateTo   string
	Page     int
	PageSize int
}

type DriverEarningsAdjResult struct {
	Items      []DriverEarningsAdj
	TotalCount int
	Page       int
	PageSize   int
}

type TruckEarningsAdjResult struct {
	Items      []TruckEarningsAdj
	TotalCount int
	Page       int
	PageSize   int
}
```

**Step 4: Write store**

```go
// internal/store/earnings_adjustment_store.go
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EarningsAdjStore struct {
	pool *pgxpool.Pool
}

func NewEarningsAdjStore(pool *pgxpool.Pool) *EarningsAdjStore {
	return &EarningsAdjStore{pool: pool}
}

// --- Driver earnings ---

func (s *EarningsAdjStore) ListDriver(ctx context.Context, f models.EarningsAdjFilter) (*models.DriverEarningsAdjResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	qb := newQueryBuilder()
	qb.Add("d.company_id = ?", companyID)
	qb.AddRaw("d.deleted_at IS NULL")
	if f.EntityID > 0 {
		qb.Add("d.employee_id = ?", f.EntityID)
	}
	if f.DateFrom != "" {
		qb.Add("d.adj_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		qb.Add("d.adj_date <= ?", f.DateTo)
	}

	var total int
	if err := s.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM driver_earnings_adjustments d "+qb.Where(), qb.Args()...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count driver adjustments: %w", err)
	}

	query := fmt.Sprintf(`SELECT d.id, d.company_id, d.employee_id,
		COALESCE(e.name, '') as employee_name,
		d.adj_date, d.description, d.adj_type, d.amount::text, d.reference, d.created_at, d.updated_at
		FROM driver_earnings_adjustments d
		LEFT JOIN employees e ON e.id = d.employee_id
		%s ORDER BY d.adj_date DESC, d.id DESC %s`,
		qb.Where(), qb.Paginate(f.PageSize, f.Page))
	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list driver adjustments: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.DriverEarningsAdj, error) {
		var a models.DriverEarningsAdj
		err := row.Scan(&a.ID, &a.CompanyID, &a.EmployeeID, &a.EmployeeName,
			&a.AdjDate, &a.Description, &a.AdjType, &a.Amount, &a.Reference,
			&a.CreatedAt, &a.UpdatedAt)
		return a, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan driver adjustments: %w", err)
	}
	return &models.DriverEarningsAdjResult{Items: items, TotalCount: total, Page: f.Page, PageSize: f.PageSize}, nil
}

func (s *EarningsAdjStore) GetDriverByID(ctx context.Context, id int) (*models.DriverEarningsAdj, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	var a models.DriverEarningsAdj
	err = s.pool.QueryRow(ctx,
		`SELECT d.id, d.company_id, d.employee_id,
		COALESCE(e.name, '') as employee_name,
		d.adj_date, d.description, d.adj_type, d.amount::text, d.reference, d.created_at, d.updated_at
		FROM driver_earnings_adjustments d
		LEFT JOIN employees e ON e.id = d.employee_id
		WHERE d.id = $1 AND d.company_id = $2 AND d.deleted_at IS NULL`,
		id, companyID,
	).Scan(&a.ID, &a.CompanyID, &a.EmployeeID, &a.EmployeeName,
		&a.AdjDate, &a.Description, &a.AdjType, &a.Amount, &a.Reference,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get driver adjustment %d: %w", id, err)
	}
	return &a, nil
}

func (s *EarningsAdjStore) CreateDriver(ctx context.Context, a *models.DriverEarningsAdj) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	a.CompanyID = companyID
	return s.pool.QueryRow(ctx,
		`INSERT INTO driver_earnings_adjustments (company_id, employee_id, adj_date, description, adj_type, amount, reference)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at, updated_at`,
		a.CompanyID, a.EmployeeID, a.AdjDate, a.Description, a.AdjType, a.Amount, a.Reference,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (s *EarningsAdjStore) UpdateDriver(ctx context.Context, a *models.DriverEarningsAdj) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE driver_earnings_adjustments
		 SET employee_id=$1, adj_date=$2, description=$3, adj_type=$4, amount=$5, reference=$6
		 WHERE id=$7 AND company_id=$8 AND deleted_at IS NULL`,
		a.EmployeeID, a.AdjDate, a.Description, a.AdjType, a.Amount, a.Reference,
		a.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update driver adjustment %d: %w", a.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("driver adjustment %d not found", a.ID)
	}
	return nil
}

func (s *EarningsAdjStore) DeleteDriver(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		"UPDATE driver_earnings_adjustments SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL",
		id, companyID,
	)
	if err != nil {
		return fmt.Errorf("delete driver adjustment %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("driver adjustment %d not found", id)
	}
	return nil
}

// --- Truck earnings (parallel structure) ---

func (s *EarningsAdjStore) ListTruck(ctx context.Context, f models.EarningsAdjFilter) (*models.TruckEarningsAdjResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	qb := newQueryBuilder()
	qb.Add("t.company_id = ?", companyID)
	qb.AddRaw("t.deleted_at IS NULL")
	if f.EntityID > 0 {
		qb.Add("t.truck_id = ?", f.EntityID)
	}
	if f.DateFrom != "" {
		qb.Add("t.adj_date >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		qb.Add("t.adj_date <= ?", f.DateTo)
	}

	var total int
	if err := s.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM truck_earnings_adjustments t "+qb.Where(), qb.Args()...,
	).Scan(&total); err != nil {
		return nil, fmt.Errorf("count truck adjustments: %w", err)
	}

	query := fmt.Sprintf(`SELECT t.id, t.company_id, t.truck_id,
		COALESCE(tr.truck_number, '') as truck_number,
		t.adj_date, t.description, t.adj_type, t.amount::text, t.reference, t.created_at, t.updated_at
		FROM truck_earnings_adjustments t
		LEFT JOIN trucks tr ON tr.id = t.truck_id
		%s ORDER BY t.adj_date DESC, t.id DESC %s`,
		qb.Where(), qb.Paginate(f.PageSize, f.Page))
	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list truck adjustments: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.TruckEarningsAdj, error) {
		var a models.TruckEarningsAdj
		err := row.Scan(&a.ID, &a.CompanyID, &a.TruckID, &a.TruckNumber,
			&a.AdjDate, &a.Description, &a.AdjType, &a.Amount, &a.Reference,
			&a.CreatedAt, &a.UpdatedAt)
		return a, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan truck adjustments: %w", err)
	}
	return &models.TruckEarningsAdjResult{Items: items, TotalCount: total, Page: f.Page, PageSize: f.PageSize}, nil
}

func (s *EarningsAdjStore) GetTruckByID(ctx context.Context, id int) (*models.TruckEarningsAdj, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	var a models.TruckEarningsAdj
	err = s.pool.QueryRow(ctx,
		`SELECT t.id, t.company_id, t.truck_id,
		COALESCE(tr.truck_number, '') as truck_number,
		t.adj_date, t.description, t.adj_type, t.amount::text, t.reference, t.created_at, t.updated_at
		FROM truck_earnings_adjustments t
		LEFT JOIN trucks tr ON tr.id = t.truck_id
		WHERE t.id = $1 AND t.company_id = $2 AND t.deleted_at IS NULL`,
		id, companyID,
	).Scan(&a.ID, &a.CompanyID, &a.TruckID, &a.TruckNumber,
		&a.AdjDate, &a.Description, &a.AdjType, &a.Amount, &a.Reference,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get truck adjustment %d: %w", id, err)
	}
	return &a, nil
}

func (s *EarningsAdjStore) CreateTruck(ctx context.Context, a *models.TruckEarningsAdj) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	a.CompanyID = companyID
	return s.pool.QueryRow(ctx,
		`INSERT INTO truck_earnings_adjustments (company_id, truck_id, adj_date, description, adj_type, amount, reference)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at, updated_at`,
		a.CompanyID, a.TruckID, a.AdjDate, a.Description, a.AdjType, a.Amount, a.Reference,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

func (s *EarningsAdjStore) UpdateTruck(ctx context.Context, a *models.TruckEarningsAdj) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		`UPDATE truck_earnings_adjustments
		 SET truck_id=$1, adj_date=$2, description=$3, adj_type=$4, amount=$5, reference=$6
		 WHERE id=$7 AND company_id=$8 AND deleted_at IS NULL`,
		a.TruckID, a.AdjDate, a.Description, a.AdjType, a.Amount, a.Reference,
		a.ID, companyID,
	)
	if err != nil {
		return fmt.Errorf("update truck adjustment %d: %w", a.ID, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("truck adjustment %d not found", a.ID)
	}
	return nil
}

func (s *EarningsAdjStore) DeleteTruck(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx,
		"UPDATE truck_earnings_adjustments SET deleted_at = NOW() WHERE id = $1 AND company_id = $2 AND deleted_at IS NULL",
		id, companyID,
	)
	if err != nil {
		return fmt.Errorf("delete truck adjustment %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("truck adjustment %d not found", id)
	}
	return nil
}

// FormatAdjDate parses and formats time for use in form default
func FormatAdjDate(t time.Time) string {
	return t.Format("2006-01-02")
}
```

**Step 5: Write handler** (`internal/handler/earnings_adjustment_handler.go`)

The handler follows the customer handler pattern but serves two resource types on separate URL paths. Key routes:
- `GET /accounting/driver-adjustments` — list all (or filter by employee)
- `GET /accounting/driver-adjustments/new` — new form
- `POST /accounting/driver-adjustments` — create
- `GET /accounting/driver-adjustments/{id}/edit` — edit form
- `PUT /accounting/driver-adjustments/{id}` — update
- `DELETE /accounting/driver-adjustments/{id}` — delete
- Same pattern for `/accounting/truck-adjustments`

The form must have a dropdown for Employee (or Truck). The handler needs the employee store for listing employees for the dropdown. Keep it minimal: pass `employeeStore` with only a `ListAll(ctx) ([]models.Employee, error)` interface.

For brevity, the handler and form follow the same pattern as VendorHandler. Key difference in `bindDriverAdjForm`:
```go
func bindDriverAdjForm(r *http.Request) *models.DriverEarningsAdj {
    a := &models.DriverEarningsAdj{
        Description: formStringRequired(r, "description"),
        AdjType:     r.FormValue("adj_type"),
        Amount:      r.FormValue("amount"),
        Reference:   formString(r, "reference"),
    }
    if eid := r.FormValue("employee_id"); eid != "" {
        if id, err := strconv.Atoi(eid); err == nil {
            a.EmployeeID = id
        }
    }
    if d := r.FormValue("adj_date"); d != "" {
        if t, err := time.Parse("2006-01-02", d); err == nil {
            a.AdjDate = t
        }
    } else {
        a.AdjDate = time.Now()
    }
    if a.AdjType != "Add" && a.AdjType != "Ded" {
        a.AdjType = "Add"
    }
    return a
}
```

**Step 6: Write templ components** (`internal/handler/components/earnings/`)

Three files per adjustment type, or use a unified package with shared layout. The table shows: Employee/Truck, Date, Description, Type (Add/Ded), Amount, Reference, Actions. Use `badge-active` for Add and `badge-inactive` for Ded styling to distinguish visually.

**Step 7: Wire in main.go and add to nav**

Add to nav under Accounting dropdown:
```html
<a href="/accounting/driver-adjustments" class="dropdown-item">Driver Adjustments</a>
<a href="/accounting/truck-adjustments" class="dropdown-item">Truck Adjustments</a>
```

**Step 8: Build, test, commit**

```bash
make generate && go build ./...
make run
# Test both adjustment types: create Add and Ded records, verify correct sign display

git add internal/database/migrations/020_earnings_adjustments.sql \
        internal/models/earnings_adjustment.go \
        internal/store/earnings_adjustment_store.go \
        internal/handler/earnings_adjustment_handler.go \
        internal/handler/components/earnings/ \
        internal/handler/components/nav.templ \
        internal/handler/components/nav_templ.go \
        cmd/server/main.go
git commit -m "feat: add driver and truck earnings adjustments (A75/A76)"
```

---

## Task 3: "Inbound" Vehicle Status

**Context:** Vehicles arriving via EDI exist in an "Inbound" state before they are physically grounded (ready for pickup). The lifecycle needs to be: **Inbound → Waiting → Scheduled → Loaded → Delivered → Confirmed**. No schema change needed — status is a `VARCHAR(10)`. Need to: (1) add Inbound as a valid status throughout the codebase, (2) add a "Ground" action that transitions Inbound → Waiting, (3) ensure Inbound vehicles appear in relevant views.

**Files:**
- Modify: `internal/handler/vehicle_handler.go` (add ground action route)
- Modify: `internal/handler/components/orders/vehicle_table.templ` (show Ground button for Inbound)
- Modify: `internal/handler/components/orders/vehicle_form.templ` (add Inbound to status options if manually creating)
- Modify: `internal/store/vehicle_store.go` (check if status filtering needs update)
- Modify: `internal/store/order_store.go` (check if DashboardCounts/active filters need update)

**Step 1: Add Ground route to vehicle handler**

In `internal/handler/vehicle_handler.go`, in the `Register` method, add:
```go
mux.HandleFunc("POST /dispatch/vehicles/{id}/ground", h.ground)
```

Add the handler method:
```go
func (h *VehicleHandler) ground(w http.ResponseWriter, r *http.Request) {
    h.transitionStatus(w, r, "Waiting")
}
```

**Step 2: Update vehicle_table.templ to show Ground button**

In the vehicle status action buttons section in `internal/handler/components/orders/vehicle_table.templ`, alongside the existing Schedule/Load/Deliver buttons, add a Ground button that appears when status is "Inbound":

```go
if v.Status == "Inbound" {
    <button
        class="btn btn-sm btn-primary"
        hx-post={ fmt.Sprintf("/dispatch/vehicles/%d/ground", v.ID) }
        hx-confirm="Ground this vehicle (move to Waiting)?"
        hx-target="closest tr"
        hx-swap="outerHTML"
    >Ground</button>
}
```

**Step 3: Add Inbound to vehicle form status dropdown**

In `internal/handler/components/orders/vehicle_form.templ`, find the status select and add Inbound as an option before Waiting:
```html
<option value="Inbound" selected?={ vehicle.Status == "Inbound" }>Inbound</option>
```

**Step 4: Review store queries for Inbound**

Check `vehicle_store.go` — if any queries filter `status != 'Waiting'` or similar, ensure Inbound vehicles are included in "active" counts but not in "available to schedule" lists. Specifically:
- `ListAvailableForTrip` should only return `Waiting` status (not Inbound) — verify this is already the case.
- Order `waiting_count` field should also count Inbound separately or together — document the decision (suggest: count Inbound in `waiting_count` for now, as they are both "not yet scheduled").

**Step 5: Update order vehicle creation default**

In `vehicle_handler.go` line 89, the default status is `"Waiting"`. This stays — manual vehicle creation should still default to Waiting. Only EDI-sourced vehicles would start as Inbound.

**Step 6: Build and test**

```bash
make generate && go build ./...
make run
# Create a vehicle manually, then change its status to Inbound via edit form
# Verify the Ground button appears and transitions to Waiting
```

**Step 7: Commit**

```bash
git add internal/handler/vehicle_handler.go \
        internal/handler/components/orders/vehicle_table.templ \
        internal/handler/components/orders/vehicle_table_templ.go \
        internal/handler/components/orders/vehicle_form.templ \
        internal/handler/components/orders/vehicle_form_templ.go
git commit -m "feat: add Inbound vehicle status and Ground action"
```

---

## Task 4: Waiting Grid (Dispatch Board)

**Context:** A dedicated view showing all vehicles in "Waiting" status across all active orders. Used by dispatchers to quickly see what's available to schedule. Groups/filters by state of origin. This is a new report-style page in the Dispatch section.

**Files:**
- Modify: `internal/store/vehicle_store.go` (add `WaitingGrid` query)
- Create: `internal/handler/components/orders/waiting_grid.templ`
- Modify: `internal/handler/order_handler.go` (add waiting grid route) OR create a new handler
- Modify: `internal/handler/components/nav.templ` (add to Dispatch dropdown)
- Modify: `cmd/server/main.go` (if new handler)

**Step 1: Add WaitingGrid store method to vehicle_store.go**

```go
type WaitingVehicleRow struct {
    VehicleID    int
    VIN          string
    Year         *string
    Make         *string
    Model        *string
    OrderID      int
    OrderNumber  string
    PickupName   string
    PickupCity   *string
    PickupState  *string
    DropName     string
    DropCity     *string
    DropState    *string
    Amount       *string
    OrderDate    *time.Time
}

func (s *VehicleStore) WaitingGrid(ctx context.Context, state string) ([]WaitingVehicleRow, error) {
    companyID, err := auth.GetCompanyID(ctx)
    if err != nil {
        return nil, err
    }
    qb := newQueryBuilder()
    qb.Add("v.company_id = ?", companyID)
    qb.AddRaw("v.status = 'Waiting'")
    qb.AddRaw("v.active = true")
    qb.AddRaw("v.deleted_at IS NULL")
    if state != "" {
        qb.Add("pu.state = ?", state)
    }

    query := fmt.Sprintf(`
        SELECT v.id, v.vin,
            v.year, v.make, v.model,
            o.id, o.order_number,
            COALESCE(pu.name,'') as pickup_name, pu.city, pu.state,
            COALESCE(do.name,'') as drop_name, do.city, do.state,
            v.amount, o.order_date
        FROM order_vehicles v
        JOIN orders o ON o.id = v.order_id
        LEFT JOIN customers pu ON pu.id = o.load_customer_id
        LEFT JOIN customers do ON do.id = o.drop_customer_id
        %s
        ORDER BY pu.state, pu.city, v.id`, qb.Where())

    rows, err := s.pool.Query(ctx, query, qb.Args()...)
    if err != nil {
        return nil, fmt.Errorf("waiting grid: %w", err)
    }
    defer rows.Close()

    var result []WaitingVehicleRow
    for rows.Next() {
        var r WaitingVehicleRow
        if err := rows.Scan(&r.VehicleID, &r.VIN, &r.Year, &r.Make, &r.Model,
            &r.OrderID, &r.OrderNumber,
            &r.PickupName, &r.PickupCity, &r.PickupState,
            &r.DropName, &r.DropCity, &r.DropState,
            &r.Amount, &r.OrderDate); err != nil {
            return nil, fmt.Errorf("scan waiting vehicle: %w", err)
        }
        result = append(result, r)
    }
    return result, rows.Err()
}
```

**Step 2: Add route to order_handler.go**

Add to `OrderHandler.Register`:
```go
mux.HandleFunc("GET /dispatch/waiting", h.waitingGrid)
```

Add the handler method (the `orderHandler` will need access to `vehicleStore`; check if it already has one — if not, add a `waitingStore` interface):
```go
func (h *OrderHandler) waitingGrid(w http.ResponseWriter, r *http.Request) {
    stateFilter := r.URL.Query().Get("state")
    rows, err := h.vehicleStore.WaitingGrid(r.Context(), stateFilter)
    if err != nil {
        serverError(w, err)
        return
    }
    if isHTMX(r) {
        h.deps.renderTempl(w, r, orders.WaitingGridTable(rows, stateFilter))
        return
    }
    pg := h.deps.pageContext(w, r)
    h.deps.renderTempl(w, r, orders.WaitingGridPage(pg, rows, stateFilter))
}
```

**Step 3: Create waiting_grid.templ**

`internal/handler/components/orders/waiting_grid.templ`:
```go
package orders

import (
    "fmt"
    "github.com/brady1408/atlinks/internal/handler/components"
    "github.com/brady1408/atlinks/internal/store"
)

templ WaitingGridPage(pg components.PageContext, rows []store.WaitingVehicleRow, stateFilter string) {
    @components.Layout(pg, "Waiting Grid - ATLinks") {
        <div class="page-header">
            <h1>Waiting Grid</h1>
            <span class="text-muted">{ fmt.Sprintf("%d vehicles waiting", len(rows)) }</span>
        </div>
        <div class="filter-bar">
            <div class="form-group">
                <label for="state">Origin State</label>
                <input
                    type="text" id="state" name="state" class="form-control" maxlength="2"
                    placeholder="e.g. MI" value={ stateFilter }
                    hx-get="/dispatch/waiting"
                    hx-trigger="keyup changed delay:400ms"
                    hx-target="#waiting-table"
                    hx-include="[name=state]"
                    style="text-transform:uppercase; width:80px"
                />
            </div>
        </div>
        <div id="waiting-table">
            @WaitingGridTable(rows, stateFilter)
        </div>
    }
}

templ WaitingGridTable(rows []store.WaitingVehicleRow, stateFilter string) {
    <div class="table-container">
        <table>
            <thead>
                <tr>
                    <th>VIN</th>
                    <th>Year/Make/Model</th>
                    <th>Order</th>
                    <th>Pickup</th>
                    <th>State</th>
                    <th>Dropoff</th>
                    <th class="text-right">Amount</th>
                    <th class="text-right">Actions</th>
                </tr>
            </thead>
            <tbody>
                if len(rows) > 0 {
                    for _, r := range rows {
                        <tr>
                            <td class="font-mono text-sm">{ r.VIN }</td>
                            <td>{ components.Deref(r.Year) } { components.Deref(r.Make) } { components.Deref(r.Model) }</td>
                            <td><a href={ templ.SafeURL(fmt.Sprintf("/dispatch/orders/%d", r.OrderID)) }>{ r.OrderNumber }</a></td>
                            <td>{ r.PickupName } { components.Deref(r.PickupCity) }</td>
                            <td>{ components.Deref(r.PickupState) }</td>
                            <td>{ r.DropName } { components.Deref(r.DropCity) }</td>
                            <td class="text-right">{ components.Deref(r.Amount) }</td>
                            <td class="text-right">
                                <a href={ templ.SafeURL(fmt.Sprintf("/dispatch/orders/%d", r.OrderID)) } class="btn btn-sm">View Order</a>
                            </td>
                        </tr>
                    }
                } else {
                    <tr><td colspan="8" class="text-center text-muted">No vehicles waiting</td></tr>
                }
            </tbody>
        </table>
    </div>
}
```

**Step 4: Add to nav**

In `nav.templ` Dispatch dropdown, add:
```html
<a href="/dispatch/waiting" class="dropdown-item">Waiting Grid</a>
```

**Step 5: Build, test, commit**

```bash
make generate && go build ./...
make run
# Navigate to /dispatch/waiting, verify vehicles in Waiting status appear
# Test state filter

git add internal/store/vehicle_store.go \
        internal/handler/order_handler.go \
        internal/handler/components/orders/waiting_grid.templ \
        internal/handler/components/orders/waiting_grid_templ.go \
        internal/handler/components/nav.templ \
        internal/handler/components/nav_templ.go
git commit -m "feat: add waiting grid dispatch board"
```

---

## Task 5: Truck Expiration Warnings

**Context:** The trucks table already has `truck_license_exp`, `trailer_license_exp`, `truck_safety_inspection`, `trailer_safety_inspection`, `insurance_exp_date`. Need to: (1) add an `ExpiringTrucks` query to the truck store, (2) surface warnings on the dashboard, (3) show expiration badges on the trucks list.

**Files:**
- Modify: `internal/store/truck_store.go` (add `ExpiringTrucks` query)
- Modify: `internal/handler/components/pages/dashboard.templ` (add expiration widget)
- Modify: `internal/handler/dashboard_handler.go` (pass expiration data)
- Modify: `internal/handler/components/trucks/table.templ` (add expiration badges)

**Step 1: Add ExpiringTrucks to truck_store.go**

```go
type ExpiringTruck struct {
    TruckID        int
    TruckNumber    string
    ExpType        string // "Truck License", "Trailer License", "Truck Inspection", "Trailer Inspection", "Insurance"
    ExpDate        time.Time
    DaysUntilExp   int    // negative = already expired
}

func (s *TruckStore) ExpiringWithin(ctx context.Context, days int) ([]ExpiringTruck, error) {
    companyID, err := auth.GetCompanyID(ctx)
    if err != nil {
        return nil, err
    }
    cutoff := time.Now().AddDate(0, 0, days)
    query := `
        SELECT id, truck_number, expiry_type, exp_date,
               EXTRACT(DAY FROM exp_date - NOW())::int as days_until
        FROM (
            SELECT id, COALESCE(truck_number,'') as truck_number, 'Truck License' as expiry_type, truck_license_exp as exp_date
            FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND truck_license_exp IS NOT NULL AND truck_license_exp <= $2
            UNION ALL
            SELECT id, COALESCE(truck_number,''), 'Trailer License', trailer_license_exp
            FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND trailer_license_exp IS NOT NULL AND trailer_license_exp <= $2
            UNION ALL
            SELECT id, COALESCE(truck_number,''), 'Truck Inspection', truck_safety_inspection
            FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND truck_safety_inspection IS NOT NULL AND truck_safety_inspection <= $2
            UNION ALL
            SELECT id, COALESCE(truck_number,''), 'Trailer Inspection', trailer_safety_inspection
            FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND trailer_safety_inspection IS NOT NULL AND trailer_safety_inspection <= $2
            UNION ALL
            SELECT id, COALESCE(truck_number,''), 'Insurance', insurance_exp_date
            FROM trucks WHERE company_id=$1 AND deleted_at IS NULL AND insurance_exp_date IS NOT NULL AND insurance_exp_date <= $2
        ) t
        ORDER BY exp_date ASC`

    rows, err := s.pool.Query(ctx, query, companyID, cutoff)
    if err != nil {
        return nil, fmt.Errorf("expiring trucks: %w", err)
    }
    defer rows.Close()

    var result []ExpiringTruck
    for rows.Next() {
        var e ExpiringTruck
        if err := rows.Scan(&e.TruckID, &e.TruckNumber, &e.ExpType, &e.ExpDate, &e.DaysUntilExp); err != nil {
            return nil, err
        }
        result = append(result, e)
    }
    return result, rows.Err()
}
```

**Step 2: Update dashboard handler**

Add `truckStore` interface to `DashboardHandler` with `ExpiringWithin` method. Fetch expiring trucks (within 60 days) and pass to template.

**Step 3: Update dashboard template**

Add a new card in the `dashboard-grid` div:
```go
if len(expiringTrucks) > 0 {
    <div class="card">
        <div class="card-header" style="color:var(--danger)">⚠ Truck Expirations (within 60 days)</div>
        <div class="card-body">
            <table>
                <tbody>
                for _, t := range expiringTrucks {
                    <tr>
                        <td><a href={ templ.SafeURL(fmt.Sprintf("/global/trucks/%d/edit", t.TruckID)) }>{ t.TruckNumber }</a></td>
                        <td>{ t.ExpType }</td>
                        <td>
                            if t.DaysUntilExp < 0 {
                                <span class="badge badge-inactive">EXPIRED { fmt.Sprintf("%d days ago", -t.DaysUntilExp) }</span>
                            } else {
                                <span class="badge badge-warning">{ t.ExpDate.Format("Jan 2, 2006") } ({ fmt.Sprintf("%d days", t.DaysUntilExp) })</span>
                            }
                        </td>
                    </tr>
                }
                </tbody>
            </table>
        </div>
    </div>
}
```

Note: Add `badge-warning` CSS if not present (yellow/amber variant).

**Step 4: Build, test, commit**

```bash
make generate && go build ./...
# Set a truck's insurance_exp_date to 30 days from now, verify dashboard shows it
git add internal/store/truck_store.go \
        internal/handler/dashboard_handler.go \
        internal/handler/components/pages/dashboard.templ \
        internal/handler/components/pages/dashboard_templ.go
git commit -m "feat: add truck expiration warnings to dashboard"
```

---

## Task 6: Documents/Attachments Browse

**Context:** The upload system (handler + store) and the `attachments` table are fully built. The `AttachmentList` component already renders per-entity. The missing piece is a browse page that shows all attachments for an order or trip, with upload capability.

**Files:**
- Modify: `internal/handler/upload_handler.go` (add `POST /dispatch/orders/{id}/attachments` and `/dispatch/trips/{id}/attachments`)
- Modify: `internal/handler/components/orders/show.templ` (add attachments section)
- Modify: `internal/handler/components/trips/show.templ` (add attachments section)

**Step 1: Add upload routes for orders and trips**

In `upload_handler.go`, add to `Register`:
```go
mux.HandleFunc("POST /dispatch/orders/{id}/attachments", h.uploadOrder)
mux.HandleFunc("POST /dispatch/trips/{id}/attachments", h.uploadTrip)
```

Add handlers:
```go
func (h *UploadHandler) uploadOrder(w http.ResponseWriter, r *http.Request) {
    id, err := parseID(r)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }
    h.handleGenericUpload(w, r, "order", id)
}

func (h *UploadHandler) uploadTrip(w http.ResponseWriter, r *http.Request) {
    id, err := parseID(r)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }
    h.handleGenericUpload(w, r, "trip", id)
}
```

Add `handleGenericUpload` which accepts any file type (PDF, images, etc.) unlike the image-only `handleImageUpload`. Allow: PDF, images, and common document types. Max 25MB.

```go
var allowedDocTypes = map[string]bool{
    "image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true,
    "application/pdf": true,
    "text/plain": true, "text/csv": true,
    "application/vnd.ms-excel": true,
    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
}

func (h *UploadHandler) handleGenericUpload(w http.ResponseWriter, r *http.Request, category string, entityID int) {
    // Same as handleImageUpload but use allowedDocTypes instead of allowedImageTypes
    // ...
}
```

**Step 2: Add attachments section to order show page**

In `internal/handler/components/orders/show.templ`, at the bottom before closing `@components.Layout`, add an attachments section. The order show page needs to load the attachment list — pass `[]models.Attachment` as an additional parameter to `ShowPage`.

In `order_handler.go` show method, after loading the order, also load:
```go
atts, _ := h.attachmentStore.ListByEntity(r.Context(), "order", id)
```

Then pass to template. In the template:
```go
<div class="card">
    <div class="card-header">Attachments</div>
    <div class="card-body">
        <form hx-post={ fmt.Sprintf("/dispatch/orders/%d/attachments", order.ID) }
              hx-target="#order-attachment-list" hx-swap="outerHTML"
              hx-encoding="multipart/form-data">
            @components.CSRFField(pg.CSRFToken)
            <input type="file" name="file" class="form-control" accept=".pdf,.jpg,.jpeg,.png,.csv"/>
            <button type="submit" class="btn btn-sm">Upload</button>
        </form>
        <div id="order-attachment-list">
            @attachments.AttachmentList(atts, "order", order.ID)
        </div>
    </div>
</div>
```

**Step 3: Same for trip show page**

Apply identical pattern to `internal/handler/components/trips/show.templ` and `trip_handler.go`.

**Step 4: Build, test, commit**

```bash
make generate && go build ./...
make run
# Upload a PDF to an order, verify it appears in the list and can be downloaded

git add internal/handler/upload_handler.go \
        internal/handler/order_handler.go \
        internal/handler/trip_handler.go \
        internal/handler/components/orders/show.templ \
        internal/handler/components/orders/show_templ.go \
        internal/handler/components/trips/show.templ \
        internal/handler/components/trips/show_templ.go
git commit -m "feat: add document/attachment browse and upload for orders and trips"
```

---

## Task 7: Batch Invoice Recalculation UI

**Context:** `InvoiceService.RecalcTotals(ctx, invoiceID)` already exists. Need a simple UI page that runs recalc across invoices matching a date range or customer. Add as a utility page under Accounting.

**Files:**
- Modify: `internal/handler/invoice_handler.go` (add recalc batch routes)
- Create: `internal/handler/components/invoices/recalc.templ`
- Modify: `internal/handler/components/nav.templ` (optional: add to Accounting)

**Step 1: Add batch recalc to invoice store**

In `invoice_store.go`, add method to find invoice IDs by date range:
```go
func (s *InvoiceStore) IDsByDateRange(ctx context.Context, dateFrom, dateTo string) ([]int, error) {
    companyID, err := auth.GetCompanyID(ctx)
    if err != nil {
        return nil, err
    }
    qb := newQueryBuilder()
    qb.Add("company_id = ?", companyID)
    qb.AddRaw("deleted_at IS NULL")
    qb.AddRaw("status != 'Void'")
    if dateFrom != "" {
        qb.Add("invoice_date >= ?", dateFrom)
    }
    if dateTo != "" {
        qb.Add("invoice_date <= ?", dateTo)
    }
    rows, err := s.pool.Query(ctx, "SELECT id FROM invoices "+qb.Where()+" ORDER BY id", qb.Args()...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var ids []int
    for rows.Next() {
        var id int
        if err := rows.Scan(&id); err != nil {
            return nil, err
        }
        ids = append(ids, id)
    }
    return ids, rows.Err()
}
```

**Step 2: Add routes to invoice_handler.go**

```go
mux.HandleFunc("GET /accounting/invoices/recalc", h.recalcForm)
mux.HandleFunc("POST /accounting/invoices/recalc", h.recalcRun)
```

Handler methods:
```go
func (h *InvoiceHandler) recalcForm(w http.ResponseWriter, r *http.Request) {
    pg := h.deps.pageContext(w, r)
    h.deps.renderTempl(w, r, invoices.RecalcPage(pg, 0, ""))
}

func (h *InvoiceHandler) recalcRun(w http.ResponseWriter, r *http.Request) {
    dateFrom := r.FormValue("date_from")
    dateTo := r.FormValue("date_to")
    ids, err := h.invoiceStore.IDsByDateRange(r.Context(), dateFrom, dateTo)
    if err != nil {
        serverError(w, err)
        return
    }
    count := 0
    for _, id := range ids {
        if err := h.invoiceSvc.RecalcTotals(r.Context(), id); err != nil {
            log.Printf("recalc invoice %d: %v", id, err)
        } else {
            count++
        }
    }
    pg := h.deps.pageContext(w, r)
    h.deps.renderTempl(w, r, invoices.RecalcPage(pg, count, fmt.Sprintf("Recalculated %d invoices", count)))
}
```

**Step 3: Create recalc.templ**

```go
package invoices

import "github.com/brady1408/atlinks/internal/handler/components"

templ RecalcPage(pg components.PageContext, count int, msg string) {
    @components.Layout(pg, "Recalculate Invoices - ATLinks") {
        <div class="page-header">
            <h1>Batch Invoice Recalculation</h1>
            <a href="/accounting/invoices" class="btn">Back to Invoices</a>
        </div>
        if msg != "" {
            <div class="alert alert-success">{ msg }</div>
        }
        <div class="card">
            <div class="card-body">
                <p class="text-muted">Recalculates subtotals, taxes, and balances for all non-void invoices in the selected date range based on their current detail lines.</p>
                <form method="POST" action="/accounting/invoices/recalc">
                    @components.CSRFField(pg.CSRFToken)
                    <div class="form-row">
                        <div class="form-group">
                            <label for="date_from">Invoice Date From</label>
                            <input type="date" id="date_from" name="date_from" class="form-control"/>
                        </div>
                        <div class="form-group">
                            <label for="date_to">Invoice Date To</label>
                            <input type="date" id="date_to" name="date_to" class="form-control"/>
                        </div>
                    </div>
                    <div class="form-actions">
                        <button type="submit" class="btn btn-primary"
                            onclick="return confirm('Recalculate all matching invoices?')">
                            Run Recalculation
                        </button>
                    </div>
                </form>
            </div>
        </div>
    }
}
```

**Step 4: Build, test, commit**

```bash
make generate && go build ./...
make run
# Navigate to /accounting/invoices/recalc, run with a date range, verify success message

git add internal/store/invoice_store.go \
        internal/handler/invoice_handler.go \
        internal/handler/components/invoices/recalc.templ \
        internal/handler/components/invoices/recalc_templ.go
git commit -m "feat: add batch invoice recalculation UI"
```

---

## Task 8: Customer Statements

**Context:** A standard AR statement document per customer. Lists all open invoices, payments applied, current balance, and aging buckets. Both an HTML screen view and a printable version. Should be accessible from the Accounting → Reports section.

**Files:**
- Modify: `internal/store/invoice_store.go` (add `StatementData` query)
- Modify: `internal/handler/report_handler.go` (add statement route)
- Create: `internal/handler/components/reports/statement.templ`
- Modify: `internal/handler/components/reports/index.templ` (add link)

**Step 1: Add StatementData query to invoice_store.go**

```go
type StatementRow struct {
    InvoiceNumber string
    InvoiceDate   *time.Time
    DueDate       *time.Time
    OrderNumber   *string
    TotalAmount   *string
    AmountPaid    *string
    Balance       *string
    Status        *string
    DaysOld       int
}

type StatementData struct {
    CustomerID     int
    CustomerNumber string
    CustomerName   string
    BillToAddress  *string
    BillToCity     *string
    BillToState    *string
    BillToZip      *string
    StatementDate  time.Time
    Rows           []StatementRow
    TotalBalance   string
    Current        string // 0-30 days
    Days31         string
    Days61         string
    Days90         string // 90+
}

func (s *InvoiceStore) GetStatement(ctx context.Context, customerID int) (*StatementData, error) {
    companyID, err := auth.GetCompanyID(ctx)
    if err != nil {
        return nil, err
    }
    // Get customer info
    var stmt StatementData
    stmt.CustomerID = customerID
    stmt.StatementDate = time.Now()

    err = s.pool.QueryRow(ctx,
        `SELECT COALESCE(number,''), name,
            address, city, state, zip
        FROM customers WHERE id=$1 AND company_id=$2 AND deleted_at IS NULL`,
        customerID, companyID,
    ).Scan(&stmt.CustomerNumber, &stmt.CustomerName,
        &stmt.BillToAddress, &stmt.BillToCity, &stmt.BillToState, &stmt.BillToZip)
    if err != nil {
        return nil, fmt.Errorf("get statement customer: %w", err)
    }

    rows, err := s.pool.Query(ctx,
        `SELECT invoice_number, invoice_date, due_date, order_number,
            total_amount::text, amount_paid::text, balance::text, status,
            EXTRACT(DAY FROM NOW() - invoice_date)::int as days_old
        FROM invoices
        WHERE customer_id=$1 AND company_id=$2 AND deleted_at IS NULL
            AND status NOT IN ('Void') AND balance::numeric > 0
        ORDER BY invoice_date`,
        customerID, companyID,
    )
    if err != nil {
        return nil, fmt.Errorf("statement rows: %w", err)
    }
    defer rows.Close()
    for rows.Next() {
        var r StatementRow
        if err := rows.Scan(&r.InvoiceNumber, &r.InvoiceDate, &r.DueDate, &r.OrderNumber,
            &r.TotalAmount, &r.AmountPaid, &r.Balance, &r.Status, &r.DaysOld); err != nil {
            return nil, err
        }
        stmt.Rows = append(stmt.Rows, r)
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }

    // Aging totals
    err = s.pool.QueryRow(ctx,
        `SELECT
            COALESCE(SUM(balance::numeric) FILTER (WHERE EXTRACT(DAY FROM NOW()-invoice_date) <= 30), 0)::text,
            COALESCE(SUM(balance::numeric) FILTER (WHERE EXTRACT(DAY FROM NOW()-invoice_date) BETWEEN 31 AND 60), 0)::text,
            COALESCE(SUM(balance::numeric) FILTER (WHERE EXTRACT(DAY FROM NOW()-invoice_date) BETWEEN 61 AND 90), 0)::text,
            COALESCE(SUM(balance::numeric) FILTER (WHERE EXTRACT(DAY FROM NOW()-invoice_date) > 90), 0)::text,
            COALESCE(SUM(balance::numeric), 0)::text
        FROM invoices
        WHERE customer_id=$1 AND company_id=$2 AND deleted_at IS NULL
            AND status NOT IN ('Void') AND balance::numeric > 0`,
        customerID, companyID,
    ).Scan(&stmt.Current, &stmt.Days31, &stmt.Days61, &stmt.Days90, &stmt.TotalBalance)
    if err != nil {
        return nil, fmt.Errorf("statement aging: %w", err)
    }

    return &stmt, nil
}
```

**Step 2: Add to report_handler.go**

Add interface method and route:
```go
mux.HandleFunc("GET /reports/statement", h.statementForm)
mux.HandleFunc("GET /reports/statement/{id}", h.statementShow)
```

`statementForm` shows a customer search/select form. `statementShow` takes a customer_id path param and renders the statement.

**Step 3: Create statement.templ**

Renders: company header, customer address block, statement date, invoice table with aging columns, aging summary footer. Has a Print button. Follows the AR aging report pattern.

**Step 4: Build, test, commit**

```bash
make generate && go build ./...
make run
# Navigate to /reports/statement, select a customer, verify statement renders

git add internal/store/invoice_store.go \
        internal/handler/report_handler.go \
        internal/handler/components/reports/statement.templ \
        internal/handler/components/reports/statement_templ.go \
        internal/handler/components/reports/index.templ \
        internal/handler/components/reports/index_templ.go
git commit -m "feat: add customer AR statement report"
```

---

## Task 9: Invoice/Payment Posting (Period Close)

**Context:** "Posting" means marking invoices and payments as finalized for an accounting period — they become read-only after posting. This requires: (1) a `posted_at` column on invoices and payments, (2) a posting screen with date filter, (3) preventing edits to posted records.

**Files:**
- Create: `internal/database/migrations/021_posting.sql`
- Modify: `internal/models/invoice.go` (add PostedAt field)
- Modify: `internal/models/payment.go` (add PostedAt field)
- Modify: `internal/store/invoice_store.go` (add posting query)
- Modify: `internal/store/payment_store.go` (add posting query)
- Create: `internal/handler/components/invoices/posting.templ`
- Modify: `internal/handler/invoice_handler.go` (add posting routes, guard edits on posted invoices)

**Step 1: Migration**

```sql
-- internal/database/migrations/021_posting.sql
-- +goose Up
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS posted_at TIMESTAMPTZ;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS posted_by VARCHAR(50);
ALTER TABLE payments ADD COLUMN IF NOT EXISTS posted_at TIMESTAMPTZ;
ALTER TABLE payments ADD COLUMN IF NOT EXISTS posted_by VARCHAR(50);

-- +goose Down
ALTER TABLE invoices DROP COLUMN IF EXISTS posted_at;
ALTER TABLE invoices DROP COLUMN IF EXISTS posted_by;
ALTER TABLE payments DROP COLUMN IF EXISTS posted_at;
ALTER TABLE payments DROP COLUMN IF EXISTS posted_by;
```

**Step 2: Update models**

Add `PostedAt *time.Time` and `PostedBy *string` to `Invoice` and `Payment` models.

**Step 3: Add posting store methods**

In invoice_store.go:
```go
func (s *InvoiceStore) PostByDateRange(ctx context.Context, dateFrom, dateTo, username string) (int, error) {
    companyID, err := auth.GetCompanyID(ctx)
    if err != nil {
        return 0, err
    }
    result, err := s.pool.Exec(ctx,
        `UPDATE invoices SET posted_at = NOW(), posted_by = $1
         WHERE company_id = $2 AND deleted_at IS NULL
           AND posted_at IS NULL AND status != 'Void'
           AND invoice_date >= $3 AND invoice_date <= $4`,
        username, companyID, dateFrom, dateTo,
    )
    if err != nil {
        return 0, fmt.Errorf("post invoices: %w", err)
    }
    return int(result.RowsAffected()), nil
}
```

Same pattern for payments.

**Step 4: Add posting routes to invoice_handler.go**

```go
mux.HandleFunc("GET /accounting/posting", h.postingForm)
mux.HandleFunc("POST /accounting/posting", h.postingRun)
```

The handler shows a form with `date_from`/`date_to`, previews the count of unposted invoices and payments in that range, then on POST runs `PostByDateRange` for both.

**Step 5: Guard edits on posted invoices**

In `invoice_handler.go` update method, after loading the invoice:
```go
if inv.PostedAt != nil {
    http.Error(w, "Cannot modify a posted invoice", http.StatusForbidden)
    return
}
```

Same guard in the void handler.

**Step 6: Create posting.templ**

Simple form with date range, preview counts, and a confirm button. Show warning that posting cannot be undone.

**Step 7: Add to nav**

Under Accounting dropdown:
```html
<a href="/accounting/posting" class="dropdown-item">Period Posting</a>
```

**Step 8: Build, test, commit**

```bash
make generate && go build ./...
make run
# Create a test invoice, try posting it, verify edit is blocked afterward

git add internal/database/migrations/021_posting.sql \
        internal/models/invoice.go \
        internal/models/payment.go \
        internal/store/invoice_store.go \
        internal/store/payment_store.go \
        internal/handler/invoice_handler.go \
        internal/handler/components/invoices/posting.templ \
        internal/handler/components/invoices/posting_templ.go \
        internal/handler/components/nav.templ \
        internal/handler/components/nav_templ.go
git commit -m "feat: add invoice/payment period posting with posted-at guard"
```

---

## Final Verification

After all 9 tasks are complete:

**Step 1: Run full build and tests**

```bash
make generate
go build ./...
go test ./...
```
Expected: clean build, all tests pass.

**Step 2: Run against production database (staging check)**

```bash
# Deploy to production
./scripts/deploy.sh
```

**Step 3: Verify each feature on https://atlinks.app**

- [ ] `/global/vendors` — create, edit, delete vendor
- [ ] `/accounting/driver-adjustments` — create Add and Ded adjustment
- [ ] `/accounting/truck-adjustments` — create Add and Ded adjustment
- [ ] Order vehicle edit — Inbound status selectable, Ground button works
- [ ] `/dispatch/waiting` — waiting grid with state filter
- [ ] Dashboard — truck expiration card appears for trucks expiring within 60 days
- [ ] Order show page — file upload panel present, upload works
- [ ] Trip show page — file upload panel present
- [ ] `/accounting/invoices/recalc` — date range recalc runs
- [ ] `/reports/statement` — customer statement renders with aging footer
- [ ] `/accounting/posting` — posting marks invoices + payments, blocks edits

**Step 4: Close feedback items in ATLinks**

Mark feedback items #35-#44 as closed in the ATLinks feedback system.
