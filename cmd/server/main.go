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

	"github.com/brady1408/auto-transport-logistics/internal/audit"
	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/brady1408/auto-transport-logistics/internal/config"
	crpc "github.com/brady1408/auto-transport-logistics/internal/connectrpc"
	"github.com/brady1408/auto-transport-logistics/internal/database"
	"github.com/brady1408/auto-transport-logistics/internal/email"
	"github.com/brady1408/auto-transport-logistics/internal/geocode"
	"github.com/brady1408/auto-transport-logistics/internal/handler"
	"github.com/brady1408/auto-transport-logistics/internal/handler/components"
	"github.com/brady1408/auto-transport-logistics/internal/middleware"
	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/brady1408/auto-transport-logistics/internal/qbo"
	"github.com/brady1408/auto-transport-logistics/internal/service"
	"github.com/brady1408/auto-transport-logistics/internal/storage"
	"github.com/brady1408/auto-transport-logistics/internal/store"
	"log/slog"

	"github.com/brady1408/auto-transport-logistics/internal/worker"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"golang.org/x/oauth2"
	"riverqueue.com/riverui"
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

	oauthCfg := qbo.NewOAuthConfig(cfg.QBOClientID, cfg.QBOClientSecret, cfg.QBORedirectURL)
	qboStore := store.NewQBOStore(pool)

	deps := initDeps(pool, cfg)
	mux, loadboardSvc, loadboardSt, routeStores := initRoutes(pool, cfg, deps)

	riverClient, err := initRiver(ctx, pool, cfg, qboStore, oauthCfg, routeStores)
	if err != nil {
		log.Fatalf("init river: %v", err)
	}
	// Wire RiverClient into stores for QBO sync
	routeStores.customerStore.RiverClient = riverClient
	routeStores.invoiceStore.RiverClient = riverClient
	routeStores.paymentStore.RiverClient = riverClient
	routeStores.invoiceSvc.RiverClient = riverClient

	// Mount River UI for super_admin job monitoring (bypasses CSRF — uses its own API).
	const riverUIPrefix = "/admin/riverui"
	uiServer, err := initRiverUI(ctx, riverClient, riverUIPrefix)
	if err != nil {
		log.Fatalf("river ui: %v", err)
	}
	authMw := middleware.RequireAuth(deps.JWT, deps.SecureCookies)
	mux.Handle(riverUIPrefix+"/",
		authMw(middleware.RequireRole("super_admin")(uiServer)),
	)

	// Register integrations handler (needs riverClient, only available post-initRiver).
	// Mutating routes (connect/disconnect/sync-all) require company_admin or super_admin.
	handler.NewIntegrationsHandler(
		qboStore, oauthCfg, riverClient,
		routeStores.customerStore,
		routeStores.invoiceStore,
		routeStores.paymentStore,
		deps,
	).Register(routeStores.protectedMux, middleware.RequireRole("company_admin", "super_admin"))

	// Register MSSQL migration handler (super_admin only).
	handler.NewMigrationHandler(
		routeStores.migrationRunStore,
		routeStores.companyStore,
		riverClient,
		cfg.MigrationsDir,
		deps,
	).Register(routeStores.protectedMux, middleware.RequireRole("super_admin"))

	// Admin + User Management (registered post-initRiver to receive migration deps).
	adminHandler := handler.NewAdminHandler(
		routeStores.companyStore,
		routeStores.userStore,
		routeStores.subscriptionStore,
		routeStores.migrationRunStore,
		routeStores.truckStore,
		routeStores.apiKeyStore,
		riverClient,
		cfg.MigrationsDir,
		deps,
	)
	adminHandler.RegisterAdmin(routeStores.protectedMux, middleware.RequireRole("super_admin"))
	adminHandler.RegisterSettings(routeStores.protectedMux, middleware.RequireRole("company_admin", "super_admin"))
	adminHandler.RegisterProfile(routeStores.protectedMux)

	// Background: expire loadboard listings every 5 minutes
	go runLoadboardExpiry(ctx, loadboardSvc)

	// Background: backfill geocode coords for existing listings
	go backfillGeocode(ctx, loadboardSt)

	// Apply logging middleware to all routes
	var httpHandler http.Handler = mux
	httpHandler = middleware.RequestLogger(httpHandler)

	runServer(cfg, httpHandler, ctx, riverClient)
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
	subscriptionStore := store.NewSubscriptionStore(pool)

	return &handler.Deps{
		JWT:               jwtSvc,
		Audit:             auditSvc,
		CompanyStore:      companyStore,
		SubscriptionStore: subscriptionStore,
		BuildVersion:      version,
		SecureCookies:     secureCookies,
	}
}

