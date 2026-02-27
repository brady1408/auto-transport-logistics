# QuickBooks Online Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Sync ATLinks customers, invoices, and payments to QuickBooks Online automatically via per-company OAuth 2.0 and async River workers.

**Architecture:** Each company connects their own QBO account via an OAuth 2.0 flow. After every customer/invoice/payment write, ATLinks enqueues a River job. The worker fetches fresh tokens, calls the QBO REST API, and writes back the QBO entity ID + SyncToken. Corrections push updates using the stored SyncToken with stale-token retry.

**Tech Stack:** `golang.org/x/oauth2` (OAuth 2.0), `github.com/riverqueue/river` + `riverpgxv5` (async workers), QBO REST API v3, existing pgx/v5 pool, templ for UI.

---

## Prerequisites

Before starting, register an app on the Intuit Developer Portal (developer.intuit.com):
1. Create an account and new app with "Accounting" scope
2. Note the **Client ID** and **Client Secret**
3. Add redirect URI: `{APP_BASE_URL}/integrations/qbo/callback`
4. Use **Sandbox** environment for development (separate credentials from Production)

---

### Task 1: Add Dependencies

**Files:**
- Modify: `go.mod`

**Step 1: Add River and oauth2 packages**

```bash
go get github.com/riverqueue/river@latest
go get github.com/riverqueue/river/riverdriver/riverpgxv5@latest
go get golang.org/x/oauth2@latest
```

**Step 2: Verify the build compiles**

```bash
go build ./...
```
Expected: no errors.

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add river and oauth2 dependencies"
```

---

### Task 2: Add FeatureQBO to Subscription Model

**Files:**
- Modify: `internal/models/subscription.go`

**Step 1: Add the constant and wire it into all tiers**

In `internal/models/subscription.go`, add `FeatureQBO` after `FeatureEDI` and add it to all three tiers:

```go
const (
    FeatureDispatch   Feature = "dispatch"
    FeatureAccounting Feature = "accounting"
    FeatureReports    Feature = "reports"
    FeatureLoadboard  Feature = "loadboard"
    FeatureEDI        Feature = "edi"
    FeatureQBO        Feature = "quickbooks"
)

var TierFeatures = map[Tier][]Feature{
    TierBasic: {
        FeatureDispatch,
        FeatureAccounting,
        FeatureReports,
        FeatureQBO,
    },
    TierPro: {
        FeatureDispatch,
        FeatureAccounting,
        FeatureReports,
        FeatureLoadboard,
        FeatureQBO,
    },
    TierEnterprise: {
        FeatureDispatch,
        FeatureAccounting,
        FeatureReports,
        FeatureLoadboard,
        FeatureEDI,
        FeatureQBO,
    },
}
```

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/models/subscription.go
git commit -m "feat: add FeatureQBO feature gate to all subscription tiers"
```

---

### Task 3: Add QBO Config Fields

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add QBO fields to Config struct and Load()**

```go
type Config struct {
    // ... existing fields ...
    QBOClientID     string
    QBOClientSecret string
    QBORedirectURL  string
    QBOSandbox      bool   // true in dev, false in production
}

func Load() (*Config, error) {
    cfg := &Config{
        // ... existing fields ...
        QBOClientID:     getEnv("QBO_CLIENT_ID", ""),
        QBOClientSecret: getEnv("QBO_CLIENT_SECRET", ""),
        QBORedirectURL:  getEnv("QBO_REDIRECT_URL", "http://localhost:8080/integrations/qbo/callback"),
        QBOSandbox:      getEnv("QBO_SANDBOX", "true") == "true",
    }
    // ...
}
```

**Step 2: Add to local `.env` (gitignored)**

Add these lines to your local `.env` file (not `.env.prod` yet — use sandbox credentials for dev):
```
QBO_CLIENT_ID=your_sandbox_client_id
QBO_CLIENT_SECRET=your_sandbox_client_secret
QBO_REDIRECT_URL=http://localhost:8080/integrations/qbo/callback
QBO_SANDBOX=true
```

**Step 3: Verify build**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add QBO OAuth config fields"
```

---

### Task 4: Database Migration

**Files:**
- Create: `internal/database/migrations/XXX_qbo_integration.up.sql` (use the next sequential number)
- Create: `internal/database/migrations/XXX_qbo_integration.down.sql`

**Step 1: Create the up migration**

```sql
-- qbo_connections: one row per company, stores OAuth tokens
CREATE TABLE qbo_connections (
    id              bigserial PRIMARY KEY,
    company_id      integer NOT NULL UNIQUE REFERENCES companies(id) ON DELETE CASCADE,
    realm_id        text    NOT NULL,
    access_token    text    NOT NULL,
    refresh_token   text    NOT NULL,
    token_expiry    timestamptz NOT NULL,
    connected_by    text    NOT NULL,
    connected_at    timestamptz NOT NULL DEFAULT NOW(),
    created_at      timestamptz NOT NULL DEFAULT NOW(),
    updated_at      timestamptz NOT NULL DEFAULT NOW()
);

-- qbo_sync_log: audit trail for every sync attempt
CREATE TABLE qbo_sync_log (
    id            bigserial PRIMARY KEY,
    company_id    integer NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    entity_type   text    NOT NULL CHECK (entity_type IN ('customer', 'invoice', 'payment')),
    entity_id     integer NOT NULL,
    qbo_id        text,
    action        text    NOT NULL CHECK (action IN ('create', 'update', 'void')),
    status        text    NOT NULL CHECK (status IN ('success', 'failed')),
    error_message text,
    attempted_at  timestamptz NOT NULL DEFAULT NOW(),
    completed_at  timestamptz
);

CREATE INDEX idx_qbo_sync_log_company_entity ON qbo_sync_log(company_id, entity_type, entity_id);

-- Add QBO columns to existing tables
ALTER TABLE customers ADD COLUMN qbo_customer_id text;

ALTER TABLE invoices ADD COLUMN qbo_invoice_id  text;
ALTER TABLE invoices ADD COLUMN qbo_sync_token  text;
ALTER TABLE invoices ADD COLUMN qbo_synced_at   timestamptz;

ALTER TABLE payments ADD COLUMN qbo_payment_id  text;
ALTER TABLE payments ADD COLUMN qbo_sync_token  text;
ALTER TABLE payments ADD COLUMN qbo_synced_at   timestamptz;
```

**Step 2: Create the down migration**

```sql
ALTER TABLE payments DROP COLUMN IF EXISTS qbo_payment_id;
ALTER TABLE payments DROP COLUMN IF EXISTS qbo_sync_token;
ALTER TABLE payments DROP COLUMN IF EXISTS qbo_synced_at;

ALTER TABLE invoices DROP COLUMN IF EXISTS qbo_invoice_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS qbo_sync_token;
ALTER TABLE invoices DROP COLUMN IF EXISTS qbo_synced_at;

ALTER TABLE customers DROP COLUMN IF EXISTS qbo_customer_id;

