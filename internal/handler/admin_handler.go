package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/handler/components/admin"
	"github.com/brady1408/atlinks/internal/handler/components/settings"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/riverargs"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

var digitsOnly = regexp.MustCompile(`\D`)
var slugRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// stripPhone removes all non-digit characters from a phone string.
func stripPhone(s *string) *string {
	if s == nil {
		return nil
	}
	v := digitsOnly.ReplaceAllString(*s, "")
	if v == "" {
		return nil
	}
	return &v
}

// validateCompany checks field lengths against DB constraints and returns per-field errors.
func validateCompany(c *models.Company) map[string]string {
	errs := make(map[string]string)
	if c.CompanyName == "" {
		errs["company_name"] = "Company name is required"
	} else if len(c.CompanyName) > 40 {
		errs["company_name"] = "Company name must be 40 characters or less"
	}
	if c.Slug == "" {
		errs["slug"] = "Slug is required"
	} else if len(c.Slug) < 3 || len(c.Slug) > 30 {
		errs["slug"] = "Slug must be 3-30 characters"
	} else if !slugRegex.MatchString(c.Slug) {
		errs["slug"] = "Only lowercase letters, numbers, and hyphens allowed"
	}
	if s := derefStr(c.Address); len(s) > 30 {
		errs["address"] = "Address must be 30 characters or less"
	}
	if s := derefStr(c.Address2); len(s) > 30 {
		errs["address2"] = "Address 2 must be 30 characters or less"
	}
	if s := derefStr(c.City); len(s) > 25 {
		errs["city"] = "City must be 25 characters or less"
	}
	if s := derefStr(c.State); len(s) > 2 {
		errs["state"] = "State must be a 2-letter abbreviation"
	}
	if s := derefStr(c.Zip); len(s) > 10 {
		errs["zip"] = "Zip must be 10 characters or less"
	}
	if s := derefStr(c.Phone); len(s) > 10 {
		errs["phone"] = "Phone must be 10 digits or less"
	}
	if s := derefStr(c.Fax); len(s) > 10 {
		errs["fax"] = "Fax must be 10 digits or less"
	}
	if s := derefStr(c.SCAC); len(s) > 4 {
		errs["scac"] = "SCAC must be 4 characters or less"
	}
	if s := derefStr(c.FederalID); len(s) > 15 {
		errs["federal_id"] = "Federal ID must be 15 characters or less"
	}
	if s := derefStr(c.MCNumber); len(s) > 15 {
		errs["mc_number"] = "MC Number must be 15 characters or less"
	}
	if s := derefStr(c.DOTNumber); len(s) > 15 {
		errs["dot_number"] = "DOT Number must be 15 characters or less"
	}
	if s := derefStr(c.SPLC); len(s) > 10 {
		errs["splc"] = "SPLC must be 10 characters or less"
	}
	return errs
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// uppercaseState normalizes state to uppercase.
func uppercaseState(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.ToUpper(strings.TrimSpace(*s))
	if v == "" {
		return nil
	}
	return &v
}

type adminCompanyStore interface {
	ListAll(ctx context.Context) ([]models.Company, error)
	GetByID(ctx context.Context, id int) (*models.Company, error)
	Create(ctx context.Context, c *models.Company) error
	UpdateByID(ctx context.Context, c *models.Company) error
}

type adminUserStore interface {
	ListByCompany(ctx context.Context, companyID int) ([]models.User, error)
	GetByID(ctx context.Context, id int) (*models.User, error)
	Create(ctx context.Context, u *models.User) error
	Update(ctx context.Context, u *models.User) error
	UpdatePassword(ctx context.Context, id int, companyID int, hash string) error
	UpdatePasswordByID(ctx context.Context, id int, hash string) error
}

type adminSubscriptionStore interface {
	GetByCompanyID(ctx context.Context, companyID int) (*models.Subscription, error)
	Upsert(ctx context.Context, sub *models.Subscription) error
	ListAll(ctx context.Context) ([]models.Subscription, error)
}

type adminTruckStore interface {
	ListAll(ctx context.Context) ([]models.Truck, error)
}

type adminApiKeyStore interface {
	List(ctx context.Context) ([]models.ApiKey, error)
	Create(ctx context.Context, userID int, label, keyHash string) (*models.ApiKey, error)
	Revoke(ctx context.Context, id int) error
}

type AdminHandler struct {
	companyStore      adminCompanyStore
	userStore         adminUserStore
	subscriptionStore adminSubscriptionStore
	migrationRunStore *store.MigrationRunStore
	truckStore        adminTruckStore
	apiKeyStore       adminApiKeyStore
	river             *river.Client[pgx.Tx]
	migrationsDir     string
	deps              *Deps
}

func NewAdminHandler(
	companyStore adminCompanyStore,
	userStore adminUserStore,
	subscriptionStore adminSubscriptionStore,
	migrationRunStore *store.MigrationRunStore,
	truckStore adminTruckStore,
	apiKeyStore adminApiKeyStore,
	riverClient *river.Client[pgx.Tx],
	migrationsDir string,
	deps *Deps,
) *AdminHandler {
	return &AdminHandler{
		companyStore:      companyStore,
		userStore:         userStore,
		subscriptionStore: subscriptionStore,
		migrationRunStore: migrationRunStore,
		truckStore:        truckStore,
		apiKeyStore:       apiKeyStore,
		river:             riverClient,
		migrationsDir:     migrationsDir,
		deps:              deps,
	}
}

// enqueueMigration checks for a .bak file upload, saves it, creates a migration run,
// and enqueues a River job. Returns the run ID (or 0 if no file was uploaded).
func (h *AdminHandler) enqueueMigration(ctx context.Context, r *http.Request, companyID int) (int64, error) {
	if h.river == nil {
		return 0, nil
	}
	file, header, err := r.FormFile("backup")
	if err == http.ErrMissingFile {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read backup file: %w", err)
	}
	defer file.Close()

	run, err := h.migrationRunStore.Create(ctx, int64(companyID), header.Filename)
	if err != nil {
		return 0, fmt.Errorf("create migration run: %w", err)
	}

	if err := os.MkdirAll(h.migrationsDir, 0755); err != nil {
		return 0, fmt.Errorf("create migrations dir: %w", err)
	}
	bakPath := filepath.Join(h.migrationsDir, fmt.Sprintf("%d.bak", run.ID))
	f, err := os.Create(bakPath)
	if err != nil {
		return 0, fmt.Errorf("save bak file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, file); err != nil {
		return 0, fmt.Errorf("write bak file: %w", err)
	}

	_, err = h.river.Insert(ctx, riverargs.MigrateArgs{
		RunID:     run.ID,
		CompanyID: companyID,
		BakPath:   bakPath,
	}, &river.InsertOpts{Queue: "migration"})
	if err != nil {
		return 0, fmt.Errorf("enqueue migration: %w", err)
	}

	return run.ID, nil
}

// RegisterAdmin registers super_admin-only routes with the given middleware applied per-handler.
func (h *AdminHandler) RegisterAdmin(mux *http.ServeMux, mw func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return mw(fn) }
	mux.Handle("GET /admin/companies", wrap(h.listCompanies))
	mux.Handle("GET /admin/companies/new", wrap(h.newCompany))
	mux.Handle("POST /admin/companies", wrap(h.createCompany))
	mux.Handle("GET /admin/companies/{id}/edit", wrap(h.editCompany))
	mux.Handle("POST /admin/companies/{id}", wrap(h.updateCompany))
	mux.Handle("GET /admin/companies/{companyID}/users", wrap(h.adminListUsers))
	mux.Handle("GET /admin/companies/{companyID}/users/new", wrap(h.adminNewUser))
	mux.Handle("POST /admin/companies/{companyID}/users", wrap(h.adminCreateUser))
	mux.Handle("GET /admin/companies/{companyID}/users/{id}/edit", wrap(h.adminEditUser))
	mux.Handle("POST /admin/companies/{companyID}/users/{id}", wrap(h.adminUpdateUser))
	mux.Handle("GET /admin/api-keys", wrap(h.listApiKeys))
	mux.Handle("POST /admin/api-keys", wrap(h.createApiKey))
	mux.Handle("POST /admin/api-keys/{id}/revoke", wrap(h.revokeApiKey))
}

// RegisterSettings registers company-level user management routes with the given middleware.
func (h *AdminHandler) RegisterSettings(mux *http.ServeMux, mw func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return mw(fn) }
	mux.Handle("GET /settings/users", wrap(h.listUsers))
	mux.Handle("GET /settings/users/new", wrap(h.newUser))
	mux.Handle("POST /settings/users", wrap(h.createUser))
	mux.Handle("GET /settings/users/{id}/edit", wrap(h.editUser))
	mux.Handle("POST /settings/users/{id}", wrap(h.updateUser))
}

