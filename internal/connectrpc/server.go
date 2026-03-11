package connectrpc

import (
	"net/http"

	"connectrpc.com/connect"
	"github.com/brady1408/atlinks/internal/audit"
	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/gen/atlinks/v1/atlinkspbconnect"
	"github.com/brady1408/atlinks/internal/service"
	"github.com/brady1408/atlinks/internal/store"
)

// MountConfig holds all dependencies for Connect-RPC service handlers.
type MountConfig struct {
	JWT   *auth.JWTService
	Audit *audit.Service

	// Stores
	CustomerStore     *store.CustomerStore
	OrderStore        *store.OrderStore
	VehicleStore      *store.VehicleStore
	FeedbackStore     *store.FeedbackStore
	EmployeeStore     *store.EmployeeStore
	TruckStore        *store.TruckStore
	VendorStore       *store.VendorStore
	ZoneStore         *store.ZoneStore
	ZonePricingStore  *store.ZonePricingStore
	ChargeStore       *store.ChargeStore
	DamageStore       *store.DamageStore
	NoteStore         *store.NoteStore
	DamageClaimStore  *store.DamageClaimStore
	CreditMemoStore   *store.CreditMemoStore
	TripStore         *store.TripStore
	LoadDetailStore   *store.LoadDetailStore
	TripFuelStore     *store.TripFuelStore
	TripExpenseStore  *store.TripExpenseStore
	TripRouteStore    *store.TripRouteStore
	InvoiceStore      *store.InvoiceStore
	InvoiceDetailStore *store.InvoiceDetailStore
	PaymentStore      *store.PaymentStore
	PaymentDetailStore *store.PaymentDetailStore
	APStore           *store.AccountsPayableStore
	EarningsAdjStore  *store.EarningsAdjStore
	LookupStores      map[string]*store.LookupStore
	TermsStore        *store.TermsStore
	TaxCodeStore      *store.TaxCodeStore
	ItemStore         *store.ItemStore

	// Services
	OrderSvc   *service.OrderService
	TripSvc    *service.TripService
	InvoiceSvc *service.InvoiceService
	PaymentSvc *service.PaymentService
}

// Mount registers all Connect-RPC service handlers onto the given mux.
// Auth is enforced via the interceptor — no CSRF middleware needed (API-only).
func Mount(mux *http.ServeMux, cfg MountConfig) {
	interceptors := connect.WithInterceptors(AuthInterceptor(cfg.JWT))

	path, handler := atlinkspbconnect.NewCustomerServiceHandler(
		NewCustomerServer(cfg.CustomerStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewOrderServiceHandler(
		NewOrderServer(cfg.OrderStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewVehicleServiceHandler(
		NewVehicleServer(cfg.VehicleStore, cfg.OrderSvc, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewFeedbackServiceHandler(
		NewFeedbackServer(cfg.FeedbackStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewEmployeeServiceHandler(
		NewEmployeeServer(cfg.EmployeeStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewTruckServiceHandler(
		NewTruckServer(cfg.TruckStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewVendorServiceHandler(
		NewVendorServer(cfg.VendorStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewZoneServiceHandler(
		NewZoneServer(cfg.ZoneStore, cfg.ZonePricingStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewChargeServiceHandler(
		NewChargeServer(cfg.ChargeStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewDamageServiceHandler(
		NewDamageServer(cfg.DamageStore, cfg.NoteStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewDamageClaimServiceHandler(
		NewDamageClaimServer(cfg.DamageClaimStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewCreditMemoServiceHandler(
		NewCreditMemoServer(cfg.CreditMemoStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewTripServiceHandler(
		NewTripServer(cfg.TripStore, cfg.LoadDetailStore, cfg.TripFuelStore, cfg.TripExpenseStore, cfg.TripRouteStore, cfg.TripSvc, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewInvoiceServiceHandler(
		NewInvoiceServer(cfg.InvoiceStore, cfg.InvoiceDetailStore, cfg.InvoiceSvc, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewPaymentServiceHandler(
		NewPaymentServer(cfg.PaymentStore, cfg.PaymentDetailStore, cfg.PaymentSvc, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewAPServiceHandler(
		NewAPServer(cfg.APStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewEarningsServiceHandler(
		NewEarningsServer(cfg.EarningsAdjStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)

	path, handler = atlinkspbconnect.NewLookupServiceHandler(
		NewLookupServer(cfg.LookupStores, cfg.TermsStore, cfg.TaxCodeStore, cfg.ItemStore, cfg.Audit),
		interceptors,
	)
	mux.Handle(path, handler)
}
