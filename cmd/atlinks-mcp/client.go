package main

import (
	"fmt"
	"net/http"
	"sync"

	"connectrpc.com/connect"
	"github.com/brady1408/auto-transport-logistics/internal/gen/atlinks/v1/atlinkspbconnect"
)

// atlClient wraps Connect-RPC clients with auto-refresh.
type atlClient struct {
	mu        sync.Mutex
	tokens    *tokenFile
	serverURL string

	customers    atlinkspbconnect.CustomerServiceClient
	orders       atlinkspbconnect.OrderServiceClient
	vehicles     atlinkspbconnect.VehicleServiceClient
	feedback     atlinkspbconnect.FeedbackServiceClient
	employees    atlinkspbconnect.EmployeeServiceClient
	trucks       atlinkspbconnect.TruckServiceClient
	vendors      atlinkspbconnect.VendorServiceClient
	zones        atlinkspbconnect.ZoneServiceClient
	charges      atlinkspbconnect.ChargeServiceClient
	damages      atlinkspbconnect.DamageServiceClient
	damageClaims atlinkspbconnect.DamageClaimServiceClient
	creditMemos  atlinkspbconnect.CreditMemoServiceClient
	trips        atlinkspbconnect.TripServiceClient
	invoices     atlinkspbconnect.InvoiceServiceClient
	payments     atlinkspbconnect.PaymentServiceClient
	ap           atlinkspbconnect.APServiceClient
	earnings     atlinkspbconnect.EarningsServiceClient
	lookups      atlinkspbconnect.LookupServiceClient
}

func newAtlClient(serverURL string, tokens *tokenFile) *atlClient {
	c := &atlClient{
		tokens:    tokens,
		serverURL: serverURL,
	}
	c.rebuildClients()
	return c
}

func (c *atlClient) rebuildClients() {
	httpClient := &http.Client{Transport: &authTransport{client: c}}
	c.customers = atlinkspbconnect.NewCustomerServiceClient(httpClient, c.serverURL)
	c.orders = atlinkspbconnect.NewOrderServiceClient(httpClient, c.serverURL)
	c.vehicles = atlinkspbconnect.NewVehicleServiceClient(httpClient, c.serverURL)
	c.feedback = atlinkspbconnect.NewFeedbackServiceClient(httpClient, c.serverURL)
	c.employees = atlinkspbconnect.NewEmployeeServiceClient(httpClient, c.serverURL)
	c.trucks = atlinkspbconnect.NewTruckServiceClient(httpClient, c.serverURL)
	c.vendors = atlinkspbconnect.NewVendorServiceClient(httpClient, c.serverURL)
	c.zones = atlinkspbconnect.NewZoneServiceClient(httpClient, c.serverURL)
	c.charges = atlinkspbconnect.NewChargeServiceClient(httpClient, c.serverURL)
	c.damages = atlinkspbconnect.NewDamageServiceClient(httpClient, c.serverURL)
	c.damageClaims = atlinkspbconnect.NewDamageClaimServiceClient(httpClient, c.serverURL)
	c.creditMemos = atlinkspbconnect.NewCreditMemoServiceClient(httpClient, c.serverURL)
	c.trips = atlinkspbconnect.NewTripServiceClient(httpClient, c.serverURL)
	c.invoices = atlinkspbconnect.NewInvoiceServiceClient(httpClient, c.serverURL)
	c.payments = atlinkspbconnect.NewPaymentServiceClient(httpClient, c.serverURL)
	c.ap = atlinkspbconnect.NewAPServiceClient(httpClient, c.serverURL)
	c.earnings = atlinkspbconnect.NewEarningsServiceClient(httpClient, c.serverURL)
	c.lookups = atlinkspbconnect.NewLookupServiceClient(httpClient, c.serverURL)
}

// getAccessToken returns a valid access token, refreshing if needed.
func (c *atlClient) getAccessToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokens == nil || c.tokens.AccessToken == "" {
		return "", fmt.Errorf("not authenticated — run device code flow")
	}

	if !isTokenExpired(c.tokens.AccessToken) {
		return c.tokens.AccessToken, nil
	}

	// Try to refresh
	if c.tokens.RefreshToken == "" {
		return "", fmt.Errorf("access token expired and no refresh token available")
	}

	newAccess, newRefresh, err := refreshAccessToken(c.serverURL, c.tokens.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("token refresh failed: %w", err)
	}

	c.tokens.AccessToken = newAccess
	if newRefresh != "" {
		c.tokens.RefreshToken = newRefresh
	}
	_ = saveTokens(c.tokens)

	return c.tokens.AccessToken, nil
}

// authTransport injects Bearer token into Connect-RPC requests.
type authTransport struct {
	client *atlClient
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.client.getAccessToken()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return http.DefaultTransport.RoundTrip(req)
}

// connectErr extracts a user-friendly error message from a Connect error.
func connectErr(err error) string {
	if err == nil {
		return ""
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		return err.Error()
	}
	msg := ce.Message()
	switch ce.Code() {
	case connect.CodeNotFound:
		return "not found: " + msg
	case connect.CodeAlreadyExists:
		return "already exists: " + msg
	case connect.CodeInvalidArgument:
		return "invalid input: " + msg
	case connect.CodePermissionDenied:
		return "permission denied: " + msg
	case connect.CodeUnauthenticated:
		return "not authenticated — try re-running 'mcp login'"
	case connect.CodeFailedPrecondition:
		return "cannot perform this action: " + msg
	case connect.CodeAborted:
		return "conflict (record was modified by another user): " + msg
	case connect.CodeResourceExhausted:
		return "rate limit exceeded — try again shortly"
	case connect.CodeUnavailable:
		return "server unavailable — check your connection or try again"
	case connect.CodeDeadlineExceeded:
		return "request timed out — try again"
	default:
		return msg
	}
}