DROP TABLE IF EXISTS qbo_sync_log;
DROP TABLE IF EXISTS qbo_connections;
```

**Step 3: Run migration**

```bash
make migrate-up
```
Expected: migration runs without errors.

**Step 4: Add QBO fields to existing models**

In `internal/models/customer.go`, add:
```go
QBOCustomerID *string `json:"qbo_customer_id,omitempty"`
```

In `internal/models/invoice.go`, add:
```go
QBOInvoiceID *string    `json:"qbo_invoice_id,omitempty"`
QBOSyncToken *string    `json:"qbo_sync_token,omitempty"`
QBOSyncedAt  *time.Time `json:"qbo_synced_at,omitempty"`
```

In `internal/models/payment.go`, add:
```go
QBOPaymentID *string    `json:"qbo_payment_id,omitempty"`
QBOSyncToken *string    `json:"qbo_sync_token,omitempty"`
QBOSyncedAt  *time.Time `json:"qbo_synced_at,omitempty"`
```

**Step 5: Update scanCustomer, scanInvoice, scanPayment in their respective stores to include the new fields**

In `internal/store/customer_store.go`:
- Add `qbo_customer_id` to `customerColumns` const
- Add `&c.QBOCustomerID` to `scanCustomer`

In `internal/store/invoice_store.go`:
- Add `qbo_invoice_id, qbo_sync_token, qbo_synced_at` to `invoiceColumns`
- Add `&inv.QBOInvoiceID, &inv.QBOSyncToken, &inv.QBOSyncedAt` to `scanInvoice`

In `internal/store/payment_store.go`:
- Add `qbo_payment_id, qbo_sync_token, qbo_synced_at` to `paymentColumns`
- Add the three fields to `scanPayment`

**Step 6: Verify build**

```bash
go build ./...
```

**Step 7: Commit**

```bash
git add internal/database/migrations/ internal/models/ internal/store/
git commit -m "feat: add QBO schema migration and model fields"
```

---

### Task 5: River Schema Migration

River needs its own tables in PostgreSQL. Add them via a goose migration.

**Files:**
- Create: `internal/database/migrations/XXX_river_schema.up.sql`
- Create: `internal/database/migrations/XXX_river_schema.down.sql`

**Step 1: Get River's SQL**

River ships SQL migrations. Run this to print them:

```bash
go run github.com/riverqueue/river/cmd/river@latest migrate-get --all --direction up
```

Copy the output into `XXX_river_schema.up.sql`. Do the same with `--direction down` for the down file.

Alternatively, find River's migration SQL in `$(go env GOPATH)/pkg/mod/github.com/riverqueue/river*/riverdriver/riverpgxv5/migration/`.

**Step 2: Run migration**

```bash
make migrate-up
```
Expected: River tables created (river_job, river_leader, river_migration, river_queue).

**Step 3: Commit**

```bash
git add internal/database/migrations/
git commit -m "chore: add River job queue schema migration"
```

---

### Task 6: QBO Models

**Files:**
- Create: `internal/qbo/models.go`
- Create: `internal/models/qbo_connection.go`

**Step 1: Create ATLinks-side QBO connection model**

`internal/models/qbo_connection.go`:

```go
package models

import "time"

type QBOConnection struct {
    ID           int
    CompanyID    int
    RealmID      string
    AccessToken  string
    RefreshToken string
    TokenExpiry  time.Time
    ConnectedBy  string
    ConnectedAt  time.Time
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

type QBOSyncLog struct {
    ID           int64
    CompanyID    int
    EntityType   string // "customer", "invoice", "payment"
    EntityID     int
    QBOID        *string
    Action       string // "create", "update", "void"
    Status       string // "success", "failed"
    ErrorMessage *string
    AttemptedAt  time.Time
    CompletedAt  *time.Time
}
```

**Step 2: Create QBO API types**

`internal/qbo/models.go`:

```go
package qbo

// Addr is a QBO mailing address.
type Addr struct {
    Line1                  string `json:"Line1,omitempty"`
    Line2                  string `json:"Line2,omitempty"`
    City                   string `json:"City,omitempty"`
    CountrySubDivisionCode string `json:"CountrySubDivisionCode,omitempty"`
    PostalCode             string `json:"PostalCode,omitempty"`
}

// Phone is a QBO phone number.
type Phone struct {
    FreeFormNumber string `json:"FreeFormNumber,omitempty"`
}

// Ref is a QBO entity reference (Id + optional Name).
type Ref struct {
    Value string `json:"value"`
    Name  string `json:"name,omitempty"`
}

// Customer is the QBO Customer object (subset of fields we use).
type Customer struct {
    ID           string  `json:"Id,omitempty"`
    SyncToken    string  `json:"SyncToken,omitempty"`
    DisplayName  string  `json:"DisplayName"`
    BillAddr     *Addr   `json:"BillAddr,omitempty"`
    PrimaryPhone *Phone  `json:"PrimaryPhone,omitempty"`
    Active       bool    `json:"Active"`
}

// CustomerResponse wraps QBO's create/update response.
type CustomerResponse struct {
    Customer Customer `json:"Customer"`
}

// SalesItemLineDetail is a QBO line item.
type SalesItemLineDetail struct {
    Qty       float64 `json:"Qty,omitempty"`
    UnitPrice float64 `json:"UnitPrice,omitempty"`
}

// Line is a single line on a QBO invoice.
type Line struct {
    DetailType          string               `json:"DetailType"`
    Amount              float64              `json:"Amount"`
    Description         string               `json:"Description,omitempty"`
    SalesItemLineDetail *SalesItemLineDetail `json:"SalesItemLineDetail,omitempty"`
}

// Invoice is the QBO Invoice object.
type Invoice struct {
    ID          string  `json:"Id,omitempty"`
    SyncToken   string  `json:"SyncToken,omitempty"`
    DocNumber   string  `json:"DocNumber,omitempty"`
    CustomerRef *Ref    `json:"CustomerRef"`
    TxnDate     string  `json:"TxnDate,omitempty"` // "2006-01-02"
    DueDate     string  `json:"DueDate,omitempty"`
    Line        []Line  `json:"Line"`
    PrivateNote string  `json:"PrivateNote,omitempty"`
}

// InvoiceResponse wraps QBO's invoice response.
type InvoiceResponse struct {
    Invoice Invoice `json:"Invoice"`
}

// LinkedTxn links a payment line to an invoice.
type LinkedTxn struct {
    TxnID   string `json:"TxnId"`
    TxnType string `json:"TxnType"` // "Invoice"
}

// PaymentLine links a payment amount to an invoice.
type PaymentLine struct {
    Amount    float64     `json:"Amount"`
    LinkedTxn []LinkedTxn `json:"LinkedTxn,omitempty"`
}

// Payment is the QBO Payment object.
type Payment struct {
    ID          string        `json:"Id,omitempty"`
    SyncToken   string        `json:"SyncToken,omitempty"`
    CustomerRef *Ref          `json:"CustomerRef"`
    TotalAmt    float64       `json:"TotalAmt"`
    TxnDate     string        `json:"TxnDate,omitempty"`
    Line        []PaymentLine `json:"Line,omitempty"`
}

// PaymentResponse wraps QBO's payment response.
type PaymentResponse struct {
    Payment Payment `json:"Payment"`
}
```

**Step 3: Verify build**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add internal/qbo/ internal/models/qbo_connection.go
git commit -m "feat: add QBO API models and connection model"
```

---

### Task 7: QBO Store

**Files:**
- Create: `internal/store/qbo_store.go`

**Step 1: Write the store**

```go
package store

import (
    "context"
    "fmt"
    "time"

    "github.com/brady1408/atlinks/internal/models"
    "github.com/jackc/pgx/v5/pgxpool"
)

type QBOStore struct {
    pool *pgxpool.Pool
}

func NewQBOStore(pool *pgxpool.Pool) *QBOStore {
    return &QBOStore{pool: pool}
}

func (s *QBOStore) GetConnection(ctx context.Context, companyID int) (*models.QBOConnection, error) {
    var c models.QBOConnection
    err := s.pool.QueryRow(ctx, `
        SELECT id, company_id, realm_id, access_token, refresh_token,
               token_expiry, connected_by, connected_at, created_at, updated_at
        FROM qbo_connections WHERE company_id = $1`, companyID,
    ).Scan(&c.ID, &c.CompanyID, &c.RealmID, &c.AccessToken, &c.RefreshToken,
        &c.TokenExpiry, &c.ConnectedBy, &c.ConnectedAt, &c.CreatedAt, &c.UpdatedAt)
    if err != nil {
        return nil, fmt.Errorf("get qbo connection: %w", err)
    }
    return &c, nil
}

func (s *QBOStore) UpsertConnection(ctx context.Context, c *models.QBOConnection) error {
    _, err := s.pool.Exec(ctx, `
        INSERT INTO qbo_connections
            (company_id, realm_id, access_token, refresh_token, token_expiry, connected_by, connected_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW())
        ON CONFLICT (company_id) DO UPDATE SET
            realm_id      = EXCLUDED.realm_id,
            access_token  = EXCLUDED.access_token,
            refresh_token = EXCLUDED.refresh_token,
            token_expiry  = EXCLUDED.token_expiry,
            connected_by  = EXCLUDED.connected_by,
            connected_at  = NOW(),
            updated_at    = NOW()`,
        c.CompanyID, c.RealmID, c.AccessToken, c.RefreshToken, c.TokenExpiry, c.ConnectedBy,
    )
    return err
}

func (s *QBOStore) UpdateTokens(ctx context.Context, companyID int, accessToken, refreshToken string, expiry time.Time) error {
    _, err := s.pool.Exec(ctx, `
        UPDATE qbo_connections
        SET access_token = $2, refresh_token = $3, token_expiry = $4, updated_at = NOW()
        WHERE company_id = $1`,
        companyID, accessToken, refreshToken, expiry,
    )
    return err
}

func (s *QBOStore) DeleteConnection(ctx context.Context, companyID int) error {
    _, err := s.pool.Exec(ctx, `DELETE FROM qbo_connections WHERE company_id = $1`, companyID)
    return err
}

func (s *QBOStore) Log(ctx context.Context, entry *models.QBOSyncLog) error {
    _, err := s.pool.Exec(ctx, `
        INSERT INTO qbo_sync_log
            (company_id, entity_type, entity_id, qbo_id, action, status, error_message, attempted_at, completed_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
        entry.CompanyID, entry.EntityType, entry.EntityID, entry.QBOID,
        entry.Action, entry.Status, entry.ErrorMessage,
    )
    return err
}

