package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"time"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/storage"
	"github.com/brady1408/atlinks/internal/store"
)

// MobileHandler serves JSON API endpoints for the driver mobile app.
type MobileHandler struct {
	userStore       mobileUserStore
	tripStore       mobileTripStore
	loadDetailStore mobileLoadDetailStore
	vehicleStore    mobileVehicleStore
	orderSvc        mobileOrderService
	damageStore     mobileDamageStore
	attachmentStore mobileAttachmentStore
	storageSvc      *storage.Service
	deps            *Deps
	truckStore      mobileTruckStore
	checkinStore    mobileCheckinStore
}

type mobileUserStore interface {
	GetByUsername(ctx context.Context, username string) (*models.User, error)
}

type mobileTripStore interface {
	List(ctx context.Context, f models.TripFilter) (*models.TripListResult, error)
	GetByID(ctx context.Context, id int) (*models.Trip, error)
}

type mobileLoadDetailStore interface {
	ListByTripWithOrder(ctx context.Context, tripID int) ([]store.LoadDetailWithOrder, error)
	UpdateStatusLocation(ctx context.Context, id int, lat, lng float64) error
}

type mobileVehicleStore interface {
	GetByID(ctx context.Context, id int) (*models.OrderVehicle, error)
}

type mobileOrderService interface {
	UpdateVehicleStatus(ctx context.Context, vehicleID int, newStatus string, confirmedBy *string) error
}

type mobileDamageStore interface {
	ListByVehicle(ctx context.Context, vehicleID int) ([]models.VehicleDamage, error)
	Create(ctx context.Context, d *models.VehicleDamage) error
}

type mobileAttachmentStore interface {
	Create(ctx context.Context, att *models.Attachment) error
	ListByEntity(ctx context.Context, category string, entityID int) ([]models.Attachment, error)
}

type mobileTruckStore interface {
	ListAll(ctx context.Context) ([]models.Truck, error)
}

type mobileCheckinStore interface {
	Create(ctx context.Context, c *models.TruckCheckin) error
}


func NewMobileHandler(
	userStore mobileUserStore,
	tripStore mobileTripStore,
	loadDetailStore mobileLoadDetailStore,
	vehicleStore mobileVehicleStore,
	orderSvc mobileOrderService,
	damageStore mobileDamageStore,
	attachmentStore mobileAttachmentStore,
	storageSvc *storage.Service,
	deps *Deps,
	truckStore mobileTruckStore,
	checkinStore mobileCheckinStore,
) *MobileHandler {
	return &MobileHandler{
		userStore:       userStore,
		tripStore:       tripStore,
		loadDetailStore: loadDetailStore,
		vehicleStore:    vehicleStore,
		orderSvc:        orderSvc,
		damageStore:     damageStore,
		attachmentStore: attachmentStore,
		storageSvc:      storageSvc,
		deps:            deps,
		truckStore:      truckStore,
		checkinStore:    checkinStore,
	}
}

// Register registers the mobile API routes on the given mux.
// These routes should be mounted under auth middleware.
func (h *MobileHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/driver/trips", h.listTrips)
	mux.HandleFunc("GET /api/v1/driver/trips/{id}", h.getTrip)
	mux.HandleFunc("POST /api/v1/driver/vehicles/{id}/status", h.updateVehicleStatus)
	mux.HandleFunc("GET /api/v1/driver/vehicles/{id}/damage", h.listDamage)
	mux.HandleFunc("POST /api/v1/driver/vehicles/{id}/damage", h.createDamage)
	mux.HandleFunc("GET /api/v1/driver/vehicles/{id}/photos", h.listPhotos)
	mux.HandleFunc("POST /api/v1/driver/vehicles/{id}/photos", h.uploadPhoto)
	mux.HandleFunc("GET /api/v1/driver/trucks", h.listTrucks)
	mux.HandleFunc("POST /api/v1/driver/checkin", h.createCheckin)
}

// RegisterAuth registers the login endpoint on the public mux (no auth required).
func (h *MobileHandler) RegisterAuth(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
}

func (h *MobileHandler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("mobile api: json encode: %v", err)
	}
}

func (h *MobileHandler) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}

// --- Auth ---

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  loginUserDTO `json:"user"`
}

type loginUserDTO struct {
	ID             int  `json:"id"`
	Username       string `json:"username"`
	Role           string `json:"role"`
	CompanyID      int    `json:"company_id"`
	DefaultTruckID *int   `json:"default_truck_id,omitempty"`
}