// riverStores holds the store references needed by initRiver and post-init registration.
type riverStores struct {
	customerStore      *store.CustomerStore
	invoiceStore       *store.InvoiceStore
	invoiceDetailStore *store.InvoiceDetailStore
	paymentStore       *store.PaymentStore
	paymentDetailStore *store.PaymentDetailStore
	invoiceSvc         *service.InvoiceService
	migrationRunStore  *store.MigrationRunStore
	companyStore       *store.CompanyStore
	userStore          *store.UserStore
	subscriptionStore  *store.SubscriptionStore
	activityStore      *store.ActivityStore
	truckStore         *store.TruckStore
	apiKeyStore        *store.ApiKeyStore
	deviceCodeStore    *store.DeviceCodeStore
	refreshTokenStore  *store.RefreshTokenStore
	protectedMux       *http.ServeMux
}

func initRoutes(pool *pgxpool.Pool, cfg *config.Config, deps *handler.Deps) (*http.ServeMux, *service.LoadboardService, *store.LoadboardStore, riverStores) {
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
	seqStore := store.NewSequenceStore(pool)
	orderStore := store.NewOrderStore(pool, seqStore)
	vehicleStore := store.NewVehicleStore(pool)
	tripStore := store.NewTripStore(pool, seqStore)
	loadDetailStore := store.NewLoadDetailStore(pool)
	chargeStore := store.NewChargeStore(pool)
	damageStore := store.NewDamageStore(pool)
	noteStore := store.NewNoteStore(pool)
	fuelStore := store.NewTripFuelStore(pool)
	expenseStore := store.NewTripExpenseStore(pool)
	routeStore := store.NewTripRouteStore(pool)
	invoiceStore := store.NewInvoiceStore(pool, seqStore)
	invoiceDetailStore := store.NewInvoiceDetailStore(pool)
	paymentStore := store.NewPaymentStore(pool)
	paymentDetailStore := store.NewPaymentDetailStore(pool)
	creditMemoStore := store.NewCreditMemoStore(pool, seqStore)
	damageClaimStore := store.NewDamageClaimStore(pool, seqStore)
	apStore := store.NewAccountsPayableStore(pool)
	feedbackStore := store.NewFeedbackStore(pool)
	attachmentStore := store.NewAttachmentStore(pool)
	earningsAdjStore := store.NewEarningsAdjStore(pool)
	activityStore := store.NewActivityStore(pool)
	apiKeyStore := store.NewApiKeyStore(pool)

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
	authSubStore := store.NewSubscriptionStore(pool)

	// Landing page (public)
	handler.NewLandingHandler(deps, emailSvc).Register(mux)

	// Auth routes (public)
	authHandler := handler.NewAuthHandler(userStore, companyStore, authSubStore, cfg.InviteCode, deps, emailSvc, resetTokenStore, pendingRegStore, cfg.AppBaseURL)
	authHandler.Register(mux)

	// Mobile API (public auth endpoint)
	checkinStore := store.NewCheckinStore(pool)
	mobileHandler := handler.NewMobileHandler(
		userStore, tripStore, loadDetailStore, vehicleStore, orderSvc,
		damageStore, attachmentStore, storageSvc, deps,
		truckStore, checkinStore,
	)
	mobileHandler.RegisterAuth(mux)

	// Protected routes
	protectedMux := http.NewServeMux()

	// Dashboard
	dashHandler := handler.NewDashboardHandler(orderStore, invoiceStore, tripStore, truckStore, deps)
	dashHandler.Register(protectedMux)

	// Global Masters
	handler.NewCustomerHandler(customerStore, deps).Register(protectedMux)
	handler.NewEmployeeHandler(employeeStore, deps).Register(protectedMux)
	handler.NewTruckHandler(truckStore, deps).Register(protectedMux)
	handler.NewZoneHandler(zoneStore, zonePricingStore, deps).Register(protectedMux)
	handler.NewCompanyHandler(companyStore, deps).Register(protectedMux)

	vendorStore := store.NewVendorStore(pool)
	handler.NewVendorHandler(vendorStore, deps).Register(protectedMux)

	// Lookup tables
	lookupStoresMap := registerLookups(protectedMux, pool, deps)

	// Terms, Tax Codes, Items
	termsStore := store.NewTermsStore(pool)
	taxCodeStore := store.NewTaxCodeStore(pool)
	itemStore := store.NewItemStore(pool)
	handler.NewTermsHandler(termsStore, deps).Register(protectedMux)
	handler.NewTaxCodeHandler(taxCodeStore, deps).Register(protectedMux)
	handler.NewItemHandler(itemStore, deps).Register(protectedMux)

	// Loadboard (gated to Pro+)
	loadboardStore := store.NewLoadboardStore(pool)
	loadboardSvc := service.NewLoadboardService(pool, loadboardStore, orderStore, vehicleStore, companyStore, orderSvc, auditSvc)
	featureCheck := func(r *http.Request) models.FeatureSet { return deps.GetFeatures(r) }
	loadboardGate := middleware.RequireFeature(featureCheck, models.FeatureLoadboard, deps.UpgradeHandler(models.FeatureLoadboard))
	handler.NewLoadboardHandler(loadboardStore, orderStore, vehicleStore, companyStore, loadboardSvc, deps).Register(protectedMux, loadboardGate)

	// Dispatch
	handler.NewOrderHandler(orderStore, invoiceSvc, vehicleStore, attachmentStore, loadboardStore, zonePricingStore, vehicleStore, deps).Register(protectedMux)
	handler.NewVehicleHandler(vehicleStore, orderStore, orderSvc, zonePricingStore, deps).Register(protectedMux)
	damageLabelStore, err := store.NewDamageLabelStore(pool)
	if err != nil {
		log.Fatalf("init damage label store: %v", err)
	}
	handler.NewTripHandler(tripStore, loadDetailStore, vehicleStore, tripSvc, attachmentStore, damageStore, damageLabelStore, deps).Register(protectedMux)
	handler.NewChargeHandler(chargeStore, deps).Register(protectedMux)
	handler.NewDamageHandler(damageStore, deps).Register(protectedMux)
	handler.NewNoteHandler(noteStore, deps).Register(protectedMux)
	handler.NewFuelHandler(fuelStore, deps).Register(protectedMux)
	handler.NewExpenseHandler(expenseStore, deps).Register(protectedMux)
	handler.NewRouteHandler(routeStore, deps).Register(protectedMux)
	handler.NewAPIHandler(customerStore, vehicleStore, deps).Register(protectedMux)

	// Mobile API — own mux so driver role can reach it without accessing web routes
	mobileMux := http.NewServeMux()
	mobileHandler.Register(mobileMux)

	// Accounting
	handler.NewInvoiceHandler(invoiceStore, invoiceDetailStore, paymentDetailStore, invoiceSvc, paymentStore, deps).Register(protectedMux)
	handler.NewPaymentHandler(paymentStore, paymentDetailStore, invoiceStore, paymentSvc, deps).Register(protectedMux)
	handler.NewCreditMemoHandler(creditMemoStore, deps).Register(protectedMux)
	handler.NewDamageClaimHandler(damageClaimStore, attachmentStore, storageSvc, deps).Register(protectedMux)
	handler.NewAccountsPayableHandler(apStore, deps).Register(protectedMux)
	handler.NewEarningsAdjHandler(earningsAdjStore, employeeStore, truckStore, deps).Register(protectedMux)

	// Feedback
	handler.NewFeedbackHandler(feedbackStore, attachmentStore, storageSvc, deps).Register(protectedMux)

	// Activity (super_admin only)
	handler.NewActivityHandler(activityStore, deps).Register(protectedMux, middleware.RequireRole("super_admin"))

	// Notifications
	notificationStore := store.NewNotificationStore(pool)
	handler.NewNotificationHandler(notificationStore, deps).Register(protectedMux)

	// API key-authenticated routes (feedback + activity)
	apiMux := http.NewServeMux()
	handler.NewFeedbackAPIHandler(feedbackStore, deps).Register(apiMux)
	handler.NewActivityAPIHandler(activityStore).Register(apiMux)
	apiKeyMiddleware := middleware.RequireAPIKey(apiKeyStore)
	mux.Handle("/api/feedback", apiKeyMiddleware(apiMux))
	mux.Handle("/api/feedback/", apiKeyMiddleware(apiMux))
	mux.Handle("/api/activity", apiKeyMiddleware(apiMux))
	mux.Handle("/api/activity/", apiKeyMiddleware(apiMux))

	// VIN Search + Reports
	handler.NewVinSearchHandler(vehicleStore, deps).Register(protectedMux)
	handler.NewReportHandler(orderStore, invoiceStore, tripStore, vehicleStore, paymentStore, damageClaimStore, deps).Register(protectedMux)

	// Upload handler (serve + delete routes for attachments)
	uploadHandler := handler.NewUploadHandler(attachmentStore, storageSvc, deps)
	uploadHandler.Register(protectedMux)
	uploadHandler.RegisterAdmin(protectedMux, middleware.RequireRole("super_admin"))

	// Suspended info page (GET only — accessible even when account is suspended)
	protectedMux.HandleFunc("GET /suspended", deps.SuspendedPageHandler())

	// Wrap protected routes with auth + activity tracking + CSRF + read-only-if-suspended middleware
	// Driver role is blocked from web routes — they only access /api/v1/ via the mobile app.
	authMiddleware := middleware.RequireAuth(deps.JWT, deps.SecureCookies)
	activityMw := middleware.ActivityTracker(activityStore)
	csrfMiddleware := middleware.CSRF(deps.SecureCookies)
	readOnlyGate := middleware.ReadOnlyIfSuspended(deps.IsSuspended, deps.SuspendedBlockHandler())
	blockDriver := middleware.BlockRole("driver")
	mux.Handle("/", authMiddleware(blockDriver(activityMw(csrfMiddleware(readOnlyGate(protectedMux))))))

	// Mobile API — JWT auth only, no CSRF, driver role allowed
	mux.Handle("/api/v1/", authMiddleware(mobileMux))

	migrationRunStore := store.NewMigrationRunStore(pool)
	subscriptionStore := store.NewSubscriptionStore(pool)

	// Mount Connect-RPC services (API-only, auth via interceptor — no CSRF)
	crpc.Mount(mux, crpc.MountConfig{
		JWT:                deps.JWT,
		Audit:              deps.Audit,
		CustomerStore:      customerStore,
		OrderStore:         orderStore,
		VehicleStore:       vehicleStore,
		FeedbackStore:      feedbackStore,
		EmployeeStore:      employeeStore,
		TruckStore:         truckStore,
		VendorStore:        vendorStore,
		ZoneStore:          zoneStore,
		ZonePricingStore:   zonePricingStore,
		ChargeStore:        chargeStore,
		DamageStore:        damageStore,
		NoteStore:          noteStore,
		DamageClaimStore:   damageClaimStore,
		CreditMemoStore:    creditMemoStore,
		TripStore:          tripStore,
		LoadDetailStore:    loadDetailStore,
		TripFuelStore:      fuelStore,
		TripExpenseStore:   expenseStore,
		TripRouteStore:     routeStore,
		InvoiceStore:       invoiceStore,
		InvoiceDetailStore: invoiceDetailStore,
		PaymentStore:       paymentStore,
		PaymentDetailStore: paymentDetailStore,
		APStore:            apStore,
		EarningsAdjStore:   earningsAdjStore,
		LookupStores:       lookupStoresMap,
		TermsStore:         termsStore,
		TaxCodeStore:       taxCodeStore,
		ItemStore:          itemStore,
		OrderSvc:           orderSvc,
		TripSvc:            tripSvc,
		InvoiceSvc:         invoiceSvc,
		PaymentSvc:         paymentSvc,
	})

	// OAuth2 Device Code Flow
	deviceCodeStore := store.NewDeviceCodeStore(pool)
	refreshTokenStore := store.NewRefreshTokenStore(pool)
	oauthHandler := handler.NewOAuthHandler(deviceCodeStore, refreshTokenStore, userStore, deps.JWT, deps)
	oauthHandler.RegisterPublic(mux)
	oauthHandler.RegisterProtected(protectedMux)

	rs := riverStores{
		customerStore:      customerStore,
		invoiceStore:       invoiceStore,
		invoiceDetailStore: invoiceDetailStore,
		paymentStore:       paymentStore,
		paymentDetailStore: paymentDetailStore,
		invoiceSvc:         invoiceSvc,
		migrationRunStore:  migrationRunStore,
		companyStore:       companyStore,
		userStore:          userStore,
		subscriptionStore:  subscriptionStore,
		activityStore:      activityStore,
		truckStore:         truckStore,
		apiKeyStore:        apiKeyStore,
		deviceCodeStore:    deviceCodeStore,
		refreshTokenStore:  refreshTokenStore,
		protectedMux:       protectedMux,
	}
	return mux, loadboardSvc, loadboardStore, rs
}