func (s *QBOStore) RecentFailures(ctx context.Context, companyID int) ([]models.QBOSyncLog, error) {
    rows, err := s.pool.Query(ctx, `
        SELECT id, company_id, entity_type, entity_id, qbo_id, action, status,
               error_message, attempted_at, completed_at
        FROM qbo_sync_log
        WHERE company_id = $1 AND status = 'failed'
        ORDER BY attempted_at DESC LIMIT 20`,
        companyID,
    )
    if err != nil {
        return nil, fmt.Errorf("recent failures: %w", err)
    }
    defer rows.Close()

    var results []models.QBOSyncLog
    for rows.Next() {
        var e models.QBOSyncLog
        if err := rows.Scan(&e.ID, &e.CompanyID, &e.EntityType, &e.EntityID, &e.QBOID,
            &e.Action, &e.Status, &e.ErrorMessage, &e.AttemptedAt, &e.CompletedAt); err != nil {
            return nil, err
        }
        results = append(results, e)
    }
    return results, rows.Err()
}

// UpdateCustomerQBOID writes the QBO customer ID back to the customers table.
func (s *QBOStore) UpdateCustomerQBOID(ctx context.Context, customerID int, qboID string) error {
    _, err := s.pool.Exec(ctx,
        `UPDATE customers SET qbo_customer_id = $2 WHERE id = $1`,
        customerID, qboID,
    )
    return err
}

// UpdateInvoiceQBO writes the QBO invoice ID + SyncToken back to the invoices table.
func (s *QBOStore) UpdateInvoiceQBO(ctx context.Context, invoiceID int, qboID, syncToken string) error {
    _, err := s.pool.Exec(ctx,
        `UPDATE invoices SET qbo_invoice_id = $2, qbo_sync_token = $3, qbo_synced_at = NOW() WHERE id = $1`,
        invoiceID, qboID, syncToken,
    )
    return err
}

// UpdatePaymentQBO writes the QBO payment ID + SyncToken back to the payments table.
func (s *QBOStore) UpdatePaymentQBO(ctx context.Context, paymentID int, qboID, syncToken string) error {
    _, err := s.pool.Exec(ctx,
        `UPDATE payments SET qbo_payment_id = $2, qbo_sync_token = $3, qbo_synced_at = NOW() WHERE id = $1`,
        paymentID, qboID, syncToken,
    )
    return err
}
```

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/store/qbo_store.go
git commit -m "feat: add QBO store for connections and sync log"
```

---

### Task 8: QBO OAuth Helper

**Files:**
- Create: `internal/qbo/oauth.go`

**Step 1: Write the OAuth config builder**

```go
package qbo

import (
    "golang.org/x/oauth2"
)

const (
    authURL  = "https://appcenter.intuit.com/connect/oauth2"
    tokenURL = "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer"
    scope    = "com.intuit.quickbooks.accounting"
)

// NewOAuthConfig returns an oauth2.Config for QBO.
func NewOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
    return &oauth2.Config{
        ClientID:     clientID,
        ClientSecret: clientSecret,
        RedirectURL:  redirectURL,
        Scopes:       []string{scope},
        Endpoint: oauth2.Endpoint{
            AuthURL:  authURL,
            TokenURL: tokenURL,
        },
    }
}
```

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/qbo/oauth.go
git commit -m "feat: add QBO OAuth config helper"
```

---

### Task 9: QBO HTTP Client

**Files:**
- Create: `internal/qbo/client.go`

**Step 1: Write the client**

The client holds the oauth2.Config and QBOStore reference. Before every request it checks token expiry and refreshes if needed.

```go
package qbo

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "time"

    "github.com/brady1408/atlinks/internal/models"
    "github.com/brady1408/atlinks/internal/store"
    "golang.org/x/oauth2"
)

const (
    productionBase = "https://quickbooks.api.intuit.com/v3/company"
    sandboxBase    = "https://sandbox-quickbooks.api.intuit.com/v3/company"
    minorVersion   = "?minorversion=65"
)

// Client calls the QBO REST API for a specific company.
type Client struct {
    oauthCfg  *oauth2.Config
    qboStore  *store.QBOStore
    conn      *models.QBOConnection
    sandbox   bool
    httpClient *http.Client
}

// NewClient returns a Client ready to make calls for the given connection.
func NewClient(oauthCfg *oauth2.Config, qboStore *store.QBOStore, conn *models.QBOConnection, sandbox bool) *Client {
    return &Client{
        oauthCfg:   oauthCfg,
        qboStore:   qboStore,
        conn:       conn,
        sandbox:    sandbox,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *Client) baseURL() string {
    if c.sandbox {
        return sandboxBase
    }
    return productionBase
}

// ensureFreshToken refreshes the access token if it expires within 5 minutes.
func (c *Client) ensureFreshToken(ctx context.Context) error {
    if time.Until(c.conn.TokenExpiry) > 5*time.Minute {
        return nil
    }
    t := &oauth2.Token{
        AccessToken:  c.conn.AccessToken,
        RefreshToken: c.conn.RefreshToken,
        Expiry:       c.conn.TokenExpiry,
    }
    newToken, err := c.oauthCfg.TokenSource(ctx, t).Token()
    if err != nil {
        return fmt.Errorf("refresh qbo token: %w", err)
    }
    if err := c.qboStore.UpdateTokens(ctx, c.conn.CompanyID, newToken.AccessToken, newToken.RefreshToken, newToken.Expiry); err != nil {
        return fmt.Errorf("save refreshed token: %w", err)
    }
    c.conn.AccessToken = newToken.AccessToken
    c.conn.RefreshToken = newToken.RefreshToken
    c.conn.TokenExpiry = newToken.Expiry
    return nil
}

// do performs an authenticated JSON request, returning the response body bytes.
func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, int, error) {
    if err := c.ensureFreshToken(ctx); err != nil {
        return nil, 0, err
    }

    var bodyReader io.Reader
    if body != nil {
        b, err := json.Marshal(body)
        if err != nil {
            return nil, 0, fmt.Errorf("marshal request: %w", err)
        }
        bodyReader = bytes.NewReader(b)
    }

    req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+"/"+c.conn.RealmID+path, bodyReader)
    if err != nil {
        return nil, 0, err
    }
    req.Header.Set("Authorization", "Bearer "+c.conn.AccessToken)
    req.Header.Set("Accept", "application/json")
    if body != nil {
        req.Header.Set("Content-Type", "application/json")
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, 0, fmt.Errorf("qbo request: %w", err)
    }
    defer resp.Body.Close()
    respBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, resp.StatusCode, fmt.Errorf("read qbo response: %w", err)
    }
    return respBytes, resp.StatusCode, nil
}

// UpsertCustomer creates or updates a QBO Customer. Returns the QBO Customer ID.
func (c *Client) UpsertCustomer(ctx context.Context, cust Customer) (string, error) {
    path := "/customer" + minorVersion
    b, status, err := c.do(ctx, http.MethodPost, path, cust)
    if err != nil {
        return "", err
    }
    if status != http.StatusOK {
        return "", fmt.Errorf("qbo customer upsert status %d: %s", status, b)
    }
    var resp CustomerResponse
    if err := json.Unmarshal(b, &resp); err != nil {
        return "", fmt.Errorf("unmarshal customer response: %w", err)
    }
    return resp.Customer.ID, nil
}

