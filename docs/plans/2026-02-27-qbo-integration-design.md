# QuickBooks Online Integration — Design

**Date:** 2026-02-27
**Status:** Approved

## Overview

Export ATLinks invoices, customers, and payments to QuickBooks Online (QBO) automatically as they are created or updated. Each ATLinks company connects their own QBO account via OAuth 2.0. Sync is async via River workers backed by PostgreSQL — ATLinks always saves immediately, QBO sync happens in the background with automatic retry.

## Scope

- **Direction:** ATLinks → QBO only
- **Trigger:** Auto-sync on create/update of customers, invoices, and payments
- **Entities:** Customers, Invoices (with line items), Payments
- **Multi-tenancy:** Per-company QBO OAuth connection
- **Plan gating:** Available on all current tiers via `FeatureQBO`; can be restricted to specific tiers with a one-line change to `TierFeatures`

## Data Layer

### New Table: `qbo_connections`

One row per company. Stores OAuth tokens and connection metadata.

| Column | Type | Notes |
|---|---|---|
| `id` | bigserial | PK |
| `company_id` | int | unique, FK → companies |
| `realm_id` | text | QBO company ID returned from OAuth |
| `access_token` | text | short-lived (~1 hour) |
| `refresh_token` | text | long-lived (100 days) |
| `token_expiry` | timestamptz | checked before every API call |
| `connected_by` | text | username who connected |
| `connected_at` | timestamptz | |
| `created_at` | timestamptz | |
| `updated_at` | timestamptz | |

### New Table: `qbo_sync_log`

Audit trail for every sync attempt. Used for debugging and surfacing errors in the UI.

| Column | Type | Notes |
|---|---|---|
| `id` | bigserial | PK |
| `company_id` | int | FK → companies |
| `entity_type` | text | "customer", "invoice", "payment" |
| `entity_id` | int | ATLinks entity ID |
| `qbo_id` | text | QBO entity ID (nullable, populated on success) |
| `action` | text | "create", "update", "void" |
| `status` | text | "pending", "success", "failed" |
| `error_message` | text | nullable |
| `attempted_at` | timestamptz | |
| `completed_at` | timestamptz | nullable |

### New Columns on Existing Tables

```sql
-- customers
ALTER TABLE customers ADD COLUMN qbo_customer_id text;

-- invoices
ALTER TABLE invoices ADD COLUMN qbo_invoice_id   text;
ALTER TABLE invoices ADD COLUMN qbo_sync_token   text;
ALTER TABLE invoices ADD COLUMN qbo_synced_at    timestamptz;

-- payments
ALTER TABLE payments ADD COLUMN qbo_payment_id   text;
ALTER TABLE payments ADD COLUMN qbo_sync_token   text;
ALTER TABLE payments ADD COLUMN qbo_synced_at    timestamptz;
```

### Feature Gate

Add `FeatureQBO` to `internal/models/subscription.go` and include it in all three tiers:

```go
const (
    // existing...
    FeatureQBO Feature = "quickbooks"
)

var TierFeatures = map[Tier][]Feature{
    TierBasic:      {..., FeatureQBO},
    TierPro:        {..., FeatureQBO},
    TierEnterprise: {..., FeatureQBO},
}
```

To restrict to a specific tier in the future, remove `FeatureQBO` from the tiers that should not have access.

## Package Structure

```
internal/
  qbo/
    client.go     # HTTP client: token refresh + raw QBO REST calls
    oauth.go      # OAuth 2.0 connect/callback/disconnect helpers
    mapper.go     # ATLinks models → QBO API request bodies
    models.go     # QBO-side types (QBOCustomer, QBOInvoice, QBOPayment)
  worker/
    qbo_customer.go  # River worker: upsert customer in QBO
    qbo_invoice.go   # River worker: create/update/void invoice in QBO
    qbo_payment.go   # River worker: create/update payment in QBO
```

Workers are registered in `cmd/server/main.go` alongside the River client setup.

## OAuth Connect Flow

**Routes:**
```
GET  /integrations/qbo/connect    → redirect to Intuit OAuth authorization URL
GET  /integrations/qbo/callback   → exchange code for tokens, store in qbo_connections
POST /integrations/qbo/disconnect → delete qbo_connections row for company
```

**Flow:**
1. User clicks "Connect to QuickBooks" on the Integrations settings page
2. ATLinks generates a state token (stored in session for CSRF protection), redirects to Intuit's OAuth 2.0 authorization URL with scope `com.intuit.quickbooks.accounting`
3. User authorizes in Intuit's UI; Intuit redirects to `/integrations/qbo/callback?code=...&realmId=...&state=...`
4. ATLinks validates state, exchanges code for access + refresh tokens via `golang.org/x/oauth2`, upserts `qbo_connections`
5. Settings page reflects connected state

