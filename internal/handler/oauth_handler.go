package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/handler/components"
	"github.com/brady1408/atlinks/internal/handler/components/oauth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/store"
)

type OAuthHandler struct {
	deviceCodeStore   *store.DeviceCodeStore
	refreshTokenStore *store.RefreshTokenStore
	userStore         oauthUserStore
	jwt               *auth.JWTService
	deps              *Deps
}

type oauthUserStore interface {
	GetByID(ctx context.Context, id int) (*models.User, error)
}

func NewOAuthHandler(
	deviceCodeStore *store.DeviceCodeStore,
	refreshTokenStore *store.RefreshTokenStore,
	userStore oauthUserStore,
	jwt *auth.JWTService,
	deps *Deps,
) *OAuthHandler {
	return &OAuthHandler{
		deviceCodeStore:   deviceCodeStore,
		refreshTokenStore: refreshTokenStore,
		userStore:         userStore,
		jwt:               jwt,
		deps:              deps,
	}
}

// RegisterPublic registers endpoints that don't require authentication.
func (h *OAuthHandler) RegisterPublic(mux *http.ServeMux) {
	mux.HandleFunc("POST /oauth/device", h.deviceRequest)
	mux.HandleFunc("POST /oauth/token", h.tokenExchange)
}

// RegisterProtected registers endpoints that require the user to be logged in.
func (h *OAuthHandler) RegisterProtected(mux *http.ServeMux) {
	mux.HandleFunc("GET /oauth/device/verify", h.verifyForm)
	mux.HandleFunc("POST /oauth/device/verify", h.verifySubmit)
}

// POST /oauth/device — Client requests device_code + user_code
func (h *OAuthHandler) deviceRequest(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	if clientID == "" {
		clientID = "atlinks-mcp"
	}

	deviceCode, userCode, err := store.GenerateCodes()
	if err != nil {
		oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate codes"})
		return
	}

	dc := &store.DeviceCode{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ClientID:   clientID,
		ExpiresAt:  time.Now().Add(10 * time.Minute),
	}

	if err := h.deviceCodeStore.Create(r.Context(), dc); err != nil {
		log.Printf("oauth: create device code: %v", err)
		oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create device code"})
		return
	}

	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		scheme = "http"
	}
	verificationURI := scheme + "://" + r.Host + "/oauth/device/verify"

	oauthJSON(w, http.StatusOK, map[string]any{
		"device_code":               deviceCode,
		"user_code":                 userCode,
		"verification_uri":          verificationURI,
		"verification_uri_complete": verificationURI + "?code=" + userCode,
		"expires_in":                600,
		"interval":                  5,
	})
}

// GET /oauth/device/verify — User enters code in browser (requires login)
func (h *OAuthHandler) verifyForm(w http.ResponseWriter, r *http.Request) {
	brand := components.BrandFromHost(r.Host)
	csrfToken := ""
	if c, err := r.Cookie("csrf_token"); err == nil {
		csrfToken = c.Value
	}
	prefillCode := r.URL.Query().Get("code")
	h.deps.renderTempl(w, r, oauth.DeviceVerifyPage(brand, csrfToken, "", "", prefillCode))
}

// POST /oauth/device/verify — User approves/denies (requires login)
func (h *OAuthHandler) verifySubmit(w http.ResponseWriter, r *http.Request) {
	brand := components.BrandFromHost(r.Host)
	csrfToken := ""
	if c, err := r.Cookie("csrf_token"); err == nil {
		csrfToken = c.Value
	}

	userCode := r.FormValue("user_code")
	action := r.FormValue("action")

	dc, err := h.deviceCodeStore.GetByUserCode(r.Context(), userCode)
	if err != nil {
		h.deps.renderTempl(w, r, oauth.DeviceVerifyPage(brand, csrfToken, "Invalid or expired code. Please try again.", "", ""))
		return
	}

	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	switch action {
	case "approve":
		if err := h.deviceCodeStore.Approve(r.Context(), dc.ID, user.ID); err != nil {
			h.deps.renderTempl(w, r, oauth.DeviceVerifyPage(brand, csrfToken, "Failed to approve. Code may have expired.", "", ""))
			return
		}
		h.deps.renderTempl(w, r, oauth.DeviceVerifyPage(brand, csrfToken, "", "Device authorized successfully. You can close this page.", ""))
	case "deny":
		if err := h.deviceCodeStore.Deny(r.Context(), dc.ID); err != nil {
			h.deps.renderTempl(w, r, oauth.DeviceVerifyPage(brand, csrfToken, "Failed to deny. Code may have expired.", "", ""))
			return
		}
		h.deps.renderTempl(w, r, oauth.DeviceVerifyPage(brand, csrfToken, "", "Device authorization denied.", ""))
	default:
		h.deps.renderTempl(w, r, oauth.DeviceVerifyPage(brand, csrfToken, "Invalid action.", "", ""))
	}
}