func initRiver(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *config.Config,
	qboStore *store.QBOStore,
	oauthCfg *oauth2.Config,
	rs riverStores,
) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()

	customerWorker := &worker.SyncCustomerWorker{
		CustomerStore: rs.customerStore,
		QBOStore:      qboStore,
		OAuthCfg:      oauthCfg,
		Sandbox:       cfg.QBOSandbox,
	}
	river.AddWorker(workers, customerWorker)

	invoiceWorker := &worker.SyncInvoiceWorker{
		InvoiceStore:       rs.invoiceStore,
		InvoiceDetailStore: rs.invoiceDetailStore,
		CustomerStore:      rs.customerStore,
		QBOStore:           qboStore,
		OAuthCfg:           oauthCfg,
		Sandbox:            cfg.QBOSandbox,
	}
	river.AddWorker(workers, invoiceWorker)

	paymentWorker := &worker.SyncPaymentWorker{
		PaymentStore:       rs.paymentStore,
		PaymentDetailStore: rs.paymentDetailStore,
		InvoiceStore:       rs.invoiceStore,
		CustomerStore:      rs.customerStore,
		QBOStore:           qboStore,
		OAuthCfg:           oauthCfg,
		Sandbox:            cfg.QBOSandbox,
	}
	river.AddWorker(workers, paymentWorker)

	migrationWorker := &worker.MigrationWorker{
		Pool:     pool,
		RunStore: rs.migrationRunStore,
		MSSQLDSN: cfg.MSSQLMigrationDSN,
	}
	river.AddWorker(workers, migrationWorker)

	cleanupWorker := &worker.ActivityCleanupWorker{
		ActivityStore: rs.activityStore,
	}
	river.AddWorker(workers, cleanupWorker)

	oauthCleanupWorker := &worker.OAuthCleanupWorker{
		DeviceCodeStore:   rs.deviceCodeStore,
		RefreshTokenStore: rs.refreshTokenStore,
	}
	river.AddWorker(workers, oauthCleanupWorker)

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Workers: workers,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 5},
			"migration":        {MaxWorkers: 1},
			"activity":         {MaxWorkers: 1},
		},
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				activityCleanupSchedule{},
				func() (river.JobArgs, *river.InsertOpts) {
					return worker.ActivityCleanupArgs{}, &river.InsertOpts{Queue: "activity"}
				},
				&river.PeriodicJobOpts{ID: "activity_cleanup", RunOnStart: false},
			),
			river.NewPeriodicJob(
				oauthCleanupSchedule{},
				func() (river.JobArgs, *river.InsertOpts) {
					return worker.OAuthCleanupArgs{}, &river.InsertOpts{Queue: "activity"}
				},
				&river.PeriodicJobOpts{ID: "oauth_cleanup", RunOnStart: true},
			),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("river new client: %w", err)
	}

	// Set RiverClient on workers that need to enqueue jobs (after NewClient, before Start).
	invoiceWorker.RiverClient = riverClient
	paymentWorker.RiverClient = riverClient

	if err := riverClient.Start(ctx); err != nil {
		return nil, fmt.Errorf("river start: %w", err)
	}

	return riverClient, nil
}

