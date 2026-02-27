package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/handler/components/admin"
	"github.com/brady1408/atlinks/internal/handler/components/settings"
	"github.com/brady1408/atlinks/internal/models"
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
}

type adminSubscriptionStore interface {
	GetByCompanyID(ctx context.Context, companyID int) (*models.Subscription, error)
	Upsert(ctx context.Context, sub *models.Subscription) error
	ListAll(ctx context.Context) ([]models.Subscription, error)
}

type AdminHandler struct {
	companyStore      adminCompanyStore
	userStore         adminUserStore
	subscriptionStore adminSubscriptionStore
	deps              *Deps
}

func NewAdminHandler(companyStore adminCompanyStore, userStore adminUserStore, subscriptionStore adminSubscriptionStore, deps *Deps) *AdminHandler {
	return &AdminHandler{companyStore: companyStore, userStore: userStore, subscriptionStore: subscriptionStore, deps: deps}
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
		AddonEDI:  false,
	}
	if err := h.subscriptionStore.Upsert(r.Context(), sub); err != nil {
		log.Printf("create subscription for company %d: %v", c.ID, err)
	}

	h.deps.setFlash(w, "Company created successfully")
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
	sub := &models.Subscription{
		CompanyID: id,
		Tier:      tier,
		AddonEDI:  formBool(r, "addon_edi"),
		EDIMonthlyLimit: formInt(r, "edi_monthly_limit"),
	}
	if err := h.subscriptionStore.Upsert(r.Context(), sub); err != nil {
		log.Printf("upsert subscription for company %d: %v", id, err)
	}
	h.deps.InvalidateSubscription(id)

	h.deps.setFlash(w, "Company updated successfully")
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

func (h *AdminHandler) newUser(w http.ResponseWriter, r *http.Request) {
	pg := h.deps.pageContext(w, r)
	h.deps.renderTempl(w, r, settings.UserFormPage(pg, true, nil, nil, nil, ""))
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

	// Non-super_admin can only create "user" role
	if ctxUser.Role != "super_admin" && role != "user" {
		role = "user"
	}

	formData := map[string]string{"username": username, "email": email, "role": role}

	pg := h.deps.pageContext(w, r)

	if errs := validateUser(username, email, password, role, true); len(errs) > 0 {
		h.deps.renderTempl(w, r, settings.UserFormPage(pg, true, formData, nil, errs, ""))
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
		CompanyID:    &companyID,
	}

	if err := h.userStore.Create(r.Context(), u); err != nil {
		log.Printf("create user: %v", err)
		h.deps.renderTempl(w, r, settings.UserFormPage(pg, true, formData, nil, nil, "Failed to create user"))
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
	h.deps.renderTempl(w, r, settings.UserFormPage(pg, false, nil, u, nil, ""))
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

	// Non-super_admin can only assign "user" role
	if ctxUser.Role != "super_admin" && role != "user" && role != "company_admin" {
		role = "user"
	}

	u := &models.User{
		ID:        id,
		Username:  username,
		Email:     email,
		Role:      role,
		Active:    formBool(r, "active"),
		CompanyID: &companyID,
	}

	pg := h.deps.pageContext(w, r)

	if errs := validateUser(username, email, password, role, false); len(errs) > 0 {
		h.deps.renderTempl(w, r, settings.UserFormPage(pg, false, nil, u, errs, ""))
		return
	}

	if err := h.userStore.Update(r.Context(), u); err != nil {
		log.Printf("update user: %v", err)
		h.deps.renderTempl(w, r, settings.UserFormPage(pg, false, nil, u, nil, "Failed to update user"))
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
			h.deps.renderTempl(w, r, settings.UserFormPage(pg, false, nil, u, nil, "User updated but password change failed"))
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