// UpsertInvoice creates or updates a QBO Invoice. Returns QBO Invoice ID and SyncToken.
func (c *Client) UpsertInvoice(ctx context.Context, inv Invoice) (id, syncToken string, err error) {
    path := "/invoice" + minorVersion
    b, status, reqErr := c.do(ctx, http.MethodPost, path, inv)
    if reqErr != nil {
        return "", "", reqErr
    }
    if status == http.StatusConflict {
        return "", "", &SyncTokenError{EntityID: inv.ID}
    }
    if status != http.StatusOK {
        return "", "", fmt.Errorf("qbo invoice upsert status %d: %s", status, b)
    }
    var resp InvoiceResponse
    if err := json.Unmarshal(b, &resp); err != nil {
        return "", "", fmt.Errorf("unmarshal invoice response: %w", err)
    }
    return resp.Invoice.ID, resp.Invoice.SyncToken, nil
}

// VoidInvoice voids a QBO invoice by ID and SyncToken.
func (c *Client) VoidInvoice(ctx context.Context, qboID, syncToken string) error {
    path := "/invoice" + minorVersion + "&operation=void"
    body := map[string]string{"Id": qboID, "SyncToken": syncToken}
    b, status, err := c.do(ctx, http.MethodPost, path, body)
    if err != nil {
        return err
    }
    if status != http.StatusOK {
        return fmt.Errorf("qbo void invoice status %d: %s", status, b)
    }
    return nil
}

// GetInvoiceSyncToken fetches the current SyncToken for a QBO invoice (used after 409 conflict).
func (c *Client) GetInvoiceSyncToken(ctx context.Context, qboID string) (string, error) {
    path := "/invoice/" + qboID
    b, status, err := c.do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return "", err
    }
    if status != http.StatusOK {
        return "", fmt.Errorf("qbo get invoice status %d: %s", status, b)
    }
    var resp InvoiceResponse
    if err := json.Unmarshal(b, &resp); err != nil {
        return "", fmt.Errorf("unmarshal invoice: %w", err)
    }
    return resp.Invoice.SyncToken, nil
}

// UpsertPayment creates or updates a QBO Payment. Returns QBO Payment ID and SyncToken.
func (c *Client) UpsertPayment(ctx context.Context, pmt Payment) (id, syncToken string, err error) {
    path := "/payment" + minorVersion
    b, status, reqErr := c.do(ctx, http.MethodPost, path, pmt)
    if reqErr != nil {
        return "", "", reqErr
    }
    if status != http.StatusOK {
        return "", "", fmt.Errorf("qbo payment upsert status %d: %s", status, b)
    }
    var resp PaymentResponse
    if err := json.Unmarshal(b, &resp); err != nil {
        return "", "", fmt.Errorf("unmarshal payment response: %w", err)
    }
    return resp.Payment.ID, resp.Payment.SyncToken, nil
}

// SyncTokenError is returned when QBO responds with a 409 (stale SyncToken).
type SyncTokenError struct {
    EntityID string
}

func (e *SyncTokenError) Error() string {
    return "qbo sync token conflict for entity " + e.EntityID
}

// intToStr safely converts an *int to string for QBO Qty fields.
func intToStr(n *int) float64 {
    if n == nil {
        return 1
    }
    return float64(*n)
}

// strToFloat parses a *string amount (stored as numeric text in ATLinks) to float64.
func strToFloat(s *string) float64 {
    if s == nil {
        return 0
    }
    f, _ := strconv.ParseFloat(*s, 64)
    return f
}
```

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/qbo/client.go
git commit -m "feat: add QBO HTTP client with token refresh"
```

---

### Task 10: QBO Mapper

**Files:**
- Create: `internal/qbo/mapper.go`

The mapper converts ATLinks models to QBO API request objects. It is pure data transformation — no I/O.

**Step 1: Write a failing test first**

Create `internal/qbo/mapper_test.go`:

```go
package qbo_test

import (
    "testing"

    "github.com/brady1408/atlinks/internal/models"
    "github.com/brady1408/atlinks/internal/qbo"
)

func strPtr(s string) *string { return &s }
func intPtr(n int) *int       { return &n }

func TestMapCustomer(t *testing.T) {
    c := models.Customer{
        Name:    "Acme Transport",
        Address: strPtr("123 Main St"),
        City:    strPtr("Denver"),
        State:   strPtr("CO"),
        Zip:     strPtr("80201"),
        Phone:   strPtr("303-555-1234"),
    }
    got := qbo.MapCustomer(c)
    if got.DisplayName != "Acme Transport" {
        t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Acme Transport")
    }
    if got.BillAddr == nil || got.BillAddr.Line1 != "123 Main St" {
        t.Error("expected BillAddr.Line1 = 123 Main St")
    }
    if got.BillAddr.City != "Denver" {
        t.Errorf("City = %q, want Denver", got.BillAddr.City)
    }
}

func TestMapCustomerUpdate(t *testing.T) {
    c := models.Customer{Name: "Updated Co", QBOCustomerID: strPtr("42")}
    got := qbo.MapCustomer(c)
    if got.ID != "42" {
        t.Errorf("ID = %q, want 42", got.ID)
    }
}
```

**Step 2: Run to verify it fails**

```bash
go test ./internal/qbo/... -v
```
Expected: compile error — `qbo.MapCustomer` undefined.

**Step 3: Write the mapper**

`internal/qbo/mapper.go`:

```go
package qbo

import (
    "fmt"

    "github.com/brady1408/atlinks/internal/models"
)

// MapCustomer converts an ATLinks Customer to a QBO Customer request.
func MapCustomer(c models.Customer) Customer {
    q := Customer{
        DisplayName: c.Name,
        Active:      !c.Inactive,
    }
    if c.QBOCustomerID != nil {
        q.ID = *c.QBOCustomerID
    }
    addr := &Addr{}
    hasAddr := false
    if c.Address != nil {
        addr.Line1 = *c.Address
        hasAddr = true
    }
    if c.Address2 != nil {
        addr.Line2 = *c.Address2
        hasAddr = true
    }
    if c.City != nil {
        addr.City = *c.City
        hasAddr = true
    }
    if c.State != nil {
        addr.CountrySubDivisionCode = *c.State
        hasAddr = true
    }
    if c.Zip != nil {
        addr.PostalCode = *c.Zip
        hasAddr = true
    }
    if hasAddr {
        q.BillAddr = addr
    }
    if c.Phone != nil && *c.Phone != "" {
        q.PrimaryPhone = &Phone{FreeFormNumber: *c.Phone}
    }
    return q
}

// MapInvoice converts an ATLinks Invoice + its details to a QBO Invoice request.
// qboCustomerID must already be resolved before calling this.
func MapInvoice(inv models.Invoice, details []models.InvoiceDetail, qboCustomerID string) Invoice {
    q := Invoice{
        CustomerRef: &Ref{Value: qboCustomerID},
        DocNumber:   inv.InvoiceNumber,
    }
    if inv.QBOInvoiceID != nil {
        q.ID = *inv.QBOInvoiceID
        if inv.QBOSyncToken != nil {
            q.SyncToken = *inv.QBOSyncToken
        }
    }
    if inv.InvoiceDate != nil {
        q.TxnDate = inv.InvoiceDate.Format("2006-01-02")
    }
    if inv.DueDate != nil {
        q.DueDate = inv.DueDate.Format("2006-01-02")
    }
    if inv.Comments != nil {
        q.PrivateNote = *inv.Comments
    }
    for _, d := range details {
        desc := ""
        if d.Description != nil {
            desc = *d.Description
        } else if d.Year != nil || d.Make != nil || d.Model != nil {
            desc = fmt.Sprintf("%s %s %s",
                strDeref(d.Year), strDeref(d.Make), strDeref(d.Model))
            if d.VIN != nil {
                desc += " VIN:" + *d.VIN
            }
        }
        amt := strToFloat(d.Amount)
        q.Line = append(q.Line, Line{
            DetailType:  "SalesItemLineDetail",
            Amount:      amt,
            Description: desc,
            SalesItemLineDetail: &SalesItemLineDetail{
                Qty:       intToStr(d.Qty),
                UnitPrice: strToFloat(d.Rate),
            },
        })
    }
    return q
}

// MapPayment converts an ATLinks Payment + its details to a QBO Payment request.
// qboCustomerID must be resolved. qboInvoiceIDs maps ATLinks invoice_id → QBO invoice ID.
func MapPayment(pmt models.Payment, details []models.PaymentDetail, qboCustomerID string, qboInvoiceIDs map[int]string) Payment {
    q := Payment{
        CustomerRef: &Ref{Value: qboCustomerID},
        TotalAmt:    strToFloat(pmt.Amount),
    }
    if pmt.QBOPaymentID != nil {
        q.ID = *pmt.QBOPaymentID
        if pmt.QBOSyncToken != nil {
            q.SyncToken = *pmt.QBOSyncToken
        }
    }
    if pmt.PaymentDate != nil {
        q.TxnDate = pmt.PaymentDate.Format("2006-01-02")
    }
    for _, d := range details {
        if d.InvoiceID == nil {
            continue
        }
        qboInvID, ok := qboInvoiceIDs[*d.InvoiceID]
        if !ok {
            continue
        }
        q.Line = append(q.Line, PaymentLine{
            Amount: strToFloat(d.Amount),
            LinkedTxn: []LinkedTxn{
                {TxnID: qboInvID, TxnType: "Invoice"},
            },
        })
    }
    return q
}

func strDeref(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/qbo/... -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/qbo/mapper.go internal/qbo/mapper_test.go
git commit -m "feat: add QBO entity mapper with tests"
```