func (h *MobileHandler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "username and password required")
		return
	}

	user, err := h.userStore.GetByUsername(r.Context(), req.Username)
	if err != nil || !user.Active {
		h.writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	if err := auth.CheckPassword(user.PasswordHash, req.Password); err != nil {
		h.writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	companyID := 0
	if user.CompanyID != nil {
		companyID = *user.CompanyID
	}

	token, err := h.deps.JWT.GenerateToken(user.ID, user.Username, user.Role, companyID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	h.writeJSON(w, http.StatusOK, loginResponse{
		Token: token,
		User: loginUserDTO{
			ID:             user.ID,
			Username:       user.Username,
			Role:           user.Role,
			CompanyID:      companyID,
			DefaultTruckID: user.DefaultTruckID,
		},
	})
}

// --- Trips ---

type tripDTO struct {
	ID             int        `json:"id"`
	LoadNumber     string     `json:"load_number"`
	Active         bool       `json:"active"`
	TruckNumber    *string    `json:"truck_number,omitempty"`
	TrailerNumber  *string    `json:"trailer_number,omitempty"`
	Driver         *string    `json:"driver,omitempty"`
	TripDate       *time.Time `json:"trip_date,omitempty"`
	EstDeliverDate *time.Time `json:"est_deliver_date,omitempty"`
	DeliverDate    *time.Time `json:"deliver_date,omitempty"`
	Status         *string    `json:"status,omitempty"`
	EquipmentType  *string    `json:"equipment_type,omitempty"`
	Comments       *string    `json:"comments,omitempty"`
	VehicleCount   int        `json:"vehicle_count,omitempty"`
}

type tripDetailDTO struct {
	tripDTO
	Vehicles []loadDetailDTO `json:"vehicles"`
}

type loadDetailDTO struct {
	ID          int        `json:"id"`
	VehicleID   *int       `json:"vehicle_id,omitempty"`
	OrderID     *int       `json:"order_id,omitempty"`
	OrderNumber string     `json:"order_number"`
	VIN         *string    `json:"vin,omitempty"`
	Year        *string    `json:"year,omitempty"`
	Make        *string    `json:"make,omitempty"`
	Model       *string    `json:"model,omitempty"`
	Color       *string    `json:"color,omitempty"`
	Weight      *int       `json:"weight,omitempty"`
	Category    *string    `json:"category,omitempty"`
	BayNumber   *string    `json:"bay_number,omitempty"`
	Status      *string    `json:"status,omitempty"`
	LoadedDate  *time.Time `json:"loaded_date,omitempty"`
	DeliveredDate *time.Time `json:"delivered_date,omitempty"`
}

func tripToDTO(t models.Trip, vehicleCount int) tripDTO {
	return tripDTO{
		ID:             t.ID,
		LoadNumber:     t.LoadNumber,
		Active:         t.Active,
		TruckNumber:    t.TruckNumber,
		TrailerNumber:  t.TrailerNumber,
		Driver:         t.Driver,
		TripDate:       t.TripDate,
		EstDeliverDate: t.EstDeliverDate,
		DeliverDate:    t.DeliverDate,
		Status:         t.Status,
		EquipmentType:  t.EquipmentType,
		Comments:       t.Comments,
		VehicleCount:   vehicleCount,
	}
}

func (h *MobileHandler) listTrips(w http.ResponseWriter, r *http.Request) {
	filter := models.TripFilter{
		Search:      r.URL.Query().Get("search"),
		TruckNumber: r.URL.Query().Get("truck"),
		Active:      "active",
		Page:        intParam(r, "page", 1),
		PageSize:    50,
	}

	result, err := h.tripStore.List(r.Context(), filter)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to load trips")
		return
	}

	trips := make([]tripDTO, len(result.Items))
	for i, t := range result.Items {
		// Get vehicle count from load details
		loads, _ := h.loadDetailStore.ListByTripWithOrder(r.Context(), t.ID)
		trips[i] = tripToDTO(t, len(loads))
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"trips":       trips,
		"total_count": result.TotalCount,
		"page":        result.Page,
		"page_size":   result.PageSize,
	})
}

