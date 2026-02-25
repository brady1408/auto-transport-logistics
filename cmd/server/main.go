package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/config"
	"github.com/brady1408/atlinks/internal/database"
	"github.com/brady1408/atlinks/internal/email"
	"github.com/brady1408/atlinks/internal/handler"
	"github.com/brady1408/atlinks/internal/handler/components"
	"github.com/brady1408/atlinks/internal/middleware"
	"github.com/brady1408/atlinks/internal/service"
	"github.com/brady1408/atlinks/internal/storage"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

var buildVersion string

func main() {
	migrateUp := flag.Bool("migrate-up", false, "Run database migrations up")
	migrateDown := flag.Bool("migrate-down", false, "Run database migrations down (one step)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if *migrateUp {
		fmt.Println("Running migrations up...")
		if err := database.MigrateUp(cfg.DatabaseURL); err != nil {
			log.Fatalf("migrate up: %v", err)
		}
		fmt.Println("Migrations complete.")
		return
	}

	if *migrateDown {
		fmt.Println("Running migration down (one step)...")
		if err := database.MigrateDown(cfg.DatabaseURL); err != nil {
			log.Fatalf("migrate down: %v", err)
		}
		fmt.Println("Migration rolled back.")
		return
	}

	ctx := context.Background()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	deps := initDeps(pool, cfg)
	mux := initRoutes(pool, cfg, deps)

	// Apply logging middleware to all routes
	var httpHandler http.Handler = mux
	httpHandler = middleware.RequestLogger(httpHandler)

	runServer(cfg, httpHandler, ctx)
}

func initDeps(pool *pgxpool.Pool, cfg *config.Config) *handler.Deps {
	jwtSvc := auth.NewJWTService(cfg.JWTSecret)
	auditSvc := audit.NewService(pool)

	version := buildVersion
	if version == "" {
		version = strconv.FormatInt(time.Now().Unix(), 10)
	}
	components.SetBuildVersion(version)

	secureCookies := strings.HasPrefix(cfg.AppBaseURL, "https://")

	companyStore := store.NewCompanyStore(pool)

	return &handler.Deps{
		JWT:           jwtSvc,
		Audit:         auditSvc,
		CompanyStore:  companyStore,
		BuildVersion:  version,
		SecureCookies: secureCookies,
	}
}