---

### Task 11: River Worker — Customer

**Files:**
- Create: `internal/worker/qbo_customer.go`

**Step 1: Write the worker**

```go
package worker

import (
    "context"
    "errors"
    "fmt"
    "log"

    "github.com/brady1408/atlinks/internal/models"
    "github.com/brady1408/atlinks/internal/qbo"
    "github.com/brady1408/atlinks/internal/store"
    "github.com/jackc/pgx/v5"
    "github.com/riverqueue/river"
)

// SyncCustomerArgs is the River job payload for syncing a customer to QBO.
type SyncCustomerArgs struct {
    CompanyID  int `json:"company_id"`
    CustomerID int `json:"customer_id"`
}

func (SyncCustomerArgs) Kind() string { return "qbo_sync_customer" }

type SyncCustomerWorker struct {
    river.WorkerDefaults[SyncCustomerArgs]
    CustomerStore *store.CustomerStore
    QBOStore      *store.QBOStore
    OAuthCfg      *oauth2cfg // see Task 13 for how this is threaded in
    Sandbox       bool
}

func (w *SyncCustomerWorker) Work(ctx context.Context, job *river.Job[SyncCustomerArgs]) error {
    args := job.Args

    // Load connection — skip silently if not connected
    conn, err := w.QBOStore.GetConnection(ctx, args.CompanyID)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil // not connected, skip
        }
        return fmt.Errorf("get qbo connection: %w", err)
    }

    // Load customer
    cust, err := w.CustomerStore.GetByID(ctx, args.CustomerID)
    if err != nil {
        return fmt.Errorf("load customer %d: %w", args.CustomerID, err)
    }

    client := qbo.NewClient(w.OAuthCfg, w.QBOStore, conn, w.Sandbox)
    qboCustomer := qbo.MapCustomer(*cust)

    action := "create"
    if cust.QBOCustomerID != nil {
        action = "update"
    }

    qboID, err := client.UpsertCustomer(ctx, qboCustomer)
    if err != nil {
        logEntry := &models.QBOSyncLog{
            CompanyID:    args.CompanyID,
            EntityType:   "customer",
            EntityID:     args.CustomerID,
            Action:       action,
            Status:       "failed",
            ErrorMessage: strPtr(err.Error()),
        }
        _ = w.QBOStore.Log(ctx, logEntry)
        return fmt.Errorf("qbo upsert customer: %w", err)
    }

    if err := w.QBOStore.UpdateCustomerQBOID(ctx, args.CustomerID, qboID); err != nil {
        log.Printf("warn: update customer qbo_id: %v", err)
    }

    _ = w.QBOStore.Log(ctx, &models.QBOSyncLog{
        CompanyID:  args.CompanyID,
        EntityType: "customer",
        EntityID:   args.CustomerID,
        QBOID:      &qboID,
        Action:     action,
        Status:     "success",
    })
    return nil
}

func strPtr(s string) *string { return &s }
```

Note: `oauth2cfg` type alias will be `*oauth2.Config` — see Task 13 for the full wiring. For now use `interface{}` as a placeholder if needed to get the build to pass, then fill in during Task 13.

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/worker/qbo_customer.go
git commit -m "feat: add River worker for QBO customer sync"
```

---

### Task 12: River Worker — Invoice

**Files:**
- Create: `internal/worker/qbo_invoice.go`

**Step 1: Write the worker**

The invoice worker must:
1. Check that the customer has a `qbo_customer_id` — if not, re-enqueue a customer sync and retry after a short delay
2. Handle stale SyncToken (fetch fresh token from QBO and retry once)

```go
package worker

import (
    "context"
    "errors"
    "fmt"

    "github.com/brady1408/atlinks/internal/models"
    "github.com/brady1408/atlinks/internal/qbo"
    "github.com/brady1408/atlinks/internal/store"
    "github.com/jackc/pgx/v5"
    "github.com/riverqueue/river"
)

type SyncInvoiceArgs struct {
    CompanyID int    `json:"company_id"`
    InvoiceID int    `json:"invoice_id"`
    Action    string `json:"action"` // "create", "update", "void"
}

func (SyncInvoiceArgs) Kind() string { return "qbo_sync_invoice" }

type SyncInvoiceWorker struct {
    river.WorkerDefaults[SyncInvoiceArgs]
    InvoiceStore       *store.InvoiceStore
    InvoiceDetailStore *store.InvoiceDetailStore
    CustomerStore      *store.CustomerStore
    QBOStore           *store.QBOStore
    RiverClient        *river.Client[pgx.Tx]
    OAuthCfg           *oauth2cfg
    Sandbox            bool
}

func (w *SyncInvoiceWorker) Work(ctx context.Context, job *river.Job[SyncInvoiceArgs]) error {
    args := job.Args

    conn, err := w.QBOStore.GetConnection(ctx, args.CompanyID)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil
        }
        return fmt.Errorf("get qbo connection: %w", err)
    }

    inv, err := w.InvoiceStore.GetByID(ctx, args.InvoiceID)
    if err != nil {
        return fmt.Errorf("load invoice %d: %w", args.InvoiceID, err)
    }

    // Ensure customer is synced first
    if inv.CustomerID == nil {
        return nil // no customer, skip
    }
    cust, err := w.CustomerStore.GetByID(ctx, *inv.CustomerID)
    if err != nil {
        return fmt.Errorf("load customer: %w", err)
    }
    if cust.QBOCustomerID == nil {
        // Enqueue customer sync and retry this job via River's snooze mechanism
        _, _ = w.RiverClient.Insert(ctx, SyncCustomerArgs{
            CompanyID:  args.CompanyID,
            CustomerID: *inv.CustomerID,
        }, nil)
        return river.JobSnooze(30) // retry in 30 seconds
    }

    client := qbo.NewClient(w.OAuthCfg, w.QBOStore, conn, w.Sandbox)

    // Handle void separately
    if args.Action == "void" && inv.QBOInvoiceID != nil {
        syncToken := ""
        if inv.QBOSyncToken != nil {
            syncToken = *inv.QBOSyncToken
        }
        if err := client.VoidInvoice(ctx, *inv.QBOInvoiceID, syncToken); err != nil {
            logFail(ctx, w.QBOStore, args.CompanyID, "invoice", args.InvoiceID, "void", err)
            return err
        }
        logOK(ctx, w.QBOStore, args.CompanyID, "invoice", args.InvoiceID, inv.QBOInvoiceID, "void")
        return nil
    }

    details, err := w.InvoiceDetailStore.ListByInvoice(ctx, args.InvoiceID)
    if err != nil {
        return fmt.Errorf("load invoice details: %w", err)
    }

    qboInv := qbo.MapInvoice(*inv, details, *cust.QBOCustomerID)

    id, syncToken, err := client.UpsertInvoice(ctx, qboInv)
    if err != nil {
        // Handle stale SyncToken: fetch fresh and retry once
        var staleErr *qbo.SyncTokenError
        if errors.As(err, &staleErr) && inv.QBOInvoiceID != nil {
            freshToken, fetchErr := client.GetInvoiceSyncToken(ctx, *inv.QBOInvoiceID)
            if fetchErr == nil {
                qboInv.SyncToken = freshToken
                id, syncToken, err = client.UpsertInvoice(ctx, qboInv)
            }
        }
        if err != nil {
            logFail(ctx, w.QBOStore, args.CompanyID, "invoice", args.InvoiceID, args.Action, err)
            return fmt.Errorf("qbo upsert invoice: %w", err)
        }
    }

    _ = w.QBOStore.UpdateInvoiceQBO(ctx, args.InvoiceID, id, syncToken)
    logOK(ctx, w.QBOStore, args.CompanyID, "invoice", args.InvoiceID, &id, args.Action)
    return nil
}

