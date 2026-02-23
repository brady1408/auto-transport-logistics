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
	"syscall"
	"time"

	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/config"
	"github.com/brady1408/atlinks/internal/database"
	"github.com/brady1408/atlinks/internal/handler"
	"github.com/brady1408/atlinks/internal/middleware"
	"github.com/brady1408/atlinks/internal/service"
	"github.com/brady1408/atlinks/internal/store"
)

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

	// Services
	jwtSvc := auth.NewJWTService(cfg.JWTSecret)
	auditSvc := audit.NewService(pool)

	// Parse templates
	tmpl, err := handler.ParseTemplates()
	if err != nil {
		log.Fatalf("parse templates: %v", err)
	}

	deps := &handler.Deps{
		JWT:   jwtSvc,
		Audit: auditSvc,
		Tmpl:  tmpl,
	}

	// Stores
	userStore := store.NewUserStore(pool)
	customerStore := store.NewCustomerStore(pool)
	employeeStore := store.NewEmployeeStore(pool)
	truckStore := store.NewTruckStore(pool)
	zoneStore := store.NewZoneStore(pool)
	zonePricingStore := store.NewZonePricingStore(pool)
	companyStore := store.NewCompanyStore(pool)

	// Phase 2 stores
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

	// Phase 3 stores (Accounting)
	invoiceStore := store.NewInvoiceStore(pool)
	invoiceDetailStore := store.NewInvoiceDetailStore(pool)
	paymentStore := store.NewPaymentStore(pool)
	paymentDetailStore := store.NewPaymentDetailStore(pool)
	creditMemoStore := store.NewCreditMemoStore(pool)
	damageClaimStore := store.NewDamageClaimStore(pool)
	apStore := store.NewAccountsPayableStore(pool)

	// Phase 2 services
	orderSvc := service.NewOrderService(pool, orderStore, vehicleStore, auditSvc)
	tripSvc := service.NewTripService(pool, tripStore, loadDetailStore, vehicleStore, orderStore, auditSvc)

	// Phase 3 services (Accounting)
	invoiceSvc := service.NewInvoiceService(pool, invoiceStore, invoiceDetailStore, orderStore, vehicleStore, auditSvc)
	paymentSvc := service.NewPaymentService(pool, paymentStore, paymentDetailStore, invoiceStore, auditSvc)

	// Mux
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

	// Auth routes (public)
	authHandler := handler.NewAuthHandler(userStore, deps)
	authHandler.Register(mux)

	// Protected routes
	protectedMux := http.NewServeMux()

	// Dashboard
	dashHandler := handler.NewDashboardHandler(orderStore, invoiceStore, tripStore, deps)
	dashHandler.Register(protectedMux)

	// Customer CRUD
	custHandler := handler.NewCustomerHandler(customerStore, deps)
	custHandler.Register(protectedMux)

	// Employee CRUD
	empHandler := handler.NewEmployeeHandler(employeeStore, deps)
	empHandler.Register(protectedMux)

	// Truck CRUD
	truckHandler := handler.NewTruckHandler(truckStore, deps)
	truckHandler.Register(protectedMux)

	// Zone + Zone Pricing CRUD
	zoneHandler := handler.NewZoneHandler(zoneStore, zonePricingStore, deps)
	zoneHandler.Register(protectedMux)

	// Company settings
	companyHandler := handler.NewCompanyHandler(companyStore, deps)
	companyHandler.Register(protectedMux)

	// Lookup tables (generic code+description)
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
		lh := handler.NewLookupHandler(deps, ls, l.path, l.title)
		lh.Register(protectedMux)
	}

	// Terms, Tax Codes, Items (extra-field lookup tables)
	termsStore := store.NewTermsStore(pool)
	termsHandler := handler.NewTermsHandler(termsStore, deps)
	termsHandler.Register(protectedMux)

	taxCodeStore := store.NewTaxCodeStore(pool)
	taxCodeHandler := handler.NewTaxCodeHandler(taxCodeStore, deps)
	taxCodeHandler.Register(protectedMux)

	itemStore := store.NewItemStore(pool)
	itemHandler := handler.NewItemHandler(itemStore, deps)
	itemHandler.Register(protectedMux)

	// Phase 2: Dispatch
	orderHandler := handler.NewOrderHandler(orderStore, customerStore, orderSvc, invoiceSvc, deps)
	orderHandler.Register(protectedMux)

	vehicleHandler := handler.NewVehicleHandler(vehicleStore, orderStore, orderSvc, deps)
	vehicleHandler.Register(protectedMux)

	tripHandler := handler.NewTripHandler(tripStore, loadDetailStore, vehicleStore, tripSvc, deps)
	tripHandler.Register(protectedMux)

	chargeHandler := handler.NewChargeHandler(chargeStore, deps)
	chargeHandler.Register(protectedMux)

	damageHandler := handler.NewDamageHandler(damageStore, deps)
	damageHandler.Register(protectedMux)

	noteHandler := handler.NewNoteHandler(noteStore, deps)
	noteHandler.Register(protectedMux)

	fuelHandler := handler.NewFuelHandler(fuelStore, deps)
	fuelHandler.Register(protectedMux)

	expenseHandler := handler.NewExpenseHandler(expenseStore, deps)
	expenseHandler.Register(protectedMux)

	routeHandler := handler.NewRouteHandler(routeStore, deps)
	routeHandler.Register(protectedMux)

	apiHandler := handler.NewAPIHandler(customerStore, vehicleStore, deps)
	apiHandler.Register(protectedMux)

	// Phase 3: Accounting
	invoiceHandler := handler.NewInvoiceHandler(invoiceStore, invoiceDetailStore, paymentDetailStore, invoiceSvc, deps)
	invoiceHandler.Register(protectedMux)

	paymentHandler := handler.NewPaymentHandler(paymentStore, paymentDetailStore, invoiceStore, paymentSvc, deps)
	paymentHandler.Register(protectedMux)

	creditMemoHandler := handler.NewCreditMemoHandler(creditMemoStore, deps)
	creditMemoHandler.Register(protectedMux)

	damageClaimHandler := handler.NewDamageClaimHandler(damageClaimStore, deps)
	damageClaimHandler.Register(protectedMux)

	apHandler := handler.NewAccountsPayableHandler(apStore, deps)
	apHandler.Register(protectedMux)

	// Phase 4: VIN Search + Reports
	vinSearchHandler := handler.NewVinSearchHandler(vehicleStore, deps)
	vinSearchHandler.Register(protectedMux)

	reportHandler := handler.NewReportHandler(orderStore, invoiceStore, tripStore, vehicleStore, paymentStore, damageClaimStore, deps)
	reportHandler.Register(protectedMux)

	// Wrap protected routes with auth middleware
	authMiddleware := middleware.RequireAuth(jwtSvc)
	mux.Handle("/", authMiddleware(protectedMux))

	// Apply logging middleware to all routes
	var httpHandler http.Handler = mux
	httpHandler = middleware.RequestLogger(httpHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      httpHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
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
