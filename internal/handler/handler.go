package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/a-h/templ"

	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/handler/components"
	"github.com/brady1408/atlinks/internal/models"
)

// depsCompanyStore is the subset of CompanyStore used by Deps for company name lookups.
type depsCompanyStore interface {
	Get(ctx context.Context) (*models.Company, error)
}

// depsSubscriptionStore is the subset of SubscriptionStore used by Deps for feature lookups.
type depsSubscriptionStore interface {
	GetByCompanyID(ctx context.Context, companyID int) (*models.Subscription, error)
}

type Deps struct {
	JWT               *auth.JWTService
	Audit             *audit.Service
	CompanyStore      depsCompanyStore
	SubscriptionStore depsSubscriptionStore
	BuildVersion      string
	SecureCookies     bool
	companyNames  sync.Map // cache: companyID(int) -> companyName(string)
	subscriptions sync.Map // cache: companyID(int) -> *models.Subscription
}

func parseID(r *http.Request) (int, error) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %s", idStr)
	}
	return id, nil
}

func parsePathID(r *http.Request, param string) (int, error) {
	idStr := r.PathValue(param)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %s", param, idStr)
	}
	return id, nil
}

func formDate(r *http.Request, key string) *time.Time {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return nil
	}
	return &t
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (d *Deps) getCompanyName(ctx context.Context) string {
	user, ok := auth.GetUser(ctx)
	if !ok || user.CompanyID == 0 || d.CompanyStore == nil {
		return ""
	}
	if name, ok := d.companyNames.Load(user.CompanyID); ok {
		return name.(string)
	}
	c, err := d.CompanyStore.Get(ctx)
	if err != nil {
		return ""
	}
	d.companyNames.Store(user.CompanyID, c.CompanyName)
	return c.CompanyName
}

// InvalidateCompanyName removes a cached company name so it will be re-fetched.
func (d *Deps) InvalidateCompanyName(companyID int) {
	d.companyNames.Delete(companyID)
}

// getSub returns the cached Subscription for the current request's company.
// Returns nil if there is no subscription row or the company is unknown.
func (d *Deps) getSub(ctx context.Context) *models.Subscription {
	user, ok := auth.GetUser(ctx)
	if !ok || user.CompanyID == 0 || d.SubscriptionStore == nil {
		return nil
	}
	if cached, ok := d.subscriptions.Load(user.CompanyID); ok {
		return cached.(*models.Subscription)
	}
	sub, err := d.SubscriptionStore.GetByCompanyID(ctx, user.CompanyID)
	if err != nil {
		sub = nil
	}
	d.subscriptions.Store(user.CompanyID, sub)
	return sub
}

// getFeatures returns the FeatureSet for the current request's company.
// Super_admin gets all features. Results are cached per company.
func (d *Deps) getFeatures(ctx context.Context) models.FeatureSet {
	user, ok := auth.GetUser(ctx)
	if !ok {
		return models.BuildFeatureSet(nil)
	}
	// super_admin always gets everything
	if user.Role == "super_admin" {
		fs := make(models.FeatureSet)
		for _, f := range models.TierFeatures[models.TierEnterprise] {
			fs[f] = true
		}
		fs[models.FeatureEDI] = true
		return fs
	}
	return models.BuildFeatureSet(d.getSub(ctx))
}

// IsSuspended returns true if the current request's company has a suspended subscription.
// Super_admin is never suspended.
func (d *Deps) IsSuspended(r *http.Request) bool {
	user, ok := auth.GetUserFromRequest(r)
	if !ok || user.Role == "super_admin" {
		return false
	}
	sub := d.getSub(r.Context())
	return sub != nil && sub.Status == models.StatusSuspended
}

// InvalidateSubscription clears the cached Subscription for a company.
func (d *Deps) InvalidateSubscription(companyID int) {
	d.subscriptions.Delete(companyID)
}

// GetFeatures is the public entry point for feature checks, used from main.go middleware wiring.
func (d *Deps) GetFeatures(r *http.Request) models.FeatureSet {
	return d.getFeatures(r.Context())
}

// SuspendedBlockHandler is called by the read-only middleware when a write is attempted on a
// suspended account. HTMX requests get an HX-Redirect; others get a normal redirect.
func (d *Deps) SuspendedBlockHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/suspended")
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Redirect(w, r, "/suspended", http.StatusSeeOther)
	}
}

// SuspendedPageHandler renders the account-suspended info page.
func (d *Deps) SuspendedPageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pg := d.pageContext(w, r)
		d.renderTempl(w, r, components.SuspendedPage(pg))
	}
}

// UpgradeHandler returns an http.HandlerFunc that renders the upgrade page for a gated feature.
func (d *Deps) UpgradeHandler(feature models.Feature) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pg := d.pageContext(w, r)
		w.WriteHeader(http.StatusForbidden)
		d.renderTempl(w, r, components.UpgradePage(pg, string(feature)))
	}
}

// pageContext builds a PageContext from the current request.
func (d *Deps) pageContext(w http.ResponseWriter, r *http.Request) components.PageContext {
	ctx := components.PageContext{}
	if user, ok := auth.GetUserFromRequest(r); ok {
		ctx.User = &user
	}
	if companyName := d.getCompanyName(r.Context()); companyName != "" {
		ctx.CompanyName = companyName
	}
	if flash := d.getFlash(w, r); flash != "" {
		ctx.Flash = flash
	}
	if cookie, err := r.Cookie("csrf_token"); err == nil {
		ctx.CSRFToken = cookie.Value
	}
	ctx.Features = d.getFeatures(r.Context())
	ctx.Suspended = d.IsSuspended(r)
	ctx.Brand = components.BrandFromHost(r.Host)
	return ctx
}

// renderTempl renders a templ.Component with proper Content-Type header.
func (d *Deps) renderTempl(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(r.Context(), w); err != nil {
		log.Printf("template render error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// redirect sends an HTMX-aware redirect.
func redirect(w http.ResponseWriter, r *http.Request, url string) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// serverError logs the full error server-side and returns a generic message to the client.
func serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

func (d *Deps) setFlash(w http.ResponseWriter, msg string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    url.QueryEscape(msg),
		Path:     "/",
		MaxAge:   5,
		HttpOnly: true,
		Secure:   d.SecureCookies,
	})
}

func (d *Deps) getFlash(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("flash")
	if err != nil {
		return ""
	}
	// Clear the flash cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   d.SecureCookies,
	})
	val, err := url.QueryUnescape(cookie.Value)
	if err != nil {
		return cookie.Value
	}
	return val
}

// Form helpers
func formString(r *http.Request, key string) *string {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return nil
	}
	return &v
}

func formStringRequired(r *http.Request, key string) string {
	return strings.TrimSpace(r.FormValue(key))
}

func formInt(r *http.Request, key string) *int {
	v := strings.TrimSpace(r.FormValue(key))
	if v == "" {
		return nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &i
}

func formBool(r *http.Request, key string) bool {
	v := r.FormValue(key)
	return v == "on" || v == "true" || v == "1"
}

// writeCSV writes a CSV file to the response.
func writeCSV(w http.ResponseWriter, filename string, headers []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	writer := csv.NewWriter(w)
	if err := writer.Write(headers); err != nil {
		log.Printf("csv write headers: %v", err)
		return
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			log.Printf("csv write row: %v", err)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Printf("csv flush: %v", err)
	}
}
