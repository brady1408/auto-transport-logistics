package migration

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Stat holds per-table migration counts.
type Stat struct {
	Table    string
	Source   int
	Inserted int
	Skipped  int
	Elapsed  time.Duration
}

// Logger is a function the caller provides to receive progress lines.
type Logger func(line string)

// RunAll runs all 4 migration phases for the given companyID.
// logger receives one formatted line per table.
// Returns per-table stats.
func RunAll(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, logger Logger) ([]Stat, error) {
	var stats []Stat
	log := func(s Stat) {
		stats = append(stats, s)
		logger(fmt.Sprintf("[%s] src=%-6d ins=%-6d skip=%-4d  %v",
			s.Table, s.Source, s.Inserted, s.Skipped, s.Elapsed.Round(time.Millisecond)))
	}

	// Phase 1: Lookup tables
	logger("=== Phase 1: Lookup Tables ===")
	companyIDs := migrateCompanies(ctx, src, tx, companyID, log)
	migrateZones(ctx, src, tx, companyID, log)
	migrateRegions(ctx, src, tx, companyID, log)
	migrateDispatchCodes(ctx, src, tx, companyID, log)
	migrateEquipmentTypes(ctx, src, tx, companyID, log)
	migrateItems(ctx, src, tx, companyID, log)
	migrateVehicleMakes(ctx, src, tx, companyID, log)
	migrateVinDefinitions(ctx, src, tx, companyID, log)
	migrateColorCodes(ctx, src, tx, companyID, log)
	migrateHoldCodes(ctx, src, tx, companyID, log)
	migrateDeclinationCodes(ctx, src, tx, companyID, log)
	migrateFieldCodes(ctx, src, tx, companyID, "G65", "field_codes_1", log)
	migrateFieldCodes(ctx, src, tx, companyID, "G66", "field_codes_2", log)
	migrateFieldCodes(ctx, src, tx, companyID, "G67", "field_codes_3", log)
	migrateFieldCodes(ctx, src, tx, companyID, "G68", "field_codes_4", log)
	migrateFieldCodes(ctx, src, tx, companyID, "G69", "field_codes_5", log)
	migrateDamageAreas(ctx, src, tx, companyID, log)
	migrateDamageTypes(ctx, src, tx, companyID, log)
	migrateDamageSeverities(ctx, src, tx, companyID, log)
	migrateChartOfAccounts(ctx, src, tx, companyID, log)
	migrateTerms(ctx, src, tx, companyID, log)
	migrateTaxCodes(ctx, src, tx, companyID, log)
	migrateVendors(ctx, src, tx, companyID, log)
	migrateVendorGroups(ctx, src, tx, companyID, log)
	migrateCarriers(ctx, src, tx, companyID, log)
	migrateZonePricing(ctx, src, tx, companyID, log)
	_ = companyIDs

	// Phase 2: Core entities
	logger("=== Phase 2: Core Entities ===")
	customerIDs := migrateCustomers(ctx, src, tx, companyID, log)
	employeeIDs := migrateEmployees(ctx, src, tx, companyID, log)
	truckIDs := migrateTrucks(ctx, src, tx, companyID, log)

	// Phase 3: Dispatch
	logger("=== Phase 3: Dispatch ===")
	orderIDs := migrateOrders(ctx, src, tx, companyID, customerIDs, log)
	tripIDs := migrateTrips(ctx, src, tx, companyID, truckIDs, employeeIDs, log)
	vehicleIDs := migrateOrderVehicles(ctx, src, tx, companyID, orderIDs, tripIDs, log)
	migrateLoadDetails(ctx, src, tx, companyID, tripIDs, orderIDs, vehicleIDs, log)
	migrateOrderCharges(ctx, src, tx, companyID, orderIDs, vehicleIDs, tripIDs, log)
	damageIDs := migrateVehicleDamage(ctx, src, tx, companyID, orderIDs, vehicleIDs, tripIDs, log)
	migrateDamageDetails(ctx, src, tx, companyID, damageIDs, log)
	migrateVehicleNotes(ctx, src, tx, companyID, vehicleIDs, log)
	migrateTripFuel(ctx, src, tx, companyID, tripIDs, log)
	migrateTripExpenses(ctx, src, tx, companyID, tripIDs, log)
	migrateTripRoutes(ctx, src, tx, companyID, tripIDs, customerIDs, log)
	migrateSplitLoads(ctx, src, tx, companyID, orderIDs, vehicleIDs, tripIDs, log)

	// Phase 4: Accounting
	logger("=== Phase 4: Accounting ===")
	invoiceIDs := migrateInvoices(ctx, src, tx, companyID, customerIDs, orderIDs, log)
	migrateInvoiceDetails(ctx, src, tx, companyID, invoiceIDs, orderIDs, vehicleIDs, log)
	migrateCreditMemos(ctx, src, tx, companyID, customerIDs, invoiceIDs, log)
	paymentIDs := migratePayments(ctx, src, tx, companyID, customerIDs, log)
	migratePaymentDetails(ctx, src, tx, companyID, paymentIDs, invoiceIDs, log)
	migrateDamageClaims(ctx, src, tx, companyID, orderIDs, vehicleIDs, tripIDs, log)
	migrateAccountsPayable(ctx, src, tx, companyID, tripIDs, employeeIDs, truckIDs, log)

	return stats, nil
}

// ResetSequences bumps all PG sequences to MAX(id)+1 after bulk insert.
func ResetSequences(ctx context.Context, tx pgx.Tx) {
	tables := []string{
		"companies", "customers", "employees", "trucks", "zones", "zone_pricing",
		"vendors", "vendor_groups", "carriers", "regions", "dispatch_codes",
		"equipment_types", "items", "vehicle_makes", "vin_definitions", "color_codes",
		"hold_codes", "declination_codes", "field_codes_1", "field_codes_2",
		"field_codes_3", "field_codes_4", "field_codes_5", "damage_areas", "damage_types",
		"damage_severities", "terms", "tax_codes", "chart_of_accounts",
		"orders", "trips", "order_vehicles", "load_details", "order_charges",
		"vehicle_damage", "damage_details", "vehicle_notes", "trip_fuel", "trip_expenses",
		"trip_routes", "split_loads", "invoices", "invoice_details", "credit_memos",
		"payments", "payment_details", "damage_claims", "accounts_payable",
	}
	for _, t := range tables {
		q := fmt.Sprintf("SELECT setval('%s_id_seq', COALESCE((SELECT MAX(id) FROM %s), 0) + 1, false)", t, t)
		_, _ = tx.Exec(ctx, q)
	}
}