func initRiverUI(ctx context.Context, riverClient *river.Client[pgx.Tx], prefix string) (http.Handler, error) {
	endpoints := riverui.NewEndpoints(riverClient, nil)
	h, err := riverui.NewHandler(&riverui.HandlerOpts{
		Endpoints: endpoints,
		Logger:    slog.Default(),
		Prefix:    prefix,
	})
	if err != nil {
		return nil, fmt.Errorf("river ui: %w", err)
	}
	if err := h.Start(ctx); err != nil {
		return nil, fmt.Errorf("river ui start: %w", err)
	}
	return h, nil
}

func registerLookups(mux *http.ServeMux, pool *pgxpool.Pool, deps *handler.Deps) map[string]*store.LookupStore {
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
	storeMap := make(map[string]*store.LookupStore, len(lookups))
	for _, l := range lookups {
		ls, err := store.NewLookupStore(pool, l.table)
		if err != nil {
			log.Fatalf("lookup store %s: %v", l.table, err)
		}
		storeMap[l.table] = ls
		handler.NewLookupHandler(deps, ls, l.path, l.title).Register(mux)
	}
	return storeMap
}

func runServer(cfg *config.Config, handler http.Handler, ctx context.Context, riverClient *river.Client[pgx.Tx]) {
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

	if riverClient != nil {
		if err := riverClient.Stop(shutdownCtx); err != nil {
			log.Printf("river stop: %v", err)
		}
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
	log.Println("Server stopped.")
}

func runLoadboardExpiry(ctx context.Context, svc *service.LoadboardService) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := svc.ExpireListings(ctx)
			if err != nil {
				log.Printf("loadboard expiry: %v", err)
			} else if n > 0 {
				log.Printf("loadboard: expired %d listings", n)
			}
		}
	}
}