// POST /oauth/token — Client polls for tokens or refreshes
func (h *OAuthHandler) tokenExchange(w http.ResponseWriter, r *http.Request) {
	grantType := r.FormValue("grant_type")

	switch grantType {
	case "urn:ietf:params:oauth:grant-type:device_code":
		h.handleDeviceCodeGrant(w, r)
	case "refresh_token":
		h.handleRefreshTokenGrant(w, r)
	default:
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
	}
}

func (h *OAuthHandler) handleDeviceCodeGrant(w http.ResponseWriter, r *http.Request) {
	deviceCode := r.FormValue("device_code")
	if deviceCode == "" {
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	dc, err := h.deviceCodeStore.GetByDeviceCode(r.Context(), deviceCode)
	if err != nil {
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}

	if time.Now().After(dc.ExpiresAt) {
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "expired_token"})
		return
	}

	switch dc.Status {
	case "pending":
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "authorization_pending"})
	case "denied":
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "access_denied"})
	case "approved":
		if dc.UserID == nil {
			oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
			return
		}

		user, err := h.userStore.GetByID(r.Context(), *dc.UserID)
		if err != nil {
			oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
			return
		}

		companyID := 0
		if user.CompanyID != nil {
			companyID = *user.CompanyID
		}

		// Generate 1-hour access token
		accessToken, err := h.jwt.GenerateTokenWithExpiry(user.ID, user.Username, user.Role, companyID, time.Hour)
		if err != nil {
			oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
			return
		}

		// Generate 90-day refresh token
		rawRefresh, hashRefresh, err := store.GenerateRefreshToken()
		if err != nil {
			oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
			return
		}

		rt := &store.RefreshToken{
			TokenHash: hashRefresh,
			UserID:    user.ID,
			ClientID:  dc.ClientID,
			ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
		}
		if err := h.refreshTokenStore.Create(r.Context(), rt); err != nil {
			log.Printf("oauth: create refresh token: %v", err)
			oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
			return
		}

		// Clean up the used device code
		_, _ = h.deviceCodeStore.CleanupExpired(r.Context())

		oauthJSON(w, http.StatusOK, map[string]any{
			"access_token":  accessToken,
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": rawRefresh,
		})
	default:
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
	}
}

func (h *OAuthHandler) handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	rawToken := r.FormValue("refresh_token")
	if rawToken == "" {
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}

	hash := store.HashRefreshToken(rawToken)
	rt, err := h.refreshTokenStore.GetByHash(r.Context(), hash)
	if err != nil {
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
		return
	}

	user, err := h.userStore.GetByID(r.Context(), rt.UserID)
	if err != nil {
		oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	if !user.Active {
		oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "access_denied"})
		return
	}

	companyID := 0
	if user.CompanyID != nil {
		companyID = *user.CompanyID
	}

	// Rotate: revoke old refresh token, issue new pair
	_ = h.refreshTokenStore.Revoke(r.Context(), rt.ID)

	accessToken, err := h.jwt.GenerateTokenWithExpiry(user.ID, user.Username, user.Role, companyID, time.Hour)
	if err != nil {
		oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	newRaw, newHash, err := store.GenerateRefreshToken()
	if err != nil {
		oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	newRT := &store.RefreshToken{
		TokenHash: newHash,
		UserID:    user.ID,
		ClientID:  rt.ClientID,
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}
	if err := h.refreshTokenStore.Create(r.Context(), newRT); err != nil {
		log.Printf("oauth: create rotated refresh token: %v", err)
		oauthJSON(w, http.StatusInternalServerError, map[string]string{"error": "server_error"})
		return
	}

	oauthJSON(w, http.StatusOK, map[string]any{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": newRaw,
	})
}

func oauthJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