// logFail writes a failed entry to qbo_sync_log (fire-and-forget).
func logFail(ctx context.Context, s *store.QBOStore, companyID int, entityType string, entityID int, action string, err error) {
    msg := err.Error()
    _ = s.Log(ctx, &models.QBOSyncLog{
        CompanyID: companyID, EntityType: entityType, EntityID: entityID,
        Action: action, Status: "failed", ErrorMessage: &msg,
    })
}

// logOK writes a success entry to qbo_sync_log (fire-and-forget).
func logOK(ctx context.Context, s *store.QBOStore, companyID int, entityType string, entityID int, qboID *string, action string) {
    _ = s.Log(ctx, &models.QBOSyncLog{
        CompanyID: companyID, EntityType: entityType, EntityID: entityID,
        QBOID: qboID, Action: action, Status: "success",
    })
}
```

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/worker/qbo_invoice.go
git commit -m "feat: add River worker for QBO invoice sync with SyncToken retry"
```

---

### Task 13: River Worker — Payment

**Files:**
- Create: `internal/worker/qbo_payment.go`

**Step 1: Write the worker**

```go
package worker

import (
    "context"
    "errors"
    "fmt"

    "github.com/brady1408/atlinks/internal/qbo"
    "github.com/brady1408/atlinks/internal/store"
    "github.com/jackc/pgx/v5"
    "github.com/riverqueue/river"
)

type SyncPaymentArgs struct {
    CompanyID int `json:"company_id"`
    PaymentID int `json:"payment_id"`
}

func (SyncPaymentArgs) Kind() string { return "qbo_sync_payment" }

type SyncPaymentWorker struct {
    river.WorkerDefaults[SyncPaymentArgs]
    PaymentStore       *store.PaymentStore
    PaymentDetailStore *store.PaymentDetailStore
    InvoiceStore       *store.InvoiceStore
    CustomerStore      *store.CustomerStore
    QBOStore           *store.QBOStore
    OAuthCfg           *oauth2cfg
    Sandbox            bool
}

func (w *SyncPaymentWorker) Work(ctx context.Context, job *river.Job[SyncPaymentArgs]) error {
    args := job.Args

    conn, err := w.QBOStore.GetConnection(ctx, args.CompanyID)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil
        }
        return fmt.Errorf("get qbo connection: %w", err)
    }

    pmt, err := w.PaymentStore.GetByID(ctx, args.PaymentID)
    if err != nil {
        return fmt.Errorf("load payment: %w", err)
    }
    if pmt.CustomerID == nil {
        return nil
    }

    cust, err := w.CustomerStore.GetByID(ctx, *pmt.CustomerID)
    if err != nil {
        return fmt.Errorf("load customer: %w", err)
    }
    if cust.QBOCustomerID == nil {
        return river.JobSnooze(30) // wait for customer to sync
    }

    details, err := w.PaymentDetailStore.ListByPayment(ctx, args.PaymentID)
    if err != nil {
        return fmt.Errorf("load payment details: %w", err)
    }

    // Build invoice ID → QBO invoice ID map from payment detail links
    qboInvoiceIDs := make(map[int]string)
    for _, d := range details {
        if d.InvoiceID == nil {
            continue
        }
        inv, err := w.InvoiceStore.GetByID(ctx, *d.InvoiceID)
        if err != nil || inv.QBOInvoiceID == nil {
            continue
        }
        qboInvoiceIDs[*d.InvoiceID] = *inv.QBOInvoiceID
    }

    client := qbo.NewClient(w.OAuthCfg, w.QBOStore, conn, w.Sandbox)
    qboPmt := qbo.MapPayment(*pmt, details, *cust.QBOCustomerID, qboInvoiceIDs)

    action := "create"
    if pmt.QBOPaymentID != nil {
        action = "update"
    }

    id, syncToken, err := client.UpsertPayment(ctx, qboPmt)
    if err != nil {
        logFail(ctx, w.QBOStore, args.CompanyID, "payment", args.PaymentID, action, err)
        return fmt.Errorf("qbo upsert payment: %w", err)
    }

    _ = w.QBOStore.UpdatePaymentQBO(ctx, args.PaymentID, id, syncToken)
    logOK(ctx, w.QBOStore, args.CompanyID, "payment", args.PaymentID, &id, action)
    return nil
}
```

**Step 2: Check that `PaymentDetailStore` has a `ListByPayment` method** — look in `internal/store/payment_detail_store.go`. If it only has `ListByPayment(ctx, paymentID)` already, great. If not, add it — it mirrors `ListByInvoice` in `invoice_detail_store.go`.

**Step 3: Verify build**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add internal/worker/qbo_payment.go
git commit -m "feat: add River worker for QBO payment sync"
```

---

### Task 14: Wire River into main.go

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `internal/config/config.go` (already done in Task 3)

**Step 1: Resolve the `oauth2cfg` type alias in workers**

In `internal/worker/`, create `deps.go`:

```go
package worker

import "golang.org/x/oauth2"

// oauth2cfg is the oauth2.Config type used across workers.
type oauth2cfg = oauth2.Config
```

This resolves the type reference in the worker files.

**Step 2: Add River setup to main.go**

In `initRoutes` (or a new `initRiver` function), add after stores are created:

```go
import (
    "github.com/brady1408/atlinks/internal/worker"
    "github.com/brady1408/atlinks/internal/qbo"
    "github.com/riverqueue/river"
    "github.com/riverqueue/river/riverdriver/riverpgxv5"
    "golang.org/x/oauth2"
)

func initRiver(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config,
    customerStore *store.CustomerStore,
    invoiceStore *store.InvoiceStore,
    invoiceDetailStore *store.InvoiceDetailStore,
    paymentStore *store.PaymentStore,
    paymentDetailStore *store.PaymentDetailStore,
    customerStoreRef *store.CustomerStore,
    qboStore *store.QBOStore,
) *river.Client[pgx.Tx] {

    oauthCfg := qbo.NewOAuthConfig(cfg.QBOClientID, cfg.QBOClientSecret, cfg.QBORedirectURL)

    workers := river.NewWorkers()
    river.AddWorker(workers, &worker.SyncCustomerWorker{
        CustomerStore: customerStore,
        QBOStore:      qboStore,
        OAuthCfg:      oauthCfg,
        Sandbox:       cfg.QBOSandbox,
    })
    river.AddWorker(workers, &worker.SyncInvoiceWorker{
        InvoiceStore:       invoiceStore,
        InvoiceDetailStore: invoiceDetailStore,
        CustomerStore:      customerStoreRef,
        QBOStore:           qboStore,
        OAuthCfg:           oauthCfg,
        Sandbox:            cfg.QBOSandbox,
    })
    river.AddWorker(workers, &worker.SyncPaymentWorker{
        PaymentStore:       paymentStore,
        PaymentDetailStore: paymentDetailStore,
        InvoiceStore:       invoiceStore,
        CustomerStore:      customerStoreRef,
        QBOStore:           qboStore,
        OAuthCfg:           oauthCfg,
        Sandbox:            cfg.QBOSandbox,
    })

    riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
        Workers: workers,
        Queues: map[string]river.QueueConfig{
            river.QueueDefault: {MaxWorkers: 5},
        },
    })
    if err != nil {
        log.Fatalf("river client: %v", err)
    }

    if err := riverClient.Start(ctx); err != nil {
        log.Fatalf("river start: %v", err)
    }

    return riverClient
}
```

Call `initRiver` from `main()` and pass the `riverClient` to the integration handler (Task 15) and to workers that need to enqueue jobs.

The `SyncInvoiceWorker.RiverClient` field also needs to be set here — pass `riverClient` after creating it (this requires creating the client before setting the field, which may need a two-step init or passing it via a pointer).

**Step 3: Verify build**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add cmd/server/main.go internal/worker/deps.go
git commit -m "feat: wire River client and QBO workers into server startup"
```

