package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/email"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components/authpages"
	"github.com/brady1408/auto-transport-logistics/internal/middleware"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/store"
)

type authUserStore interface {
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Create(ctx context.Context, u *models.User) error
	UpdatePasswordByID(ctx context.Context, id int, hash string) error
}

type authCompanyStore interface {
	GetBySlug(ctx context.Context, slug string) (*models.Company, error)
	Create(ctx context.Context, c *models.Company) error
}

type authResetTokenStore interface {
	Create(ctx context.Context, userID int) (string, error)
	Validate(ctx context.Context, rawToken string) (userID int, tokenID int, err error)
	MarkUsed(ctx context.Context, tokenID int) error
}

type authPendingRegistrationStore interface {
	Create(ctx context.Context, reg *store.PendingRegistration) (string, error)
	Validate(ctx context.Context, rawToken string) (*store.PendingRegistration, error)
	Delete(ctx context.Context, id int) error
}

type authSubscriptionStore interface {
	Upsert(ctx context.Context, sub *models.Subscription) error
}

type AuthHandler struct {
	users             authUserStore
	companyStore      authCompanyStore
	subscriptionStore authSubscriptionStore
	inviteCode        string
	deps              *Deps
	emailSvc          *email.Service
	resetStore        authResetTokenStore
	pendingStore      authPendingRegistrationStore
	appBaseURL        string
}

func NewAuthHandler(users authUserStore, companyStore authCompanyStore, subscriptionStore authSubscriptionStore, inviteCode string, deps *Deps, emailSvc *email.Service, resetStore authResetTokenStore, pendingStore authPendingRegistrationStore, appBaseURL string) *AuthHandler {
	return &AuthHandler{
		users:             users,
		companyStore:      companyStore,
		subscriptionStore: subscriptionStore,
		inviteCode:        inviteCode,
		deps:              deps,
		emailSvc:          emailSvc,
		resetStore:        resetStore,
		pendingStore:      pendingStore,
		appBaseURL:        appBaseURL,
	}
}

func (h *AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", h.showLogin)
	mux.HandleFunc("POST /login", h.handleLogin)
	mux.HandleFunc("POST /logout", h.handleLogout)
	// Company-specific login
	mux.HandleFunc("GET /c/{slug}/login", h.showCompanyLogin)
	mux.HandleFunc("POST /c/{slug}/login", h.handleCompanyLogin)
	// Registration
	mux.HandleFunc("GET /register", h.showRegister)
	mux.HandleFunc("POST /register", h.handleRegister)
	mux.HandleFunc("GET /verify-email/{token}", h.handleVerifyEmail)
	// Password reset
	mux.HandleFunc("GET /forgot-password", h.showForgotPassword)
	mux.HandleFunc("POST /forgot-password", h.handleForgotPassword)
	mux.HandleFunc("GET /reset-password/{token}", h.showResetPassword)
	mux.HandleFunc("POST /reset-password/{token}", h.handleResetPassword)
}