func backfillGeocode(ctx context.Context, st *store.LoadboardStore) {
	listings, err := st.ListListingsNeedingGeocode(ctx)
	if err != nil {
		log.Printf("geocode backfill: list error: %v", err)
		return
	}
	if len(listings) == 0 {
		return
	}
	log.Printf("geocode backfill: %d listings to process", len(listings))

	for i, l := range listings {
		select {
		case <-ctx.Done():
			log.Printf("geocode backfill: interrupted at %d/%d", i, len(listings))
			return
		default:
		}

		var oLat, oLng, dLat, dLng *float64

		if l.OriginLat == nil {
			oLat, oLng, err = geocode.Geocode(ctx, "", deref(l.OriginCity), deref(l.OriginState), deref(l.OriginZip))
			if err != nil {
				log.Printf("geocode backfill: origin for listing %d: %v", l.ID, err)
			}
			// Rate limit: 1 req/sec for Nominatim
			time.Sleep(1100 * time.Millisecond)
		} else {
			oLat, oLng = l.OriginLat, l.OriginLng
		}

		if l.DestLat == nil {
			dLat, dLng, err = geocode.Geocode(ctx, "", deref(l.DestCity), deref(l.DestState), deref(l.DestZip))
			if err != nil {
				log.Printf("geocode backfill: dest for listing %d: %v", l.ID, err)
			}
			time.Sleep(1100 * time.Millisecond)
		} else {
			dLat, dLng = l.DestLat, l.DestLng
		}

		if err := st.UpdateListingCoords(ctx, l.ID, oLat, oLng, dLat, dLng); err != nil {
			log.Printf("geocode backfill: update listing %d: %v", l.ID, err)
		}

		if (i+1)%10 == 0 || i == len(listings)-1 {
			log.Printf("geocode backfill: %d/%d done", i+1, len(listings))
		}
	}
	log.Printf("geocode backfill: complete")
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// activityCleanupSchedule runs the activity log cleanup once every 24 hours.
type activityCleanupSchedule struct{}

func (activityCleanupSchedule) Next(t time.Time) time.Time {
	return t.Add(24 * time.Hour)
}

// oauthCleanupSchedule runs the OAuth token cleanup every 6 hours.
type oauthCleanupSchedule struct{}

func (oauthCleanupSchedule) Next(t time.Time) time.Time {
	return t.Add(6 * time.Hour)
}