---

### Task 15: Enqueue Jobs from Stores

The stores need access to a River client to enqueue jobs after writes. The cleanest pattern: accept a `*river.Client[pgx.Tx]` as an optional field set after construction (to avoid circular deps).

**Files:**
- Modify: `internal/store/customer_store.go`
- Modify: `internal/store/invoice_store.go`
- Modify: `internal/store/payment_store.go`
- Modify: `internal/service/invoice_service.go`

**Step 1: Add River client field to CustomerStore**

Add to `CustomerStore` struct:
```go
RiverClient *river.Client[pgx.Tx] // set after construction; nil = no QBO sync
```

After `Create` and `Update` succeed, add:
```go
if s.RiverClient != nil {
    companyID, _ := auth.GetCompanyID(ctx)
    _, _ = s.RiverClient.Insert(ctx, worker.SyncCustomerArgs{
        CompanyID:  companyID,
        CustomerID: c.ID,
    }, nil)
}
```

Repeat the same pattern for `InvoiceStore` (enqueue `SyncInvoiceArgs{Action: "create"}` or `"update"`) and `PaymentStore` (enqueue `SyncPaymentArgs`).

**Step 2: Handle void in InvoiceService**

In `internal/service/invoice_service.go`, after marking an invoice void, enqueue:
```go
if s.RiverClient != nil {
    _, _ = s.RiverClient.Insert(ctx, worker.SyncInvoiceArgs{
        CompanyID: companyID,
        InvoiceID: id,
        Action:    "void",
    }, nil)
}
```

Add `RiverClient *river.Client[pgx.Tx]` to `InvoiceService`.

**Step 3: Set the RiverClient in main.go after initRiver returns**

```go
riverClient := initRiver(ctx, pool, cfg, ...)
customerStore.RiverClient = riverClient
invoiceStore.RiverClient = riverClient
paymentStore.RiverClient = riverClient
invoiceSvc.RiverClient = riverClient
```

**Step 4: Verify build**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add internal/store/ internal/service/invoice_service.go cmd/server/main.go
git commit -m "feat: enqueue QBO sync jobs from stores and invoice service"
```

---

### Task 16: Integrations Handler

**Files:**
- Create: `internal/handler/integrations_handler.go`

**Step 1: Write the handler**

```go
package handler

import (
    "net/http"

    "github.com/brady1408/atlinks/internal/auth"
    "github.com/brady1408/atlinks/internal/models"
    "github.com/brady1408/atlinks/internal/qbo"
    "github.com/brady1408/atlinks/internal/store"
    "github.com/brady1408/atlinks/internal/worker"
    "github.com/google/uuid"
    "github.com/riverqueue/river"
    "github.com/jackc/pgx/v5"
    "golang.org/x/oauth2"
)

type IntegrationsHandler struct {
    qboStore    *store.QBOStore
    oauthCfg    *oauth2.Config
    riverClient *river.Client[pgx.Tx]
    // stores for sync-all backfill
    customerStore *store.CustomerStore
    invoiceStore  *store.InvoiceStore
    paymentStore  *store.PaymentStore
    deps          *Deps
}

func NewIntegrationsHandler(
    qboStore *store.QBOStore,
    oauthCfg *oauth2.Config,
    riverClient *river.Client[pgx.Tx],
    customerStore *store.CustomerStore,
    invoiceStore *store.InvoiceStore,
    paymentStore *store.PaymentStore,
    deps *Deps,
) *IntegrationsHandler {
    return &IntegrationsHandler{
        qboStore:      qboStore,
        oauthCfg:      oauthCfg,
        riverClient:   riverClient,
        customerStore: customerStore,
        invoiceStore:  invoiceStore,
        paymentStore:  paymentStore,
        deps:          deps,
    }
}

func (h *IntegrationsHandler) Register(mux *http.ServeMux) {
    mux.HandleFunc("GET /settings/integrations", h.show)
    mux.HandleFunc("GET /integrations/qbo/connect", h.connect)
    mux.HandleFunc("GET /integrations/qbo/callback", h.callback)
    mux.HandleFunc("POST /integrations/qbo/disconnect", h.disconnect)
    mux.HandleFunc("POST /integrations/qbo/sync-all", h.syncAll)
}

func (h *IntegrationsHandler) show(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    companyID, _ := auth.GetCompanyID(ctx)

    conn, _ := h.qboStore.GetConnection(ctx, companyID)
    failures, _ := h.qboStore.RecentFailures(ctx, companyID)

    pc := pageContext(w, r)
    // render integrations page templ component (Task 17)
    _ = renderIntegrationsPage(w, r, pc, conn, failures)
}

func (h *IntegrationsHandler) connect(w http.ResponseWriter, r *http.Request) {
    state := uuid.New().String()
    // Store state in a short-lived cookie for CSRF validation
    http.SetCookie(w, &http.Cookie{
        Name:     "qbo_oauth_state",
        Value:    state,
        Path:     "/",
        MaxAge:   600,
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
    })
    url := h.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
    http.Redirect(w, r, url, http.StatusFound)
}

func (h *IntegrationsHandler) callback(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Validate state
    stateCookie, err := r.Cookie("qbo_oauth_state")
    if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
        http.Error(w, "invalid oauth state", http.StatusBadRequest)
        return
    }

    code := r.URL.Query().Get("code")
    realmID := r.URL.Query().Get("realmId")
    if code == "" || realmID == "" {
        http.Error(w, "missing code or realmId", http.StatusBadRequest)
        return
    }

    token, err := h.oauthCfg.Exchange(ctx, code)
    if err != nil {
        http.Error(w, "token exchange failed", http.StatusInternalServerError)
        return
    }

    user, _ := auth.GetUser(ctx)
    companyID, _ := auth.GetCompanyID(ctx)

    conn := &models.QBOConnection{
        CompanyID:    companyID,
        RealmID:      realmID,
        AccessToken:  token.AccessToken,
        RefreshToken: token.RefreshToken,
        TokenExpiry:  token.Expiry,
        ConnectedBy:  user.Username,
    }
    if err := h.qboStore.UpsertConnection(ctx, conn); err != nil {
        http.Error(w, "failed to save connection", http.StatusInternalServerError)
        return
    }

    setFlash(w, "QuickBooks connected successfully.")
    http.Redirect(w, r, "/settings/integrations", http.StatusSeeOther)
}

func (h *IntegrationsHandler) disconnect(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    companyID, _ := auth.GetCompanyID(ctx)
    _ = h.qboStore.DeleteConnection(ctx, companyID)
    setFlash(w, "QuickBooks disconnected.")
    http.Redirect(w, r, "/settings/integrations", http.StatusSeeOther)
}