// RegisterProfile registers the self-service change-password route (all authenticated users).
func (h *AdminHandler) RegisterProfile(mux *http.ServeMux) {
	mux.HandleFunc("GET /settings/change-password", h.showChangePassword)
	mux.HandleFunc("POST /settings/change-password", h.handleChangePassword)
}

func (h *AdminHandler) showChangePassword(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, settings.ChangePasswordPage(pg, "", ""))
}

func (h *AdminHandler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)

	currentPw := r.FormValue("current_password")
	newPw := r.FormValue("new_password")
	confirmPw := r.FormValue("confirm_password")

	if currentPw == "" || newPw == "" || confirmPw == "" {
		h.deps.renderTempl(w, r, settings.ChangePasswordPage(pg, "All fields are required.", ""))
		return
	}
	if newPw != confirmPw {
		h.deps.renderTempl(w, r, settings.ChangePasswordPage(pg, "New passwords do not match.", ""))
		return
	}
	if len(newPw) < 8 {
		h.deps.renderTempl(w, r, settings.ChangePasswordPage(pg, "New password must be at least 8 characters.", ""))
		return
	}

	user, err := h.userStore.GetByID(r.Context(), pg.User.ID)
	if err != nil {
		h.deps.renderTempl(w, r, settings.ChangePasswordPage(pg, "Failed to load user.", ""))
		return
	}
	if err := auth.CheckPassword(user.PasswordHash, currentPw); err != nil {
		h.deps.renderTempl(w, r, settings.ChangePasswordPage(pg, "Current password is incorrect.", ""))
		return
	}

	hash, err := auth.HashPassword(newPw)
	if err != nil {
		h.deps.renderTempl(w, r, settings.ChangePasswordPage(pg, "Failed to hash password.", ""))
		return
	}
	if err := h.userStore.UpdatePasswordByID(r.Context(), pg.User.ID, hash); err != nil {
		h.deps.renderTempl(w, r, settings.ChangePasswordPage(pg, "Failed to update password.", ""))
		return
	}

	h.deps.renderTempl(w, r, settings.ChangePasswordPage(pg, "", "Password changed successfully."))
}

