# Sortable Table Columns

**Date:** 2026-03-11
**Status:** Draft
**Feedback:** #107

## Context

All browse/list tables in ATLinks sort by a fixed hardcoded order (e.g. orders by `order_number DESC`). Users can't reorder results, making it hard to find recent records or sort by customer/date/status. This is especially painful as data grows — you can't find a brand-new order without paging forward.

## Scope

Add server-side sortable columns to four priority tables: **Orders, Trips, Invoices, Payments**. Sort state lives in URL query params (`sort_by`, `sort_dir`), matching the existing filter pattern. Visual style: subtle arrows (Style A — faint arrows on sortable columns, active column arrow turns blue).

## Design

### 1. Shared Sort Helpers

**File:** `internal/store/sort.go`

```go
// SortConfig holds validated sort parameters.
type SortConfig struct {
    Column    string // validated SQL column expression
    Direction string // "ASC" or "DESC"
}

// ValidateSort checks sort_by against an allowlist and normalizes direction.
// Returns the default if sort_by is empty or not in the allowlist.
func ValidateSort(sortBy, sortDir string, allowed map[string]string, defaultCol, defaultDir string) SortConfig {
    col, ok := allowed[sortBy]
    if !ok {
        return SortConfig{Column: defaultCol, Direction: defaultDir}
    }
    dir := "ASC"
    if strings.EqualFold(sortDir, "desc") {
        dir = "DESC"
    }
    return SortConfig{Column: col, Direction: dir}
}

// OrderByClause returns "ORDER BY <column> <dir> NULLS LAST" for use in queries.
// NULLS LAST prevents nulls from appearing first on ASC sorts.
func (s SortConfig) OrderByClause() string {
    return "ORDER BY " + s.Column + " " + s.Direction + " NULLS LAST"
}

// OrderByWithSecondary adds a tiebreaker sort for stable pagination.
func (s SortConfig) OrderByWithSecondary(secondaryCol, secondaryDir string) string {
    return s.OrderByClause() + ", " + secondaryCol + " " + secondaryDir
}
```

The `allowed` map keys are user-facing param values; values are SQL column expressions. This prevents SQL injection by design — only pre-approved columns can appear in the query.

### 2. Filter Types

Add `SortBy` and `SortDir` string fields to each filter struct.

**`internal/models/order.go`** — `OrderFilter`:
```go
type OrderFilter struct {
    // ... existing fields ...
    SortBy  string
    SortDir string
}
```

Same pattern for `TripFilter`, `InvoiceFilter`, `PaymentFilter`.

### 3. Store Layer

Each store defines its own sort allowlist and uses `ValidateSort` in `List()`.

**`internal/store/order_store.go`** example:
```go
var orderSortColumns = map[string]string{
    "order_number": "order_number",
    "customer":     "bill_customer_name",
    "zone":         "zone",
    "status":       "dispatch_code",
    "create_date":  "create_date",
}

func (s *OrderStore) List(ctx context.Context, f models.OrderFilter) (*models.OrderListResult, error) {
    sort := ValidateSort(f.SortBy, f.SortDir, orderSortColumns, "order_number", "DESC")
    // ... existing query building ...
    // Replace hardcoded ORDER BY with: sort.OrderByClause()
}
```

**Sort allowlists per entity** (column names match actual DB columns — no table aliases since queries use bare table names):

| Entity | Sortable Columns (param → SQL) |
|--------|-------------------------------|
| Orders | `order_number` → `order_number`, `customer` → `bill_customer_name`, `zone` → `zone`, `status` → `dispatch_code`, `create_date` → `create_date` |
| Trips | `load_number` → `load_number`, `truck` → `truck_number`, `driver` → `driver`, `trip_date` → `trip_date`, `status` → `status` |
| Invoices | `invoice_number` → `invoice_number`, `customer` → `customer_name`, `date` → `invoice_date`, `total` → `total_amount`, `status` → `status` |
| Payments | `id` → `id`, `date` → `payment_date`, `customer` → `customer_name`, `amount` → `amount` |

### 4. Handler Layer

Parse `sort_by` and `sort_dir` from query params alongside existing filters. No new helper needed — just `r.URL.Query().Get()`.

```go
filter := models.OrderFilter{
    // ... existing ...
    SortBy:  r.URL.Query().Get("sort_by"),
    SortDir: r.URL.Query().Get("sort_dir"),
}
```

### 5. Templ Components

#### 5a. Shared `SortHeader` Component

**File:** `internal/handler/components/sort_header.templ`

```
templ SortHeader(label, column, currentSort, currentDir, baseURL, targetID string) {
    <th class={ "sortable", templ.KV("sorted", column == currentSort) }
        hx-get={ baseURL }
        hx-include=".filter-bar [name]"
        hx-target={ targetID }
        hx-push-url="true"
        hx-vals={ sortVals(column, currentSort, currentDir) }
        style="cursor: pointer; user-select: none;">
        { label }
        <span class="sort-arrow">{ sortArrow(column, currentSort, currentDir) }</span>
    </th>
}
```

Helper functions in the same file:
- `sortVals(column, currentSort, currentDir)` — returns JSON `{"sort_by":"col","sort_dir":"asc/desc","page":"1"}`. Toggles direction if clicking the already-sorted column. **Resets page to 1** on sort change.
- `sortArrow(column, currentSort, currentDir)` — returns `"▲"` or `"▼"` based on active state. Returns `"▲"` (faint) for non-active columns.