func (h *IntegrationsHandler) syncAll(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    companyID, _ := auth.GetCompanyID(ctx)

    // Enqueue sync for all unsynced customers
    customers, _ := h.customerStore.ListUnsynced(ctx, companyID)
    for _, c := range customers {
        _, _ = h.riverClient.Insert(ctx, worker.SyncCustomerArgs{
            CompanyID: companyID, CustomerID: c.ID,
        }, nil)
    }

    // Enqueue sync for all unsynced invoices
    invoices, _ := h.invoiceStore.ListUnsynced(ctx, companyID)
    for _, inv := range invoices {
        _, _ = h.riverClient.Insert(ctx, worker.SyncInvoiceArgs{
            CompanyID: companyID, InvoiceID: inv.ID, Action: "create",
        }, nil)
    }

    // Enqueue sync for all unsynced payments
    payments, _ := h.paymentStore.ListUnsynced(ctx, companyID)
    for _, pmt := range payments {
        _, _ = h.riverClient.Insert(ctx, worker.SyncPaymentArgs{
            CompanyID: companyID, PaymentID: pmt.ID,
        }, nil)
    }

    setFlash(w, "Sync queued for all unsynced records.")
    http.Redirect(w, r, "/settings/integrations", http.StatusSeeOther)
}
```

**Step 2: Add `ListUnsynced` to CustomerStore, InvoiceStore, PaymentStore**

Each is a simple query:
```go
// CustomerStore
func (s *CustomerStore) ListUnsynced(ctx context.Context, companyID int) ([]models.Customer, error) {
    rows, _ := s.pool.Query(ctx,
        `SELECT `+customerColumns+` FROM customers WHERE company_id = $1 AND qbo_customer_id IS NULL AND deleted_at IS NULL`,
        companyID)
    // collect rows...
}
```

Mirror for invoices (`qbo_invoice_id IS NULL`) and payments (`qbo_payment_id IS NULL`).

**Step 3: Register the handler in main.go inside the protected mux**

```go
integrationsHandler := handler.NewIntegrationsHandler(
    qboStore, oauthCfg, riverClient,
    customerStore, invoiceStore, paymentStore, deps,
)
integrationsHandler.Register(protectedMux)
```

**Step 4: Verify build**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git add internal/handler/integrations_handler.go internal/store/ cmd/server/main.go
git commit -m "feat: add QBO integrations handler with connect/callback/disconnect/sync-all"
```

---

### Task 17: Integrations UI (Templ)

**Files:**
- Create: `internal/handler/components/integrations/integrations.templ`

**Step 1: Write the templ component**

```templ
package integrations

import (
    "github.com/brady1408/atlinks/internal/handler/components"
    "github.com/brady1408/atlinks/internal/models"
)

templ IntegrationsPage(pc components.PageContext, conn *models.QBOConnection, failures []models.QBOSyncLog) {
    @components.Layout(pc) {
        <div class="page-header">
            <h1>Integrations</h1>
        </div>

        <div class="card">
            <div class="card-header">
                <h2>QuickBooks Online</h2>
            </div>
            <div class="card-body">
                if conn == nil {
                    <p>Connect your QuickBooks Online account to automatically sync customers, invoices, and payments.</p>
                    <a href="/integrations/qbo/connect" class="btn btn-primary">Connect to QuickBooks</a>
                } else {
                    <div class="status-connected">
                        <span class="badge badge-success">Connected</span>
                        <span class="text-muted">Realm: { conn.RealmID }</span>
                        <span class="text-muted">Connected by { conn.ConnectedBy }</span>
                    </div>

                    <div class="mt-3">
                        <form method="POST" action="/integrations/qbo/sync-all"
                              onsubmit="return confirm('Queue a full sync of all unsynced records?')">
                            <button type="submit" class="btn btn-secondary">Sync All</button>
                        </form>
                        <form method="POST" action="/integrations/qbo/disconnect"
                              onsubmit="return confirm('Disconnect QuickBooks? Existing QBO records will remain.')"
                              class="d-inline">
                            <button type="submit" class="btn btn-danger">Disconnect</button>
                        </form>
                    </div>

                    if len(failures) > 0 {
                        <div class="alert alert-warning mt-3">
                            <strong>Recent sync failures:</strong>
                            <ul>
                                for _, f := range failures {
                                    <li>{ f.EntityType } #{ itoa(f.EntityID) } — { deref(f.ErrorMessage) }</li>
                                }
                            </ul>
                        </div>
                    }
                }
            </div>
        </div>
    }
}
```

Create a `helpers.go` file in the `integrations` package with `itoa` and `deref` helpers, or import from the shared components helpers.

**Step 2: Wire the templ render call in the handler**

In `integrations_handler.go`, replace `renderIntegrationsPage` placeholder:

```go
import icomponents "github.com/brady1408/atlinks/internal/handler/components/integrations"

func renderIntegrationsPage(...) {
    return h.deps.renderTempl(w, r, icomponents.IntegrationsPage(pc, conn, failures))
}
```

**Step 3: Run templ generate**

```bash
templ generate
```

**Step 4: Verify build**

```bash
go build ./...
```

**Step 5: Add QBO sync badge to invoice list**

In the invoice list templ component, add a small indicator column. When `inv.QBOInvoiceID != nil && inv.QBOSyncedAt != nil`, show a green "QBO" badge. When `qbo_invoice_id` is nil and the company has a QBO connection, show a grey pending badge.

Repeat for customer list and payment list.

**Step 6: Add nav link for Integrations**

In the nav component, add "Integrations" under Settings (or as a top-level settings submenu item).

**Step 7: Commit**

```bash
git add internal/handler/components/integrations/ internal/handler/
git commit -m "feat: add QBO integrations settings page and sync status badges"
```

---

### Task 18: Feature Gate in Handler

**Files:**
- Modify: `internal/handler/integrations_handler.go`

**Step 1: Add feature gate check to the show handler**

```go
func (h *IntegrationsHandler) show(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    fs := auth.GetFeatureSet(ctx)
    if !fs.Has(models.FeatureQBO) {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }
    // ... rest of handler
}
```

Apply the same gate to `connect`, `callback`, `disconnect`, and `syncAll`.

**Step 2: Conditionally show Integrations nav link**

In the nav templ component, wrap the Integrations link:
```templ
if pc.Features.Has(models.FeatureQBO) {
    <a href="/settings/integrations">Integrations</a>
}
```

**Step 3: Verify build and run**

```bash
go build ./...
make run
```

Visit `http://localhost:8080/settings/integrations` — should show the Integrations page.

**Step 4: Commit**

```bash
git add internal/handler/integrations_handler.go
git commit -m "feat: gate QBO integration behind FeatureQBO feature flag"
```

---

### Task 19: End-to-End Manual Test

**No code changes — verification only.**

**Step 1: Set up sandbox credentials**

Get Sandbox OAuth credentials from developer.intuit.com. Add to local `.env`:
```
QBO_CLIENT_ID=your_sandbox_id
QBO_CLIENT_SECRET=your_sandbox_secret
QBO_REDIRECT_URL=http://localhost:8080/integrations/qbo/callback
QBO_SANDBOX=true
```

**Step 2: Start the app**

```bash
make run
```

**Step 3: Walk through the connect flow**

1. Go to `http://localhost:8080/settings/integrations`
2. Click "Connect to QuickBooks" — should redirect to Intuit's sandbox login
3. Log in with a sandbox account — should redirect back to `/settings/integrations` showing "Connected"

**Step 4: Create a customer and verify sync**

1. Create a new customer in ATLinks
2. Check the `qbo_sync_log` table: `SELECT * FROM qbo_sync_log ORDER BY id DESC LIMIT 5;`
3. Log in to Intuit sandbox developer portal → QuickBooks Online sandbox → Customers — customer should appear

**Step 5: Create an invoice and verify sync**

1. Generate an invoice for the synced customer
2. Check `qbo_sync_log` — should show success
3. Check QBO sandbox — invoice should appear linked to the customer

**Step 6: Apply a payment and verify sync**

1. Apply a payment to the invoice
2. Check `qbo_sync_log` and QBO sandbox

**Step 7: Test a correction**

1. Edit the invoice (change a comment or amount)
2. Verify `qbo_sync_log` shows an "update" action with success

---

### Task 20: Deploy

**Step 1: Add production QBO credentials to NAS `.env.prod`**

SSH to NAS and edit `/volume1/docker/atlinks/.env.prod`:
```
QBO_CLIENT_ID=your_production_id
QBO_CLIENT_SECRET=your_production_secret
QBO_REDIRECT_URL=https://atlinks.app/integrations/qbo/callback
QBO_SANDBOX=false
```

Register `https://atlinks.app/integrations/qbo/callback` as a redirect URI in the Intuit production app settings.

**Step 2: Deploy**

```bash
./scripts/deploy.sh
```

**Step 3: Verify**

```bash
./scripts/deploy.sh --status
```

Visit `https://atlinks.app/settings/integrations` and test the connect flow with a production QBO account.