**Token refresh:**

Before every QBO API call in `client.go`: if `token_expiry` is within 5 minutes, fetch a new token pair using the refresh token and update `qbo_connections`. If the refresh fails (e.g., refresh token expired after 100 days of inactivity), the connection is marked invalid and a notification is created for the company admin to reconnect.

## River Workers

Three worker types, all following the same pattern:

1. Load entity from DB
2. Check company has an active `qbo_connections` row; if not, discard job silently
3. Check `FeatureQBO` is enabled for the company; if not, discard
4. Refresh OAuth token if within 5 minutes of expiry
5. If `qbo_*_id` is null → POST (create); otherwise POST with ID + SyncToken (update)
6. On success: write back `qbo_*_id`, `qbo_sync_token`, `qbo_synced_at`; insert success row in `qbo_sync_log`
7. On 409 conflict (stale SyncToken): fetch current record from QBO for fresh SyncToken, retry once; if still failing let River handle backoff
8. On other failure: insert failed row in `qbo_sync_log`; River retries with exponential backoff

**Enqueue points:**
- `CustomerStore.Create` / `CustomerStore.Update` → enqueue `SyncCustomerWorker`
- `InvoiceStore.Create` / `InvoiceStore.Update` / `InvoiceService.VoidInvoice` → enqueue `SyncInvoiceWorker`
- `PaymentStore.Create` / `PaymentStore.Update` → enqueue `SyncPaymentWorker`

**Invoice dependency:** Invoice worker checks that `customer.qbo_customer_id` is set. If not, it enqueues a `SyncCustomerWorker` first and re-queues itself with a short delay.

## Entity Mapping

### Customer → QBO Customer

| QBO Field | ATLinks Source |
|---|---|
| `DisplayName` | `customer.Name` |
| `BillAddr.Line1` | `customer.Address` |
| `BillAddr.Line2` | `customer.Address2` |
| `BillAddr.City` | `customer.City` |
| `BillAddr.CountrySubDivisionCode` | `customer.State` |
| `BillAddr.PostalCode` | `customer.Zip` |
| `PrimaryPhone.FreeFormNumber` | `customer.Phone` |

### Invoice → QBO Invoice

| QBO Field | ATLinks Source |
|---|---|
| `DocNumber` | `invoice.InvoiceNumber` |
| `CustomerRef.value` | `customer.qbo_customer_id` |
| `TxnDate` | `invoice.InvoiceDate` |
| `DueDate` | `invoice.DueDate` |
| `PrivateNote` | `invoice.Comments` |
| `Line[]` | one `SalesItemLine` per `invoice_detail` |
| `Line[].Description` | `detail.Description` (or Year/Make/Model/VIN) |
| `Line[].Amount` | `detail.Amount` |
| `Line[].SalesItemLineDetail.Qty` | `detail.Qty` |
| `Line[].SalesItemLineDetail.UnitPrice` | `detail.Rate` |

Voided invoices are sent as a QBO void operation (POST with `?operation=void`).

### Payment → QBO Payment

| QBO Field | ATLinks Source |
|---|---|
| `CustomerRef.value` | `customer.qbo_customer_id` |
| `TotalAmt` | `payment.Amount` |
| `TxnDate` | `payment.PaymentDate` |
| `PaymentMethodRef` | mapped from `payment.PaymentMethod` |
| `Line[]` | one linked-txn line per `payment_detail` (invoice reference) |

## UI

### Integrations Settings Page (`/settings/integrations`)

Linked from company settings nav. Shows a QuickBooks Online card:

**Disconnected state:**
- Brief description of what syncs (customers, invoices, payments)
- "Connect to QuickBooks" button

**Connected state:**
- Green connected badge
- Realm ID, connected by, connected at
- Last sync time and status
- "Sync All" button — enqueues backfill jobs for all unsynced entities (useful after first connect or after resolving prolonged failures)
- "Disconnect" button with confirmation

### Sync Status Badges

Small QBO status indicator on invoice list/detail, customer list/detail, and payment list/detail rows:

- **Grey** — not yet synced (no connection or pending)
- **Green** — synced successfully (shows `qbo_synced_at` on hover)
- **Red** — last sync failed (shows error message on hover)

### Failure Notifications

When a River job exhausts all retries, a notification is created for the company admin (via the existing notification system) prompting them to check the Integrations page.

## Out of Scope

- QBO → ATLinks sync (payment reconciliation, customer import)
- Chart of accounts mapping
- Tax code mapping to QBO tax codes
- QuickBooks Desktop (this is QBO-only)