func initRoutes(pool *pgxpool.Pool, cfg *config.Config, deps *handler.Deps) *http.ServeMux {
	mux := http.NewServeMux()

	// Static files
	staticFS, err := fs.Sub(handler.StaticFS, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	// Stores
	userStore := store.NewUserStore(pool)
	customerStore := store.NewCustomerStore(pool)
	employeeStore := store.NewEmployeeStore(pool)
	truckStore := store.NewTruckStore(pool)
	zoneStore := store.NewZoneStore(pool)
	zonePricingStore := store.NewZonePricingStore(pool)
	companyStore := store.NewCompanyStore(pool)
	orderStore := store.NewOrderStore(pool)
	vehicleStore := store.NewVehicleStore(pool)
	tripStore := store.NewTripStore(pool)
	loadDetailStore := store.NewLoadDetailStore(pool)
	chargeStore := store.NewChargeStore(pool)
	damageStore := store.NewDamageStore(pool)
	noteStore := store.NewNoteStore(pool)
	fuelStore := store.NewTripFuelStore(pool)
	expenseStore := store.NewTripExpenseStore(pool)
	routeStore := store.NewTripRouteStore(pool)
	invoiceStore := store.NewInvoiceStore(pool)
	invoiceDetailStore := store.NewInvoiceDetailStore(pool)
	paymentStore := store.NewPaymentStore(pool)
	paymentDetailStore := store.NewPaymentDetailStore(pool)
	creditMemoStore := store.NewCreditMemoStore(pool)
	damageClaimStore := store.NewDamageClaimStore(pool)
	apStore := store.NewAccountsPayableStore(pool)
	feedbackStore := store.NewFeedbackStore(pool)
	attachmentStore := store.NewAttachmentStore(pool)

	// Storage service
	storageSvc, err := storage.NewService(cfg.UploadDir)
	if err != nil {
		log.Fatalf("storage service: %v", err)
	}

	// Services
	auditSvc := deps.Audit
	orderSvc := service.NewOrderService(pool, orderStore, vehicleStore, auditSvc)
	tripSvc := service.NewTripService(pool, tripStore, loadDetailStore, vehicleStore, orderStore, auditSvc)
	invoiceSvc := service.NewInvoiceService(pool, invoiceStore, invoiceDetailStore, orderStore, vehicleStore, auditSvc)
	paymentSvc := service.NewPaymentService(pool, paymentStore, paymentDetailStore, invoiceStore, auditSvc)

	// Email + password reset + registration
	emailSvc := email.NewService(cfg.ResendAPIKey, cfg.FromEmail)
	resetTokenStore := store.NewResetTokenStore(pool)
	pendingRegStore := store.NewPendingRegistrationStore(pool)

	// Auth routes (public)
	authHandler := handler.NewAuthHandler(userStore, companyStore, cfg.InviteCode, deps, emailSvc, resetTokenStore, pendingRegStore, cfg.AppBaseURL)
	authHandler.Register(mux)

	// Protected routes
	protectedMux := http.NewServeMux()

	// Dashboard
	dashHandler := handler.NewDashboardHandler(orderStore, invoiceStore, tripStore, deps)
	dashHandler.Register(protectedMux)

	// Global Masters
	handler.NewCustomerHandler(customerStore, deps).Register(protectedMux)
	handler.NewEmployeeHandler(employeeStore, deps).Register(protectedMux)
	handler.NewTruckHandler(truckStore, deps).Register(protectedMux)
	handler.NewZoneHandler(zoneStore, zonePricingStore, deps).Register(protectedMux)
	handler.NewCompanyHandler(companyStore, deps).Register(protectedMux)

	// Lookup tables
	registerLookups(protectedMux, pool, deps)

	// Terms, Tax Codes, Items
	handler.NewTermsHandler(store.NewTermsStore(pool), deps).Register(protectedMux)
	handler.NewTaxCodeHandler(store.NewTaxCodeStore(pool), deps).Register(protectedMux)
	handler.NewItemHandler(store.NewItemStore(pool), deps).Register(protectedMux)

	// Dispatch
	handler.NewOrderHandler(orderStore, invoiceSvc, deps).Register(protectedMux)
	handler.NewVehicleHandler(vehicleStore, orderStore, orderSvc, deps).Register(protectedMux)
	handler.NewTripHandler(tripStore, loadDetailStore, vehicleStore, tripSvc, deps).Register(protectedMux)
	handler.NewChargeHandler(chargeStore, deps).Register(protectedMux)
	handler.NewDamageHandler(damageStore, deps).Register(protectedMux)
	handler.NewNoteHandler(noteStore, deps).Register(protectedMux)
	handler.NewFuelHandler(fuelStore, deps).Register(protectedMux)
	handler.NewExpenseHandler(expenseStore, deps).Register(protectedMux)
	handler.NewRouteHandler(routeStore, deps).Register(protectedMux)
	handler.NewAPIHandler(customerStore, vehicleStore, deps).Register(protectedMux)

	// Accounting
	handler.NewInvoiceHandler(invoiceStore, invoiceDetailStore, paymentDetailStore, invoiceSvc, deps).Register(protectedMux)
	handler.NewPaymentHandler(paymentStore, paymentDetailStore, invoiceStore, paymentSvc, deps).Register(protectedMux)
	handler.NewCreditMemoHandler(creditMemoStore, deps).Register(protectedMux)
	handler.NewDamageClaimHandler(damageClaimStore, attachmentStore, storageSvc, deps).Register(protectedMux)
	handler.NewAccountsPayableHandler(apStore, deps).Register(protectedMux)

	// Feedback
	handler.NewFeedbackHandler(feedbackStore, attachmentStore, storageSvc, deps).Register(protectedMux)

	// Feedback API (API key auth, separate from JWT-protected routes)
	feedbackAPIHandler := handler.NewFeedbackAPIHandler(feedbackStore, deps)
	apiMux := http.NewServeMux()
	feedbackAPIHandler.Register(apiMux)
	apiKeyMiddleware := middleware.RequireAPIKey(cfg.APIKey)
	mux.Handle("/api/feedback", apiKeyMiddleware(apiMux))
	mux.Handle("/api/feedback/", apiKeyMiddleware(apiMux))

	// VIN Search + Reports
	handler.NewVinSearchHandler(vehicleStore, deps).Register(protectedMux)
	handler.NewReportHandler(orderStore, invoiceStore, tripStore, vehicleStore, paymentStore, damageClaimStore, deps).Register(protectedMux)

	// Upload handler (serve + delete routes for attachments)
	uploadHandler := handler.NewUploadHandler(attachmentStore, storageSvc, deps)
	uploadHandler.Register(protectedMux)
	uploadHandler.RegisterAdmin(protectedMux, middleware.RequireRole("super_admin"))

	// Admin + User Management
	adminHandler := handler.NewAdminHandler(companyStore, userStore, deps)
	adminHandler.RegisterAdmin(protectedMux, middleware.RequireRole("super_admin"))
	adminHandler.RegisterSettings(protectedMux, middleware.RequireRole("company_admin", "super_admin"))

	// Wrap protected routes with auth + CSRF middleware
	authMiddleware := middleware.RequireAuth(deps.JWT)
	csrfMiddleware := middleware.CSRF(deps.SecureCookies)
	mux.Handle("/", authMiddleware(csrfMiddleware(protectedMux)))

	return mux
}

func registerLookups(mux *http.ServeMux, pool *pgxpool.Pool, deps *handler.Deps) {
	lookups := []struct{ table, path, title string }{
		{"dispatch_codes", "/global/dispatch-codes", "Dispatch Codes"},
		{"damage_areas", "/global/damage-areas", "Damage Areas"},
		{"damage_types", "/global/damage-types", "Damage Types"},
		{"damage_severities", "/global/damage-severities", "Damage Severities"},
		{"equipment_types", "/global/equipment-types", "Equipment Types"},
		{"hold_codes", "/global/hold-codes", "Hold Codes"},
		{"declination_codes", "/global/declination-codes", "Declination Codes"},
		{"regions", "/global/regions", "Regions"},
		{"field_codes_1", "/global/field-codes-1", "Field Codes 1"},
		{"field_codes_2", "/global/field-codes-2", "Field Codes 2"},
		{"field_codes_3", "/global/field-codes-3", "Field Codes 3"},
		{"field_codes_4", "/global/field-codes-4", "Field Codes 4"},
		{"field_codes_5", "/global/field-codes-5", "Field Codes 5"},
	}
	for _, l := range lookups {
		ls, err := store.NewLookupStore(pool, l.table)
		if err != nil {
			log.Fatalf("lookup store %s: %v", l.table, err)
		}
		handler.NewLookupHandler(deps, ls, l.path, l.title).Register(mux)
	}
}

func runServer(cfg *config.Config, handler http.Handler, ctx context.Context) {
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
	log.Println("Server stopped.")
}