// --- Company Management (super_admin only) ---

func (h *AdminHandler) listCompanies(w http.ResponseWriter, r *http.Request) {
	companies, err := h.companyStore.ListAll(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}

	// Build a map of companyID -> Subscription for the table
	subMap := make(map[int]models.Subscription)
	if subs, err := h.subscriptionStore.ListAll(r.Context()); err == nil {
		for _, s := range subs {
			subMap[s.CompanyID] = s
		}
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, admin.CompaniesPage(pg, companies, subMap))
}

func (h *AdminHandler) newCompany(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, admin.CompanyFormPage(pg, nil, true, nil, "", nil))
}

func (h *AdminHandler) createCompany(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 30); err != nil {
		_ = r.ParseForm()
	}

	c := &models.Company{
		CompanyName: formStringRequired(r, "company_name"),
		Slug:        formStringRequired(r, "slug"),
		Active:      formBool(r, "active"),
		Address:     formString(r, "address"),
		City:        formString(r, "city"),
		State:       uppercaseState(formString(r, "state")),
		Zip:         formString(r, "zip"),
		Phone:       stripPhone(formString(r, "phone")),
	}

	pg := h.deps.pageContext(w, r)

	if errs := validateCompany(c); len(errs) > 0 {
		h.deps.renderTempl(w, r, admin.CompanyFormPage(pg, c, true, errs, "", nil))
		return
	}

	if err := h.companyStore.Create(r.Context(), c); err != nil {
		log.Printf("create company: %v", err)
		h.deps.renderTempl(w, r, admin.CompanyFormPage(pg, c, true, nil, "Failed to create company", nil))
		return
	}

	// Create default Basic subscription for new company
	sub := &models.Subscription{
		CompanyID: c.ID,
		Tier:      models.TierBasic,
		Status:    models.StatusActive,
		AddonEDI:  false,
	}
	if err := h.subscriptionStore.Upsert(r.Context(), sub); err != nil {
		log.Printf("create subscription for company %d: %v", c.ID, err)
	}

	runID, err := h.enqueueMigration(r.Context(), r, c.ID)
	if err != nil {
		log.Printf("enqueue migration for company %d: %v", c.ID, err)
	}

	h.deps.setFlash(w, "Company created successfully")
	if runID > 0 {
		http.Redirect(w, r, fmt.Sprintf("/admin/migration/%d", runID), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/companies", http.StatusSeeOther)
}

func (h *AdminHandler) editCompany(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	c, err := h.companyStore.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	sub, err := h.subscriptionStore.GetByCompanyID(r.Context(), id)
	if err != nil {
		sub = nil
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, admin.CompanyFormPage(pg, c, false, nil, "", sub))
}

func (h *AdminHandler) updateCompany(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 30); err != nil {
		_ = r.ParseForm()
	}

	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	c := &models.Company{
		ID:          id,
		CompanyName: formStringRequired(r, "company_name"),
		Slug:        formStringRequired(r, "slug"),
		Active:      formBool(r, "active"),
		Address:     formString(r, "address"),
		Address2:    formString(r, "address2"),
		City:        formString(r, "city"),
		State:       uppercaseState(formString(r, "state")),
		Zip:         formString(r, "zip"),
		Phone:       stripPhone(formString(r, "phone")),
		Fax:         stripPhone(formString(r, "fax")),
		SCAC:        formString(r, "scac"),
		FederalID:   formString(r, "federal_id"),
		MCNumber:    formString(r, "mc_number"),
		DOTNumber:   formString(r, "dot_number"),
		SPLC:        formString(r, "splc"),
	}

	pg := h.deps.pageContext(w, r)

	if errs := validateCompany(c); len(errs) > 0 {
		sub, _ := h.subscriptionStore.GetByCompanyID(r.Context(), id)
		h.deps.renderTempl(w, r, admin.CompanyFormPage(pg, c, false, errs, "", sub))
		return
	}

	if err := h.companyStore.UpdateByID(r.Context(), c); err != nil {
		log.Printf("update company: %v", err)
		sub, _ := h.subscriptionStore.GetByCompanyID(r.Context(), id)
		h.deps.renderTempl(w, r, admin.CompanyFormPage(pg, c, false, nil, "Failed to update", sub))
		return
	}

	// Update subscription
	tier := formStringRequired(r, "tier")
	if !models.ValidTier(tier) {
		tier = models.TierBasic
	}
	status := formStringRequired(r, "sub_status")
	if status != models.StatusSuspended {
		status = models.StatusActive
	}
	sub := &models.Subscription{
		CompanyID:       id,
		Tier:            tier,
		Status:          status,
		AddonEDI:        formBool(r, "addon_edi"),
		EDIMonthlyLimit: formInt(r, "edi_monthly_limit"),
	}
	if err := h.subscriptionStore.Upsert(r.Context(), sub); err != nil {
		log.Printf("upsert subscription for company %d: %v", id, err)
	}
	h.deps.InvalidateSubscription(id)

	runID, err := h.enqueueMigration(r.Context(), r, id)
	if err != nil {
		log.Printf("enqueue migration for company %d: %v", id, err)
	}

	h.deps.setFlash(w, "Company updated successfully")
	if runID > 0 {
		http.Redirect(w, r, fmt.Sprintf("/admin/migration/%d", runID), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/companies", http.StatusSeeOther)
}

// validateUser checks field lengths and required fields, returns per-field errors.
func validateUser(username, email, password, role string, isNew bool) map[string]string {
	errs := make(map[string]string)
	if username == "" {
		errs["username"] = "Username is required"
	} else if len(username) > 50 {
		errs["username"] = "Username must be 50 characters or less"
	}
	if len(email) > 255 {
		errs["email"] = "Email must be 255 characters or less"
	}
	if isNew && password == "" {
		errs["password"] = "Password is required"
	}
	if role == "" {
		errs["role"] = "Role is required"
	} else if role != "user" && role != "company_admin" && role != "super_admin" {
		errs["role"] = "Invalid role"
	}
	return errs
}

// --- User Management (company_admin + super_admin) ---

func (h *AdminHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	companyID, err := auth.GetCompanyID(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	users, err := h.userStore.ListByCompany(r.Context(), companyID)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, settings.UsersPage(pg, users))
}

func (h *AdminHandler) loadTrucks(r *http.Request) []models.Truck {
	if h.truckStore == nil {
		return nil
	}
	trucks, _ := h.truckStore.ListAll(r.Context())
	return trucks
}

func (h *AdminHandler) newUser(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, settings.UserFormPage(pg, true, nil, nil, nil, "", h.loadTrucks(r)))
}

func (h *AdminHandler) createUser(w http.ResponseWriter, r *http.Request) {
	ctxUser, _ := auth.GetUserFromRequest(r)
	companyID, err := auth.GetCompanyID(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	username := formStringRequired(r, "username")
	email := formStringRequired(r, "email")
	password := formStringRequired(r, "password")
	role := formStringRequired(r, "role")
	defaultTruckID := formInt(r, "default_truck_id")

	// Non-super_admin can only create "user" role
	if ctxUser.Role != "super_admin" && role != "user" {
		role = "user"
	}

	formData := map[string]string{"username": username, "email": email, "role": role}

	pg := h.deps.pageContext(w, r)
	trucks := h.loadTrucks(r)

	if errs := validateUser(username, email, password, role, true); len(errs) > 0 {
		h.deps.renderTempl(w, r, settings.UserFormPage(pg, true, formData, nil, errs, "", trucks))
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	u := &models.User{
		Username:       username,
		Email:          email,
		PasswordHash:   hash,
		Role:           role,
		Active:         true,
		CompanyID:      &companyID,
		DefaultTruckID: defaultTruckID,
	}

	if err := h.userStore.Create(r.Context(), u); err != nil {
		log.Printf("create user: %v", err)
		h.deps.renderTempl(w, r, settings.UserFormPage(pg, true, formData, nil, nil, "Failed to create user", trucks))
		return
	}

	h.deps.setFlash(w, "User created successfully")
	http.Redirect(w, r, "/settings/users", http.StatusSeeOther)
}

func (h *AdminHandler) editUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	u, err := h.userStore.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Verify user belongs to same company
	companyID, err := auth.GetCompanyID(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	if u.CompanyID == nil || *u.CompanyID != companyID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, settings.UserFormPage(pg, false, nil, u, nil, "", h.loadTrucks(r)))
}

func (h *AdminHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	ctxUser, _ := auth.GetUserFromRequest(r)
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	companyID, err := auth.GetCompanyID(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	username := formStringRequired(r, "username")
	email := formStringRequired(r, "email")
	password := formStringRequired(r, "password")
	role := formStringRequired(r, "role")
	defaultTruckID := formInt(r, "default_truck_id")

	// Non-super_admin can only assign "user" role
	if ctxUser.Role != "super_admin" && role != "user" && role != "company_admin" {
		role = "user"
	}

	u := &models.User{
		ID:             id,
		Username:       username,
		Email:          email,
		Role:           role,
		Active:         formBool(r, "active"),
		CompanyID:      &companyID,
		DefaultTruckID: defaultTruckID,
	}

	pg := h.deps.pageContext(w, r)
	trucks := h.loadTrucks(r)

	if errs := validateUser(username, email, password, role, false); len(errs) > 0 {
		h.deps.renderTempl(w, r, settings.UserFormPage(pg, false, nil, u, errs, "", trucks))
		return
	}

	if err := h.userStore.Update(r.Context(), u); err != nil {
		log.Printf("update user: %v", err)
		h.deps.renderTempl(w, r, settings.UserFormPage(pg, false, nil, u, nil, "Failed to update user", trucks))
		return
	}

	// If a new password was provided, update it
	if password != "" {
		hash, err := auth.HashPassword(password)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := h.userStore.UpdatePassword(r.Context(), id, companyID, hash); err != nil {
			log.Printf("update user password: %v", err)
			h.deps.renderTempl(w, r, settings.UserFormPage(pg, false, nil, u, nil, "User updated but password change failed", trucks))
			return
		}
	}

	h.deps.setFlash(w, "User updated successfully")
	http.Redirect(w, r, "/settings/users", http.StatusSeeOther)
}

// --- Super Admin: User Management per Company ---

// adminCompanyContext loads the company by path param.
// Returns the company and true on success, or writes an error response and returns false.
func (h *AdminHandler) adminCompanyContext(w http.ResponseWriter, r *http.Request) (*models.Company, bool) {
	cid, err := parsePathID(r, "companyID")
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return nil, false
	}

	company, err := h.companyStore.GetByID(r.Context(), cid)
	if err != nil {
		http.Error(w, "Company not found", http.StatusNotFound)
		return nil, false
	}
	return company, true
}

func (h *AdminHandler) adminListUsers(w http.ResponseWriter, r *http.Request) {
	company, ok := h.adminCompanyContext(w, r)
	if !ok {
		return
	}

	users, err := h.userStore.ListByCompany(r.Context(), company.ID)
	if err != nil {
		serverError(w, err)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, admin.CompanyUsersPage(pg, company, users))
}

func (h *AdminHandler) adminNewUser(w http.ResponseWriter, r *http.Request) {
	company, ok := h.adminCompanyContext(w, r)
	if !ok {
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, admin.CompanyUserFormPage(pg, company, true, nil, nil, nil, ""))
}

func (h *AdminHandler) adminCreateUser(w http.ResponseWriter, r *http.Request) {
	company, ok := h.adminCompanyContext(w, r)
	if !ok {
		return
	}

	username := formStringRequired(r, "username")
	email := formStringRequired(r, "email")
	password := formStringRequired(r, "password")
	role := formStringRequired(r, "role")

	formData := map[string]string{"username": username, "email": email, "role": role}
	basePath := fmt.Sprintf("/admin/companies/%d/users", company.ID)

	pg := h.deps.pageContext(w, r)

	if errs := validateUser(username, email, password, role, true); len(errs) > 0 {
		h.deps.renderTempl(w, r, admin.CompanyUserFormPage(pg, company, true, formData, nil, errs, ""))
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	u := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		Active:       true,
		CompanyID:    &company.ID,
	}

	if err := h.userStore.Create(r.Context(), u); err != nil {
		log.Printf("create user: %v", err)
		h.deps.renderTempl(w, r, admin.CompanyUserFormPage(pg, company, true, formData, nil, nil, "Failed to create user"))
		return
	}

	h.deps.setFlash(w, "User created successfully")
	http.Redirect(w, r, basePath, http.StatusSeeOther)
}

func (h *AdminHandler) adminEditUser(w http.ResponseWriter, r *http.Request) {
	company, ok := h.adminCompanyContext(w, r)
	if !ok {
		return
	}

	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	u, err := h.userStore.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify user belongs to this company
	if u.CompanyID == nil || *u.CompanyID != company.ID {
		http.Error(w, "User does not belong to this company", http.StatusForbidden)
		return
	}

	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, admin.CompanyUserFormPage(pg, company, false, nil, u, nil, ""))
}

func (h *AdminHandler) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	company, ok := h.adminCompanyContext(w, r)
	if !ok {
		return
	}

	id, err := parseID(r)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	username := formStringRequired(r, "username")
	email := formStringRequired(r, "email")
	password := formStringRequired(r, "password")
	role := formStringRequired(r, "role")
	basePath := fmt.Sprintf("/admin/companies/%d/users", company.ID)

	u := &models.User{
		ID:        id,
		Username:  username,
		Email:     email,
		Role:      role,
		Active:    formBool(r, "active"),
		CompanyID: &company.ID,
	}

	pg := h.deps.pageContext(w, r)

	if errs := validateUser(username, email, password, role, false); len(errs) > 0 {
		h.deps.renderTempl(w, r, admin.CompanyUserFormPage(pg, company, false, nil, u, errs, ""))
		return
	}

	if err := h.userStore.Update(r.Context(), u); err != nil {
		log.Printf("update user: %v", err)
		h.deps.renderTempl(w, r, admin.CompanyUserFormPage(pg, company, false, nil, u, nil, "Failed to update user"))
		return
	}

	if password != "" {
		hash, err := auth.HashPassword(password)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := h.userStore.UpdatePassword(r.Context(), id, company.ID, hash); err != nil {
			log.Printf("update user password: %v", err)
			h.deps.renderTempl(w, r, admin.CompanyUserFormPage(pg, company, false, nil, u, nil, "User updated but password change failed"))
			return
		}
	}

	h.deps.setFlash(w, "User updated successfully")
	http.Redirect(w, r, basePath, http.StatusSeeOther)
}

func (h *AdminHandler) listApiKeys(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	keys, err := h.apiKeyStore.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	h.deps.renderTempl(w, r, admin.ApiKeysPage(pg, keys, ""))
}

func (h *AdminHandler) createApiKey(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	label := strings.TrimSpace(r.FormValue("label"))
	if label == "" {
		h.deps.setFlash(w, "Label is required")
		http.Redirect(w, r, "/admin/api-keys", http.StatusSeeOther)
		return
	}

	raw, hash := auth.GenerateAPIKey()
	if _, err := h.apiKeyStore.Create(r.Context(), 1, label, hash); err != nil {
		serverError(w, err)
		return
	}

	keys, err := h.apiKeyStore.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	h.deps.renderTempl(w, r, admin.ApiKeysPage(pg, keys, raw))
}

func (h *AdminHandler) revokeApiKey(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	if err := h.apiKeyStore.Revoke(r.Context(), id); err != nil {
		serverError(w, err)
		return
	}
	h.deps.setFlash(w, "API key revoked")
	http.Redirect(w, r, "/admin/api-keys", http.StatusSeeOther)
}