func (h *MobileHandler) getTrip(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid trip ID")
		return
	}

	trip, err := h.tripStore.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "trip not found")
		return
	}

	loads, err := h.loadDetailStore.ListByTripWithOrder(r.Context(), id)
	if err != nil {
		log.Printf("mobile api: load manifest for trip %d: %v", id, err)
		loads = nil
	}

	vehicles := make([]loadDetailDTO, len(loads))
	for i, ld := range loads {
		vehicles[i] = loadDetailDTO{
			ID:            ld.ID,
			VehicleID:     ld.VehicleID,
			OrderID:       ld.OrderID,
			OrderNumber:   ld.OrderNumber,
			VIN:           ld.VIN,
			Year:          ld.Year,
			Make:          ld.Make,
			Model:         ld.Model,
			Color:         ld.Color,
			Weight:        ld.Weight,
			Category:      ld.Category,
			BayNumber:     ld.BayNumber,
			Status:        ld.Status,
			LoadedDate:    ld.LoadedDate,
			DeliveredDate: ld.DeliveredDate,
		}
	}

	detail := tripDetailDTO{
		tripDTO:  tripToDTO(*trip, len(loads)),
		Vehicles: vehicles,
	}

	h.writeJSON(w, http.StatusOK, detail)
}

// --- Vehicle Status ---