#### 5b. Hidden Sort Inputs in Filter Bar

Add to each list page's filter bar (so `hx-include` picks them up for pagination):
```html
<input type="hidden" name="sort_by" value={ filter.SortBy } />
<input type="hidden" name="sort_dir" value={ filter.SortDir } />
```

#### 5c. Table Signature Changes

All four `Table` components must accept the filter to access sort state. Change signatures:
- `templ Table(result models.OrderListResult)` → `templ Table(result models.OrderListResult, filter models.OrderFilter)`
- Same for Trips, Invoices, Payments

Update all call sites: handlers rendering Table on HTMX requests, and ListPage components that embed Table.

#### 5d. Table Header Updates

Replace hardcoded `<th>` with `@SortHeader(...)` for sortable columns. Non-sortable columns (e.g. "Actions") stay as plain `<th>`.

Example for orders table:
```
@SortHeader("Order #", "order_number", filter.SortBy, filter.SortDir, "/dispatch/orders", "#order-table")
@SortHeader("Bill Customer", "customer", filter.SortBy, filter.SortDir, "/dispatch/orders", "#order-table")
@SortHeader("Zone", "zone", filter.SortBy, filter.SortDir, "/dispatch/orders", "#order-table")
<th class="text-center">Vehicles</th>  <!-- not sortable -->
@SortHeader("Status", "status", filter.SortBy, filter.SortDir, "/dispatch/orders", "#order-table")
@SortHeader("Create Date", "create_date", filter.SortBy, filter.SortDir, "/dispatch/orders", "#order-table")
<th class="text-right">Actions</th>  <!-- not sortable -->
```

### 6. CSS

**Add to `internal/handler/static/css/app.css`:**

```css
th.sortable { cursor: pointer; user-select: none; }
th.sortable:hover { color: var(--primary); }
.sort-arrow { font-size: 10px; margin-left: 4px; opacity: 0.3; }
th.sorted .sort-arrow { opacity: 1; color: var(--primary); }
th.sortable:hover .sort-arrow { opacity: 0.6; }
```

### 7. Data Flow

```
User clicks "Create Date" header
  → HTMX GET /dispatch/orders?search=...&sort_by=create_date&sort_dir=asc
    → Handler parses sort_by + sort_dir into OrderFilter
      → Store.List() calls ValidateSort() against allowlist
        → SQL: ORDER BY create_date ASC NULLS LAST, id DESC LIMIT 25 OFFSET 0
          → Response: table HTML with updated sort arrows
            → hx-push-url updates browser URL
```

Clicking the same column again toggles ASC ↔ DESC. Clicking a different column sorts ASC by default.

## Files to Create/Modify

| File | Action |
|------|--------|
| `internal/store/sort.go` | **Create** — `SortConfig`, `ValidateSort`, `OrderByClause` |
| `internal/handler/components/sort_header.templ` | **Create** — shared `SortHeader` component |
| `internal/handler/static/css/app.css` | **Modify** — add sortable th styles |
| `internal/models/order.go` | **Modify** — add SortBy/SortDir to OrderFilter |
| `internal/models/trip.go` | **Modify** — add SortBy/SortDir to TripFilter |
| `internal/models/invoice.go` | **Modify** — add SortBy/SortDir to InvoiceFilter |
| `internal/models/payment.go` | **Modify** — add SortBy/SortDir to PaymentFilter |
| `internal/store/order_store.go` | **Modify** — allowlist + ValidateSort in List() |
| `internal/store/trip_store.go` | **Modify** — allowlist + ValidateSort in List() |
| `internal/store/invoice_store.go` | **Modify** — allowlist + ValidateSort in List() |
| `internal/store/payment_store.go` | **Modify** — allowlist + ValidateSort in List() |
| `internal/handler/order_handler.go` | **Modify** — parse sort params |
| `internal/handler/trip_handler.go` | **Modify** — parse sort params |
| `internal/handler/invoice_handler.go` | **Modify** — parse sort params |
| `internal/handler/payment_handler.go` | **Modify** — parse sort params |
| `internal/handler/components/orders/list.templ` | **Modify** — hidden inputs + SortHeader |
| `internal/handler/components/orders/table.templ` | **Modify** — SortHeader calls |
| `internal/handler/components/trips/list.templ` | **Modify** — hidden inputs + SortHeader |
| `internal/handler/components/trips/table.templ` | **Modify** — SortHeader calls |
| `internal/handler/components/invoices/list.templ` | **Modify** — hidden inputs + SortHeader |
| `internal/handler/components/invoices/table.templ` | **Modify** — SortHeader calls |
| `internal/handler/components/payments/list.templ` | **Modify** — hidden inputs + SortHeader |
| `internal/handler/components/payments/table.templ` | **Modify** — SortHeader calls |

## Verification

1. `go build ./...` — compiles clean
2. `templ generate` — no errors
3. Manual test: click column headers on orders page, verify sort toggles and URL updates
4. Verify pagination preserves sort state (sort by date, go to page 2, still sorted by date)
5. Verify invalid sort_by param falls back to default (SQL injection prevention)
6. MCP: `list_orders` tool should also accept sort params (future enhancement, not in this pass)
