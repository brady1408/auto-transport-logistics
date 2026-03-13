package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/integrations"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/riverargs"
	"github.com/brady1408/auto-transport-logistics/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"golang.org/x/oauth2"
)

// IntegrationsHandler handles the QBO integrations settings page and OAuth flow.
type IntegrationsHandler struct {
	qboStore      *store.QBOStore
	oauthCfg      *oauth2.Config
	riverClient   *river.Client[pgx.Tx]
	customerStore *store.CustomerStore
	invoiceStore  *store.InvoiceStore
	paymentStore  *store.PaymentStore
	deps          *Deps
}

// NewIntegrationsHandler constructs an IntegrationsHandler.
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

// Register wires routes onto the given mux.
// mw is applied to the mutating routes (connect/callback/disconnect/sync-all);
// pass middleware.RequireRole("company_admin", "super_admin") from the caller.
func (h *IntegrationsHandler) Register(mux *http.ServeMux, mw func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return mw(fn) }
	mux.HandleFunc("GET /settings/integrations", h.show)
	mux.Handle("GET /integrations/qbo/connect", wrap(h.connect))
	mux.Handle("GET /integrations/qbo/callback", wrap(h.callback))
	mux.Handle("POST /integrations/qbo/disconnect", wrap(h.disconnect))
	mux.Handle("POST /integrations/qbo/sync-all", wrap(h.syncAll))
}

// show renders the integrations settings page.
func (h *IntegrationsHandler) show(w http.ResponseWriter, r *http.Request) {
	if !h.deps.GetFeatures(r).Has(models.FeatureQBO) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	companyID, err := auth.GetCompanyID(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	conn, err := h.qboStore.GetConnection(r.Context(), companyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isNoRowsError(err) {
			conn = nil
		} else {
			log.Printf("integrations show: get connection: %v", err)
			conn = nil
		}
	}

	failures, err := h.qboStore.RecentFailures(r.Context(), companyID)
	if err != nil {
		log.Printf("integrations show: recent failures: %v", err)
		failures = nil
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, integrations.Page(pg, conn, failures))
}

// connect initiates the OAuth2 flow by redirecting to QBO.
func (h *IntegrationsHandler) connect(w http.ResponseWriter, r *http.Request) {
	if !h.deps.GetFeatures(r).Has(models.FeatureQBO) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	state := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     "qbo_oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   h.deps.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
	authURL := h.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// callback handles the OAuth2 redirect from QBO.
func (h *IntegrationsHandler) callback(w http.ResponseWriter, r *http.Request) {
	if !h.deps.GetFeatures(r).Has(models.FeatureQBO) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	// Validate state cookie
	stateCookie, err := r.Cookie("qbo_oauth_state")
	if err != nil {
		http.Error(w, "Missing OAuth state cookie", http.StatusBadRequest)
		return
	}
	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "qbo_oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.deps.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "Invalid OAuth state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing OAuth code", http.StatusBadRequest)
		return
	}

	token, err := h.oauthCfg.Exchange(r.Context(), code)
	if err != nil {
		log.Printf("integrations callback: token exchange: %v", err)
		h.deps.setFlash(w, "Failed to connect QuickBooks: token exchange error")
		redirect(w, r, "/settings/integrations")
		return
	}

	realmID := r.URL.Query().Get("realmId")

	companyID, err := auth.GetCompanyID(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	user, _ := auth.GetUser(r.Context())

	conn := &models.QBOConnection{
		CompanyID:    companyID,
		RealmID:      realmID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry,
		ConnectedBy:  user.Username,
	}

	if err := h.qboStore.UpsertConnection(r.Context(), conn); err != nil {
		log.Printf("integrations callback: upsert connection: %v", err)
		h.deps.setFlash(w, "Failed to save QuickBooks connection")
		redirect(w, r, "/settings/integrations")
		return
	}

	h.deps.setFlash(w, "QuickBooks connected successfully")
	redirect(w, r, "/settings/integrations")
}

// disconnect removes the QBO connection for the current company.
func (h *IntegrationsHandler) disconnect(w http.ResponseWriter, r *http.Request) {
	if !h.deps.GetFeatures(r).Has(models.FeatureQBO) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	companyID, err := auth.GetCompanyID(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	if err := h.qboStore.DeleteConnection(r.Context(), companyID); err != nil {
		log.Printf("integrations disconnect: %v", err)
		h.deps.setFlash(w, "Failed to disconnect QuickBooks")
		redirect(w, r, "/settings/integrations")
		return
	}

	h.deps.setFlash(w, "QuickBooks disconnected")
	redirect(w, r, "/settings/integrations")
}

// syncAll enqueues River jobs for all unsynced customers, invoices, and payments.
func (h *IntegrationsHandler) syncAll(w http.ResponseWriter, r *http.Request) {
	if !h.deps.GetFeatures(r).Has(models.FeatureQBO) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	companyID, err := auth.GetCompanyID(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	ctx := r.Context()
	total := 0

	// Enqueue unsynced customers
	customers, err := h.customerStore.ListUnsynced(ctx, companyID)
	if err != nil {
		log.Printf("integrations sync-all: list unsynced customers: %v", err)
	} else {
		for _, c := range customers {
			args := riverargs.SyncCustomerArgs{CompanyID: companyID, CustomerID: c.ID}
			if _, err := h.riverClient.Insert(ctx, args, nil); err != nil {
				log.Printf("integrations sync-all: enqueue customer %d: %v", c.ID, err)
			} else {
				total++
			}
		}
	}

	// Enqueue unsynced invoices
	invoices, err := h.invoiceStore.ListUnsynced(ctx, companyID)
	if err != nil {
		log.Printf("integrations sync-all: list unsynced invoices: %v", err)
	} else {
		for _, inv := range invoices {
			args := riverargs.SyncInvoiceArgs{CompanyID: companyID, InvoiceID: inv.ID, Action: "create"}
			if _, err := h.riverClient.Insert(ctx, args, nil); err != nil {
				log.Printf("integrations sync-all: enqueue invoice %d: %v", inv.ID, err)
			} else {
				total++
			}
		}
	}

	// Enqueue unsynced payments
	payments, err := h.paymentStore.ListUnsynced(ctx, companyID)
	if err != nil {
		log.Printf("integrations sync-all: list unsynced payments: %v", err)
	} else {
		for _, p := range payments {
			args := riverargs.SyncPaymentArgs{CompanyID: companyID, PaymentID: p.ID, Action: "create"}
			if _, err := h.riverClient.Insert(ctx, args, nil); err != nil {
				log.Printf("integrations sync-all: enqueue payment %d: %v", p.ID, err)
			} else {
				total++
			}
		}
	}

	h.deps.setFlash(w, fmt.Sprintf("Queued %d items for sync", total))
	redirect(w, r, "/settings/integrations")
}

// isNoRowsError returns true if the error wraps pgx.ErrNoRows.
func isNoRowsError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