func (h *AuthHandler) showLogin(w http.ResponseWriter, r *http.Request) {
	// If already logged in with a valid, complete token, redirect to dashboard
	if cookie, err := r.Cookie(middleware.CookieName); err == nil {
		if claims, err := h.deps.JWT.ValidateToken(cookie.Value); err == nil {
			if claims.CompanyID != 0 || claims.Role == "super_admin" {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
	}

	brand := components.BrandFromHost(r.Host)
	h.deps.renderTempl(w, r, authpages.LoginPage(brand, "", "", "", "", "", h.deps.getFlash(w, r), "", "", nil, nil))
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	brand := components.BrandFromHost(r.Host)

	user, err := h.users.GetByUsername(r.Context(), username)
	if err != nil || !user.Active {
		h.deps.renderTempl(w, r, authpages.LoginPage(brand, "", "", "", username, "Invalid username or password", "", "", "", nil, nil))
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		h.deps.renderTempl(w, r, authpages.LoginPage(brand, "", "", "", username, "Invalid username or password", "", "", "", nil, nil))
		return
	}

	companyID := 0
	if user.CompanyID != nil {
		companyID = *user.CompanyID
	}
	token, err := h.deps.JWT.GenerateToken(user.ID, user.Username, user.Role, companyID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.deps.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(24 * time.Hour / time.Second),
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) showCompanyLogin(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	company, err := h.companyStore.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// If already logged in with a valid, complete token, redirect to dashboard
	if cookie, err := r.Cookie(middleware.CookieName); err == nil {
		if claims, err := h.deps.JWT.ValidateToken(cookie.Value); err == nil {
			if claims.CompanyID != 0 || claims.Role == "super_admin" {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
	}

	brand := components.BrandFromHost(r.Host)
	h.deps.renderTempl(w, r, authpages.LoginPage(brand, company.CompanyName, "/c/"+slug+"/login", "", "", "", h.deps.getFlash(w, r), "", "", nil, nil))
}

func (h *AuthHandler) handleCompanyLogin(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	company, err := h.companyStore.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")
	brand := components.BrandFromHost(r.Host)

	loginAction := "/c/" + slug + "/login"

	user, err := h.users.GetByUsername(r.Context(), username)
	if err != nil || !user.Active {
		h.deps.renderTempl(w, r, authpages.LoginPage(brand, company.CompanyName, loginAction, "", username, "Invalid username or password", "", "", "", nil, nil))
		return
	}

	// Verify user belongs to this company
	if user.CompanyID == nil || *user.CompanyID != company.ID {
		h.deps.renderTempl(w, r, authpages.LoginPage(brand, company.CompanyName, loginAction, "", username, "Invalid username or password", "", "", "", nil, nil))
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, password); err != nil {
		h.deps.renderTempl(w, r, authpages.LoginPage(brand, company.CompanyName, loginAction, "", username, "Invalid username or password", "", "", "", nil, nil))
		return
	}

	token, err := h.deps.JWT.GenerateToken(user.ID, user.Username, user.Role, company.ID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.deps.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(24 * time.Hour / time.Second),
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) showRegister(w http.ResponseWriter, r *http.Request) {
	// If already logged in with a valid, complete token, redirect to dashboard
	if cookie, err := r.Cookie(middleware.CookieName); err == nil {
		if claims, err := h.deps.JWT.ValidateToken(cookie.Value); err == nil {
			if claims.CompanyID != 0 || claims.Role == "super_admin" {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
	}

	brand := components.BrandFromHost(r.Host)
	h.deps.renderTempl(w, r, authpages.LoginPage(brand, "", "", "register", "", "", "", "", "", nil, nil))
}

func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	inviteCode := strings.TrimSpace(r.FormValue("invite_code"))
	companyName := strings.TrimSpace(r.FormValue("company_name"))
	slug := strings.ToLower(strings.TrimSpace(r.FormValue("slug")))
	username := strings.TrimSpace(r.FormValue("username"))
	emailAddr := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	formData := map[string]string{
		"invite_code":  inviteCode,
		"company_name": companyName,
		"slug":         slug,
		"username":     username,
		"email":        emailAddr,
	}

	brand := components.BrandFromHost(r.Host)
	errs := make(map[string]string)

	// Validate invite code (skip for open-registration domains)
	if !brand.OpenRegistration {
		if inviteCode == "" {
			errs["invite_code"] = "Invite code is required"
		} else if inviteCode != h.inviteCode {
			errs["invite_code"] = "Invalid invite code"
		}
	}

	// Validate company name
	if companyName == "" {
		errs["company_name"] = "Company name is required"
	} else if len(companyName) > 40 {
		errs["company_name"] = "Company name must be 40 characters or less"
	}

	// Validate slug
	if slug == "" {
		errs["slug"] = "URL slug is required"
	} else if len(slug) < 3 || len(slug) > 30 {
		errs["slug"] = "Slug must be 3-30 characters"
	} else if !slugRegex.MatchString(slug) {
		errs["slug"] = "Only lowercase letters, numbers, and hyphens"
	}

	// Validate user fields
	if username == "" {
		errs["username"] = "Username is required"
	} else if len(username) > 50 {
		errs["username"] = "Username must be 50 characters or less"
	}
	if emailAddr == "" {
		errs["email"] = "Email is required for password recovery"
	} else if len(emailAddr) > 255 {
		errs["email"] = "Email must be 255 characters or less"
	}
	if password == "" {
		errs["password"] = "Password is required"
	}

	// Pre-check uniqueness
	if slug != "" && errs["slug"] == "" {
		if _, err := h.companyStore.GetBySlug(r.Context(), slug); err == nil {
			errs["slug"] = "This slug is already taken"
		}
	}
	if username != "" && errs["username"] == "" {
		if _, err := h.users.GetByUsername(r.Context(), username); err == nil {
			errs["username"] = "This username is already taken"
		}
	}

	if len(errs) > 0 {
		h.deps.renderTempl(w, r, authpages.LoginPage(brand, "", "", "register", "", "", "", "", "", formData, errs))
		return
	}

	// Hash password before storing in pending table
	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store as pending registration and send verification email
	reg := &store.PendingRegistration{
		CompanyName:  companyName,
		Slug:         slug,
		Username:     username,
		Email:        emailAddr,
		PasswordHash: hash,
	}
	token, err := h.pendingStore.Create(r.Context(), reg)
	if err != nil {
		h.deps.renderTempl(w, r, authpages.LoginPage(brand, "", "", "register", "", "", "", "", "Failed to start registration. Please try again.", formData, nil))
		return
	}

	verifyURL := h.appBaseURL + "/verify-email/" + token
	if err := h.emailSvc.SendVerification(emailAddr, companyName, verifyURL); err != nil {
		log.Printf("ERROR: send verification email to %s: %v", emailAddr, err)
		h.deps.renderTempl(w, r, authpages.LoginPage(brand, "", "", "register", "", "", "", "", "Failed to send verification email. Please try again.", formData, nil))
		return
	}

	h.deps.renderTempl(w, r, authpages.LoginPage(brand, "", "", "register", "", "", "", fmt.Sprintf("Verification email sent to %s. Please check your inbox to complete registration.", emailAddr), "", nil, nil))
}

func (h *AuthHandler) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	reg, err := h.pendingStore.Validate(r.Context(), token)
	if err != nil {
		h.deps.renderTempl(w, r, authpages.ForgotPasswordPage("This verification link is invalid or has expired. Please register again.", "", ""))
		return
	}

	// Re-check uniqueness (slug/username could have been taken while pending)
	if _, err := h.companyStore.GetBySlug(r.Context(), reg.Slug); err == nil {
		h.deps.renderTempl(w, r, authpages.ForgotPasswordPage("The company URL slug has already been taken. Please register again with a different slug.", "", ""))
		return
	}
	if _, err := h.users.GetByUsername(r.Context(), reg.Username); err == nil {
		h.deps.renderTempl(w, r, authpages.ForgotPasswordPage("The username has already been taken. Please register again with a different username.", "", ""))
		return
	}

	// Create company
	company := &models.Company{
		CompanyName: reg.CompanyName,
		Slug:        reg.Slug,
		Active:      true,
	}
	if err := h.companyStore.Create(r.Context(), company); err != nil {
		log.Printf("ERROR: create company from verification: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create admin user
	user := &models.User{
		Username:     reg.Username,
		Email:        reg.Email,
		PasswordHash: reg.PasswordHash,
		Role:         "company_admin",
		Active:       true,
		CompanyID:    &company.ID,
	}
	if err := h.users.Create(r.Context(), user); err != nil {
		log.Printf("ERROR: create user from verification: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create default Basic subscription for new company
	sub := &models.Subscription{
		CompanyID: company.ID,
		Tier:      models.TierBasic,
		Status:    models.StatusActive,
		AddonEDI:  false,
	}
	if err := h.subscriptionStore.Upsert(r.Context(), sub); err != nil {
		log.Printf("ERROR: create subscription for company %d: %v", company.ID, err)
	}

	// Clean up pending registration
	if err := h.pendingStore.Delete(r.Context(), reg.ID); err != nil {
		log.Printf("ERROR: delete pending registration %d: %v", reg.ID, err)
	}

	h.deps.setFlash(w, "Email verified! Your company has been created. Please sign in.")
	http.Redirect(w, r, "/c/"+reg.Slug+"/login", http.StatusSeeOther)
}

func (h *AuthHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.deps.SecureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// --- Password Reset ---

func (h *AuthHandler) showForgotPassword(w http.ResponseWriter, r *http.Request) {
	h.deps.renderTempl(w, r, authpages.ForgotPasswordPage("", "", ""))
}

func (h *AuthHandler) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))

	// Always show the same message regardless of whether the user exists or has email
	genericMsg := "If an account exists with that username and has an email on file, we've sent a password reset link."

	if username == "" {
		h.deps.renderTempl(w, r, authpages.ForgotPasswordPage("Please enter your username", "", ""))
		return
	}

	// Look up user — silently skip if not found
	user, err := h.users.GetByUsername(r.Context(), username)
	if err == nil && user.Active && user.Email != "" {
		// User found with email — create token and send
		token, err := h.resetStore.Create(r.Context(), user.ID)
		if err != nil {
			log.Printf("ERROR: create reset token for user %d: %v", user.ID, err)
		} else {
			resetURL := h.appBaseURL + "/reset-password/" + token
			if err := h.emailSvc.SendPasswordReset(user.Email, user.Username, resetURL); err != nil {
				log.Printf("ERROR: send reset email to %s: %v", user.Email, err)
			}
		}
	} else if err == nil && user.Active && user.Email == "" {
		// User found but no email — log for debugging, show same generic message
		log.Printf("Password reset requested for user %q but no email on file", username)
	}

	h.deps.renderTempl(w, r, authpages.ForgotPasswordPage("", genericMsg, username))
}

func (h *AuthHandler) showResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	_, _, err := h.resetStore.Validate(r.Context(), token)
	if err != nil {
		h.deps.renderTempl(w, r, authpages.ResetPasswordPage("", "This reset link is invalid or has expired. Please request a new one.", true))
		return
	}

	h.deps.renderTempl(w, r, authpages.ResetPasswordPage(token, "", false))
}

func (h *AuthHandler) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if password == "" {
		h.deps.renderTempl(w, r, authpages.ResetPasswordPage(token, "Password is required", false))
		return
	}

	if password != confirm {
		h.deps.renderTempl(w, r, authpages.ResetPasswordPage(token, "Passwords do not match", false))
		return
	}

	userID, tokenID, err := h.resetStore.Validate(r.Context(), token)
	if err != nil {
		h.deps.renderTempl(w, r, authpages.ResetPasswordPage("", "This reset link is invalid or has expired. Please request a new one.", true))
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.users.UpdatePasswordByID(r.Context(), userID, hash); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := h.resetStore.MarkUsed(r.Context(), tokenID); err != nil {
		log.Printf("ERROR: mark token %d used: %v", tokenID, err)
	}

	h.deps.setFlash(w, "Password updated successfully. Please sign in with your new password.")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
