package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jackc/pgx/v5"
)

// IDMap stores old MSSQL PK → new PostgreSQL PK
type IDMap map[int]int

// --- Nullable scan helpers for MSSQL (native types, no Clarion conversion) ---

// ns scans sql.NullString → *string (nil if NULL or empty)
func ns(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := strings.TrimRight(v.String, "\x00")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// nns scans sql.NullString → string (empty if NULL)
func nns(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	s := strings.TrimRight(v.String, "\x00")
	return strings.TrimSpace(s)
}

// nt scans sql.NullTime → *time.Time (nil if NULL or zero)
func nt(v sql.NullTime) *time.Time {
	if !v.Valid || v.Time.IsZero() {
		return nil
	}
	t := v.Time
	return &t
}

// ni scans sql.NullInt64 → *int (nil if NULL or 0)
func ni(v sql.NullInt64) *int {
	if !v.Valid || v.Int64 == 0 {
		return nil
	}
	i := int(v.Int64)
	return &i
}

// nint scans sql.NullInt64 → int (0 if NULL)
func nint(v sql.NullInt64) int {
	if !v.Valid {
		return 0
	}
	return int(v.Int64)
}

// nd scans sql.NullFloat64 → *float64 (nil if NULL or 0)
func nd(v sql.NullFloat64) *float64 {
	if !v.Valid || v.Float64 == 0 {
		return nil
	}
	return &v.Float64
}

// nb scans sql.NullInt64 (tinyint) → bool
func nb(v sql.NullInt64) bool {
	return v.Valid && v.Int64 != 0
}

// lookupFK resolves old FK → new PG id via IDMap
func lookupFK(m IDMap, v sql.NullInt64) *int {
	if !v.Valid || v.Int64 == 0 {
		return nil
	}
	if newID, ok := m[int(v.Int64)]; ok {
		return &newID
	}
	return nil
}

// tableExists checks if a table exists in MSSQL
func tableExists(src *sql.DB, table string) bool {
	var count int
	err := src.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @p1", table).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// summary tracks migration stats
type migrationSummary struct {
	table    string
	source   int
	inserted int
	skipped  int
	elapsed  time.Duration
}

var summaries []migrationSummary

func logTable(table string, source, inserted, skipped int, elapsed time.Duration) {
	summaries = append(summaries, migrationSummary{table, source, inserted, skipped, elapsed})
	log.Printf("  %-25s src=%-6d ins=%-6d skip=%-4d  %v", table, source, inserted, skipped, elapsed.Round(time.Millisecond))
}

func main() {
	mssqlDSN := flag.String("mssql", "sqlserver://sa:aurorasga10*@192.168.23.44:1433?database=Demo&encrypt=disable", "MSSQL connection string")
	pgDSN := flag.String("pg", "postgres://atlinks:atlinks_dev@localhost:5432/atlinks", "PostgreSQL connection string")
	flag.Parse()

	ctx := context.Background()

	// Connect to MSSQL
	log.Println("Connecting to MSSQL...")
	src, err := sql.Open("sqlserver", *mssqlDSN)
	if err != nil {
		log.Fatalf("MSSQL connect: %v", err)
	}
	defer src.Close()
	if err := src.PingContext(ctx); err != nil {
		log.Fatalf("MSSQL ping: %v", err)
	}
	log.Println("MSSQL connected.")

	// Connect to PostgreSQL
	log.Println("Connecting to PostgreSQL...")
	dst, err := pgx.Connect(ctx, *pgDSN)
	if err != nil {
		log.Fatalf("PostgreSQL connect: %v", err)
	}
	defer dst.Close(ctx)
	log.Println("PostgreSQL connected.")

	// Start transaction
	tx, err := dst.Begin(ctx)
	if err != nil {
		log.Fatalf("Begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Truncate all tables in reverse dependency order
	log.Println("Truncating all tables...")
	truncateAll(ctx, tx)

	start := time.Now()

	// Phase 1: Lookup tables
	log.Println("\n=== Phase 1: Lookup Tables ===")
	companyIDs := migrateCompanies(ctx, src, tx)
	migrateZones(ctx, src, tx)
	migrateRegions(ctx, src, tx)
	migrateDispatchCodes(ctx, src, tx)
	migrateEquipmentTypes(ctx, src, tx)
	migrateItems(ctx, src, tx)
	migrateVehicleMakes(ctx, src, tx)
	migrateVinDefinitions(ctx, src, tx)
	migrateColorCodes(ctx, src, tx)
	migrateHoldCodes(ctx, src, tx)
	migrateDeclinationCodes(ctx, src, tx)
	migrateFieldCodes(ctx, src, tx, "G65", "field_codes_1")
	migrateFieldCodes(ctx, src, tx, "G66", "field_codes_2")
	migrateFieldCodes(ctx, src, tx, "G67", "field_codes_3")
	migrateFieldCodes(ctx, src, tx, "G68", "field_codes_4")
	migrateFieldCodes(ctx, src, tx, "G69", "field_codes_5")
	migrateDamageAreas(ctx, src, tx)
	migrateDamageTypes(ctx, src, tx)
	migrateDamageSeverities(ctx, src, tx)
	migrateChartOfAccounts(ctx, src, tx)
	migrateTerms(ctx, src, tx)
	migrateTaxCodes(ctx, src, tx)
	migrateVendors(ctx, src, tx)
	migrateVendorGroups(ctx, src, tx)
	migrateCarriers(ctx, src, tx)
	migrateZonePricing(ctx, src, tx)

	// Phase 2: Core entities
	log.Println("\n=== Phase 2: Core Entities ===")
	customerIDs := migrateCustomers(ctx, src, tx)
	employeeIDs := migrateEmployees(ctx, src, tx)
	truckIDs := migrateTrucks(ctx, src, tx)

	// Phase 3: Dispatch
	log.Println("\n=== Phase 3: Dispatch Tables ===")
	orderIDs := migrateOrders(ctx, src, tx, customerIDs)
	tripIDs := migrateTrips(ctx, src, tx, truckIDs, employeeIDs)
	vehicleIDs := migrateOrderVehicles(ctx, src, tx, orderIDs, tripIDs)
	migrateLoadDetails(ctx, src, tx, tripIDs, orderIDs, vehicleIDs)
	migrateOrderCharges(ctx, src, tx, orderIDs, vehicleIDs, tripIDs)
	damageIDs := migrateVehicleDamage(ctx, src, tx, orderIDs, vehicleIDs, tripIDs)
	migrateDamageDetails(ctx, src, tx, damageIDs)
	migrateVehicleNotes(ctx, src, tx, vehicleIDs)
	migrateTripFuel(ctx, src, tx, tripIDs)
	migrateTripExpenses(ctx, src, tx, tripIDs)
	migrateTripRoutes(ctx, src, tx, tripIDs, customerIDs)
	migrateSplitLoads(ctx, src, tx, orderIDs, vehicleIDs, tripIDs)

	// Phase 4: Accounting
	log.Println("\n=== Phase 4: Accounting Tables ===")
	invoiceIDs := migrateInvoices(ctx, src, tx, customerIDs, orderIDs)
	migrateInvoiceDetails(ctx, src, tx, invoiceIDs, orderIDs, vehicleIDs)
	migrateCreditMemos(ctx, src, tx, customerIDs, invoiceIDs)
	paymentIDs := migratePayments(ctx, src, tx, customerIDs)
	migratePaymentDetails(ctx, src, tx, paymentIDs, invoiceIDs)
	migrateDamageClaims(ctx, src, tx, orderIDs, vehicleIDs, tripIDs)
	migrateAccountsPayable(ctx, src, tx, tripIDs, employeeIDs, truckIDs)

	// Reset sequences
	log.Println("\n=== Resetting Sequences ===")
	resetSequences(ctx, tx)

	// Commit
	log.Println("\nCommitting transaction...")
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("Commit failed: %v", err)
	}

	elapsed := time.Since(start)
	log.Printf("\n=== Migration Complete in %v ===", elapsed.Round(time.Millisecond))
	log.Printf("%-25s %8s %8s %8s", "TABLE", "SOURCE", "INSERT", "SKIP")
	log.Printf("%s", strings.Repeat("-", 55))
	totalSrc, totalIns, totalSkip := 0, 0, 0
	for _, s := range summaries {
		log.Printf("%-25s %8d %8d %8d", s.table, s.source, s.inserted, s.skipped)
		totalSrc += s.source
		totalIns += s.inserted
		totalSkip += s.skipped
	}
	log.Printf("%s", strings.Repeat("-", 55))
	log.Printf("%-25s %8d %8d %8d", "TOTAL", totalSrc, totalIns, totalSkip)

	_ = companyIDs // used if needed later
}

func truncateAll(ctx context.Context, tx pgx.Tx) {
	tables := []string{
		"audit_log", "accounts_payable", "damage_claims", "payment_details", "payments",
		"credit_memos", "invoice_details", "invoices", "split_loads", "trip_routes",
		"trip_expenses", "trip_fuel", "vehicle_notes", "damage_details", "vehicle_damage",
		"order_charges", "load_details", "order_vehicles", "trips", "orders",
		"chart_of_accounts", "tax_codes", "terms", "damage_severities", "damage_types",
		"damage_areas", "field_codes_5", "field_codes_4", "field_codes_3", "field_codes_2",
		"field_codes_1", "declination_codes", "hold_codes", "color_codes", "vin_definitions",
		"vehicle_makes", "items", "equipment_types", "dispatch_codes", "regions", "carriers",
		"vendor_groups", "vendors", "zone_pricing", "zones", "trailers", "trucks",
		"employees", "customers", "companies",
	}
	for _, t := range tables {
		if _, err := tx.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", t)); err != nil {
			log.Fatalf("Truncate %s: %v", t, err)
		}
	}
}

func resetSequences(ctx context.Context, tx pgx.Tx) {
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
		if _, err := tx.Exec(ctx, q); err != nil {
			log.Printf("  WARN: reset seq %s: %v", t, err)
		}
	}
}
