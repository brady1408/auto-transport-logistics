package qbo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/store"
	"golang.org/x/oauth2"
)

const (
	productionBase = "https://quickbooks.api.intuit.com/v3/company"
	sandboxBase    = "https://sandbox-quickbooks.api.intuit.com/v3/company"
	minorVersion   = "?minorversion=65"
)

// Client calls the QBO REST API for a specific company connection.
type Client struct {
	oauthCfg   *oauth2.Config
	qboStore   *store.QBOStore
	conn       *models.QBOConnection
	sandbox    bool
	httpClient *http.Client
}

// NewClient returns a Client ready to make QBO API calls for the given connection.
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
	if err := c.qboStore.UpdateTokens(ctx, c.conn.CompanyID,
		newToken.AccessToken, newToken.RefreshToken, newToken.Expiry); err != nil {
		return fmt.Errorf("save refreshed token: %w", err)
	}
	c.conn.AccessToken = newToken.AccessToken
	c.conn.RefreshToken = newToken.RefreshToken
	c.conn.TokenExpiry = newToken.Expiry
	return nil
}

// do performs an authenticated JSON request, returning the response body bytes and status code.
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

	url := c.baseURL() + "/" + c.conn.RealmID + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
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
	b, status, err := c.do(ctx, http.MethodPost, "/customer"+minorVersion, cust)
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

// isStaleTokenError returns true if the QBO error response indicates a stale SyncToken (fault code 6140).
// QBO returns HTTP 400 (not 409) with fault code "6140" for stale SyncToken conflicts.
func isStaleTokenError(body []byte) bool {
	var fault struct {
		Fault struct {
			Error []struct {
				Code string `json:"code"`
			} `json:"Error"`
		} `json:"Fault"`
	}
	if err := json.Unmarshal(body, &fault); err != nil {
		return false
	}
	for _, e := range fault.Fault.Error {
		if e.Code == "6140" {
			return true
		}
	}
	return false
}

// UpsertInvoice creates or updates a QBO Invoice. Returns QBO Invoice ID and SyncToken.
func (c *Client) UpsertInvoice(ctx context.Context, inv Invoice) (id, syncToken string, err error) {
	b, status, reqErr := c.do(ctx, http.MethodPost, "/invoice"+minorVersion, inv)
	if reqErr != nil {
		return "", "", reqErr
	}
	if status == http.StatusBadRequest && isStaleTokenError(b) {
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

// GetInvoiceSyncToken fetches the current SyncToken for a QBO invoice.
// Used to recover from a stale-token conflict (fault code 6140).
func (c *Client) GetInvoiceSyncToken(ctx context.Context, qboID string) (string, error) {
	b, status, err := c.do(ctx, http.MethodGet, "/invoice/"+qboID, nil)
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
	b, status, reqErr := c.do(ctx, http.MethodPost, "/payment"+minorVersion, pmt)
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

// SyncTokenError is returned when QBO responds with fault code 6140 (stale SyncToken).
type SyncTokenError struct {
	EntityID string
}

func (e *SyncTokenError) Error() string {
	return "qbo sync token conflict for entity " + e.EntityID
}