type statusRequest struct {
	Status    string   `json:"status"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

func (h *MobileHandler) updateVehicleStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid vehicle ID")
		return
	}

	var req statusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	validStatuses := map[string]bool{
		"Waiting": true, "Scheduled": true, "Loaded": true,
		"Delivered": true, "Confirmed": true,
	}
	if !validStatuses[req.Status] {
		h.writeError(w, http.StatusBadRequest, "invalid status; must be one of: Waiting, Scheduled, Loaded, Delivered, Confirmed")
		return
	}

	var confirmedBy *string
	if req.Status == "Confirmed" {
		if user, ok := auth.GetUserFromRequest(r); ok {
			confirmedBy = &user.Username
		}
	}

	if err := h.orderSvc.UpdateVehicleStatus(r.Context(), id, req.Status, confirmedBy); err != nil {
		h.writeError(w, http.StatusBadRequest, "failed to update status")
		return
	}

	if req.Latitude != nil && req.Longitude != nil {
		if err := h.loadDetailStore.UpdateStatusLocation(r.Context(), id, *req.Latitude, *req.Longitude); err != nil {
			log.Printf("mobile api: save status location for vehicle %d: %v", id, err)
		}
	}

	vehicle, err := h.vehicleStore.GetByID(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "status updated but failed to fetch vehicle")
		return
	}

	h.writeJSON(w, http.StatusOK, vehicle)
}

// --- Damage ---

func (h *MobileHandler) listDamage(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid vehicle ID")
		return
	}

	damages, err := h.damageStore.ListByVehicle(r.Context(), vehicleID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list damage records")
		return
	}

	h.writeJSON(w, http.StatusOK, damages)
}

type createDamageRequest struct {
	DamageArea      string `json:"damage_area"`
	DamageType      string `json:"damage_type"`
	DamageSeverity  string `json:"damage_severity"`
	Description     string `json:"description"`
	InspectionPoint string `json:"inspection_point"`
}

func (h *MobileHandler) createDamage(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid vehicle ID")
		return
	}

	var req createDamageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	now := time.Now()
	var inspectedBy *string
	if user, ok := auth.GetUserFromRequest(r); ok {
		inspectedBy = &user.Username
	}

	d := &models.VehicleDamage{
		VehicleID:       &vehicleID,
		DamageArea:      strPtr(req.DamageArea),
		DamageType:      strPtr(req.DamageType),
		DamageSeverity:  strPtr(req.DamageSeverity),
		Description:     strPtr(req.Description),
		InspectionPoint: strPtr(req.InspectionPoint),
		InspectedBy:     inspectedBy,
		InspectionDate:  &now,
	}

	if err := h.damageStore.Create(r.Context(), d); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to create damage record")
		return
	}

	h.deps.Audit.Log(r.Context(), "vehicle_damage", d.ID, "INSERT", nil, d)
	h.writeJSON(w, http.StatusCreated, d)
}

// --- Photos ---

func (h *MobileHandler) listPhotos(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid vehicle ID")
		return
	}

	photos, err := h.attachmentStore.ListByEntity(r.Context(), "vehicle_inspection", vehicleID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list photos")
		return
	}

	type photoDTO struct {
		ID          int       `json:"id"`
		Filename    string    `json:"filename"`
		ContentType string    `json:"content_type"`
		SizeBytes   int64     `json:"size_bytes"`
		CreatedAt   time.Time `json:"created_at"`
	}

	result := make([]photoDTO, len(photos))
	for i, p := range photos {
		result[i] = photoDTO{
			ID:          p.ID,
			Filename:    p.Filename,
			ContentType: p.ContentType,
			SizeBytes:   p.SizeBytes,
			CreatedAt:   p.CreatedAt,
		}
	}

	h.writeJSON(w, http.StatusOK, result)
}

var mobileAllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

func (h *MobileHandler) uploadPhoto(w http.ResponseWriter, r *http.Request) {
	vehicleID, err := parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid vehicle ID")
		return
	}

	user, ok := auth.GetUserFromRequest(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// 25MB limit
	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)

	file, header, err := r.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			h.writeError(w, http.StatusRequestEntityTooLarge, "file too large (max 25MB)")
			return
		}
		h.writeError(w, http.StatusBadRequest, "no file provided")
		return
	}
	defer file.Close()

	// Sniff content type
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	contentType := http.DetectContentType(buf[:n])
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to process file")
		return
	}

	if !mobileAllowedImageTypes[contentType] {
		h.writeError(w, http.StatusBadRequest, "only image files allowed (JPEG, PNG, GIF, WebP)")
		return
	}

	ext := filepath.Ext(header.Filename)
	if ext == "" {
		exts, _ := mime.ExtensionsByType(contentType)
		if len(exts) > 0 {
			ext = exts[0]
		}
	}

	storageKey, written, err := h.storageSvc.Save(user.CompanyID, "vehicle_inspection", vehicleID, ext, file)
	if err != nil {
		log.Printf("mobile api: save photo: %v", err)
		h.writeError(w, http.StatusInternalServerError, "failed to save file")
		return
	}

	att := &models.Attachment{
		CompanyID:   user.CompanyID,
		Category:    "vehicle_inspection",
		EntityID:    vehicleID,
		Filename:    header.Filename,
		StorageKey:  storageKey,
		ContentType: contentType,
		SizeBytes:   written,
		UploadedBy:  &user.ID,
	}

	if err := h.attachmentStore.Create(r.Context(), att); err != nil {
		h.storageSvc.Delete(storageKey)
		log.Printf("mobile api: create attachment record: %v", err)
		h.writeError(w, http.StatusInternalServerError, "failed to save attachment")
		return
	}

	h.deps.Audit.Log(r.Context(), "attachments", att.ID, "INSERT", nil, att)

	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id":           att.ID,
		"filename":     att.Filename,
		"content_type": att.ContentType,
		"size_bytes":   att.SizeBytes,
	})
}

// --- Trucks ---

type truckListDTO struct {
	ID            int     `json:"id"`
	TruckNumber   string  `json:"truck_number"`
	TruckMake     *string `json:"truck_make,omitempty"`
	TruckModel    *string `json:"truck_model,omitempty"`
	TruckYear     *string `json:"truck_year,omitempty"`
	TrailerNumber *string `json:"trailer_number,omitempty"`
}

func (h *MobileHandler) listTrucks(w http.ResponseWriter, r *http.Request) {
	trucks, err := h.truckStore.ListAll(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to load trucks")
		return
	}

	result := make([]truckListDTO, len(trucks))
	for i, t := range trucks {
		result[i] = truckListDTO{
			ID:            t.ID,
			TruckNumber:   t.TruckNumber,
			TruckMake:     t.TruckMake,
			TruckModel:    t.TruckModel,
			TruckYear:     t.TruckYear,
			TrailerNumber: t.TrailerNumber,
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"trucks": result})
}

// --- Check-in ---

type checkinRequest struct {
	TruckID   int      `json:"truck_id"`
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Accuracy  *float64 `json:"accuracy,omitempty"`
	Speed     *float64 `json:"speed,omitempty"`
	Heading   *float64 `json:"heading,omitempty"`
}

func (h *MobileHandler) createCheckin(w http.ResponseWriter, r *http.Request) {
	var req checkinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TruckID == 0 {
		h.writeError(w, http.StatusBadRequest, "truck_id is required")
		return
	}

	c := &models.TruckCheckin{
		TruckID:   req.TruckID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Accuracy:  req.Accuracy,
		Speed:     req.Speed,
		Heading:   req.Heading,
	}

	if err := h.checkinStore.Create(r.Context(), c); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save check-in")
		return
	}

	h.writeJSON(w, http.StatusCreated, c)
}

// strPtr is defined in payment_handler.go
