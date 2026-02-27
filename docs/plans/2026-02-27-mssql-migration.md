# MSSQL Migration Feature Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow a super_admin to upload a legacy ATLinks `.bak` file, assign it to a company, and watch it migrate automatically via a River background job.

**Architecture:** A dedicated `migration` River queue (separate from QBO's `default`) restores the `.bak` to a temporary MSSQL database, runs all migration phases scoped to the target `company_id`, drops the temp DB, and streams progress to the admin UI via HTMX polling. Migration logic is extracted from `scripts/` into `internal/migration/` so both the CLI tool and worker share the same code.

**Tech Stack:** Go 1.22+, River (riverqueue/river), pgx/v5, go-mssqldb, templ, HTMX, Docker named volume for shared `.bak` files between ATLinks and MSSQL containers.

---

## Task 1: Database Migration — `migration_runs` table

**Files:**
- Create: `internal/database/migrations/027_migration_runs.sql`

**Step 1: Create the migration file**

```sql
-- +goose Up
CREATE TABLE migration_runs (
    id              BIGSERIAL PRIMARY KEY,
    company_id      BIGINT NOT NULL REFERENCES companies(id),
    status          TEXT NOT NULL DEFAULT 'pending',
    backup_filename TEXT NOT NULL,
    log             TEXT NOT NULL DEFAULT '',
    stats           JSONB,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS migration_runs;
```

**Step 2: Apply the migration**

```bash
make migrate-up
```
Expected: `OK   027_migration_runs.sql`

**Step 3: Commit**

```bash
git add internal/database/migrations/027_migration_runs.sql
git commit -m "feat: add migration_runs table"
```

---

## Task 2: Model and Store

**Files:**
- Create: `internal/models/migration_run.go`
- Create: `internal/store/migration_run_store.go`

**Step 1: Write the model**

```go
// internal/models/migration_run.go
package models

import "time"

type MigrationRun struct {
	ID             int64
	CompanyID      int64
	CompanyName    string // joined
	Status         string // pending | running | complete | failed
	BackupFilename string
	Log            string
	Stats          []MigrationTableStat
	StartedAt      *time.Time
	FinishedAt     *time.Time
	CreatedAt      time.Time
}

type MigrationTableStat struct {
	Table    string `json:"table"`
	Source   int    `json:"source"`
	Inserted int    `json:"inserted"`
	Skipped  int    `json:"skipped"`
}
```

**Step 2: Write the store**

```go
// internal/store/migration_run_store.go
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MigrationRunStore struct {
	pool *pgxpool.Pool
}

func NewMigrationRunStore(pool *pgxpool.Pool) *MigrationRunStore {
	return &MigrationRunStore{pool: pool}
}

func (s *MigrationRunStore) Create(ctx context.Context, companyID int64, backupFilename string) (*models.MigrationRun, error) {
	var run models.MigrationRun
	err := s.pool.QueryRow(ctx, `
		INSERT INTO migration_runs (company_id, backup_filename)
		VALUES ($1, $2)
		RETURNING id, company_id, status, backup_filename, log, started_at, finished_at, created_at`,
		companyID, backupFilename,
	).Scan(&run.ID, &run.CompanyID, &run.Status, &run.BackupFilename,
		&run.Log, &run.StartedAt, &run.FinishedAt, &run.CreatedAt)
	return &run, err
}

func (s *MigrationRunStore) Get(ctx context.Context, id int64) (*models.MigrationRun, error) {
	var run models.MigrationRun
	var statsJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT r.id, r.company_id, COALESCE(c.company_name,''), r.status,
		       r.backup_filename, r.log, r.stats, r.started_at, r.finished_at, r.created_at
		FROM migration_runs r
		LEFT JOIN companies c ON c.id = r.company_id
		WHERE r.id = $1`, id,
	).Scan(&run.ID, &run.CompanyID, &run.CompanyName, &run.Status,
		&run.BackupFilename, &run.Log, &statsJSON, &run.StartedAt, &run.FinishedAt, &run.CreatedAt)
	if err != nil {
		return nil, err
	}
	if statsJSON != nil {
		_ = json.Unmarshal(statsJSON, &run.Stats)
	}
	return &run, nil
}

func (s *MigrationRunStore) List(ctx context.Context) ([]models.MigrationRun, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.company_id, COALESCE(c.company_name,''), r.status,
		       r.backup_filename, r.log, r.started_at, r.finished_at, r.created_at
		FROM migration_runs r
		LEFT JOIN companies c ON c.id = r.company_id
		ORDER BY r.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []models.MigrationRun
	for rows.Next() {
		var run models.MigrationRun
		if err := rows.Scan(&run.ID, &run.CompanyID, &run.CompanyName, &run.Status,
			&run.BackupFilename, &run.Log, &run.StartedAt, &run.FinishedAt, &run.CreatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (s *MigrationRunStore) SetRunning(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE migration_runs SET status = 'running', started_at = $2 WHERE id = $1`,
		id, now)
	return err
}

func (s *MigrationRunStore) AppendLog(ctx context.Context, id int64, line string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE migration_runs SET log = log || $2 WHERE id = $1`,
		id, line+"\n")
	return err
}

func (s *MigrationRunStore) Complete(ctx context.Context, id int64, stats []models.MigrationTableStat) error {
	statsJSON, _ := json.Marshal(stats)
	now := time.Now()
	_, err := s.pool.Exec(ctx,
		`UPDATE migration_runs SET status = 'complete', finished_at = $2, stats = $3 WHERE id = $1`,
		id, now, statsJSON)
	return err
}

func (s *MigrationRunStore) Fail(ctx context.Context, id int64, reason string) error {
	now := time.Now()
	_, err := s.pool.Exec(ctx, `
		UPDATE migration_runs
		SET status = 'failed', finished_at = $2,
		    log = log || $3
		WHERE id = $1`,
		id, now, fmt.Sprintf("\nFAILED: %s\n", reason))
	return err
}

func (s *MigrationRunStore) GetLog(ctx context.Context, id int64) (string, string, error) {
	var log, status string
	err := s.pool.QueryRow(ctx,
		`SELECT log, status FROM migration_runs WHERE id = $1`, id,
	).Scan(&log, &status)
	return log, status, err
}
```

**Step 3: Verify it compiles**

```bash
go build ./internal/...
```
Expected: no errors

**Step 4: Commit**

```bash
git add internal/models/migration_run.go internal/store/migration_run_store.go
git commit -m "feat: add MigrationRun model and store"
```

---

## Task 3: Extract Migration Logic to `internal/migration/`

The scripts in `scripts/` are `package main` and can't be imported. Extract shared logic to a proper package so both the CLI and the River worker can use it.

**Files:**
- Create: `internal/migration/helpers.go` — IDMap type + all null-scan helpers
- Create: `internal/migration/phase1.go` — lookup tables
- Create: `internal/migration/phase2.go` — customers, employees, trucks
- Create: `internal/migration/phase3.go` — orders, vehicles, trips, dispatch children
- Create: `internal/migration/phase4.go` — invoices, payments, accounting
- Create: `internal/migration/runner.go` — RunAll orchestrator + Stat type

**Step 1: Create helpers.go**

Copy the helper functions from `scripts/migrate_mssql.go` into the new package with `package migration`. Functions: `IDMap`, `ns`, `nns`, `nt`, `ni`, `nint`, `nd`, `nb`, `lookupFK`.

```go
// internal/migration/helpers.go
package migration

import (
	"database/sql"
	"strings"
	"time"
)

// IDMap stores old MSSQL PK → new PostgreSQL PK.
type IDMap map[int]int

func ns(v sql.NullString) *string {
	if !v.Valid { return nil }
	s := strings.TrimRight(v.String, "\x00")
	s = strings.TrimSpace(s)
	if s == "" { return nil }
	return &s
}

func nns(v sql.NullString) string {
	if !v.Valid { return "" }
	return strings.TrimSpace(strings.TrimRight(v.String, "\x00"))
}

func nt(v sql.NullTime) *time.Time {
	if !v.Valid || v.Time.IsZero() { return nil }
	t := v.Time
	return &t
}

func ni(v sql.NullInt64) *int {
	if !v.Valid || v.Int64 == 0 { return nil }
	i := int(v.Int64)
	return &i
}

func nint(v sql.NullInt64) int {
	if !v.Valid { return 0 }
	return int(v.Int64)
}

func nd(v sql.NullFloat64) *float64 {
	if !v.Valid || v.Float64 == 0 { return nil }
	return &v.Float64
}

func nb(v sql.NullInt64) bool { return v.Valid && v.Int64 != 0 }

func lookupFK(m IDMap, v sql.NullInt64) *int {
	if !v.Valid || v.Int64 == 0 { return nil }
	if newID, ok := m[int(v.Int64)]; ok { return &newID }
	return nil
}

func tableExists(src *sql.DB, table string) bool {
	var count int
	err := src.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @p1", table).Scan(&count)
	return err == nil && count > 0
}
```

**Step 2: Create runner.go — Stat type + RunAll**

```go
// internal/migration/runner.go
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
	// ... all phase 1 functions
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
```

**Step 3: Migrate phase functions from scripts/**

Copy each `migrate*` function from `scripts/migrate_phase1.go` through `scripts/migrate_phase4.go` into the corresponding `internal/migration/phase*.go` files. Change `package main` → `package migration`. Add `companyID int` parameter to every function. Change every INSERT to include `, company_id` and `$N` placeholder for `companyID`. Remove the global `summaries` slice and `logTable()` — replace with the `log func(Stat)` parameter.

Example diff for one function in phase1.go:
```go
// Before (in scripts/migrate_phase1.go):
func migrateZones(ctx context.Context, src *sql.DB, tx pgx.Tx) {
    ...
    tx.QueryRow(ctx, `INSERT INTO zones (legacy_id, zone_name) VALUES ($1,$2) RETURNING id`, ...)
    ...
    logTable("zones", srcCount, insCount, skipCount, time.Since(t))
}

// After (in internal/migration/phase1.go):
func migrateZones(ctx context.Context, src *sql.DB, tx pgx.Tx, companyID int, log func(Stat)) {
    ...
    tx.QueryRow(ctx, `INSERT INTO zones (legacy_id, zone_name, company_id) VALUES ($1,$2,$3) RETURNING id`, ..., companyID)
    ...
    log(Stat{Table: "zones", Source: srcCount, Inserted: insCount, Skipped: skipCount, Elapsed: time.Since(t)})
}
```

**Step 4: Verify it compiles**

```bash
go build ./internal/migration/...
```
Expected: no errors

**Step 5: Update scripts/migrate_mssql.go to use internal/migration**

Replace all inline phase calls and helper functions with imports from `internal/migration`. Add a `--company-id` flag. Remove `truncateAll()`. The new main() should look like:

```go
func main() {
    mssqlDSN := flag.String("mssql", "sqlserver://sa:...", "MSSQL DSN")
    pgDSN := flag.String("pg", "postgres://...", "PostgreSQL DSN")
    companyID := flag.Int("company-id", 0, "Target company ID (required)")
    flag.Parse()
    if *companyID == 0 {
        log.Fatal("--company-id is required")
    }
    // connect, begin tx, call migration.RunAll, migration.ResetSequences, commit
}
```

**Step 6: Verify CLI still compiles**

```bash
go build ./scripts/...
```
Expected: no errors

**Step 7: Commit**

```bash
git add internal/migration/ scripts/migrate_mssql.go scripts/migrate_phase*.go
git commit -m "feat: extract migration logic into internal/migration package"
```

---

## Task 4: River Args for Migration

**Files:**
- Modify: `internal/riverargs/args.go`

**Step 1: Add MigrateArgs**

```go
// MigrateArgs is the job args type for MSSQL migration jobs.
type MigrateArgs struct {
	RunID     int64  `json:"run_id"`
	CompanyID int    `json:"company_id"`
	BakPath   string `json:"bak_path"` // full path to .bak on shared volume
}

func (MigrateArgs) Kind() string { return "mssql_migrate" }

// No UniqueOpts — allow at most one by capping queue concurrency to 1.
```

**Step 2: Verify**

```bash
go build ./internal/riverargs/...
```

**Step 3: Commit**

```bash
git add internal/riverargs/args.go
git commit -m "feat: add MigrateArgs River job type"
```

---

## Task 5: Migration Worker

**Files:**
- Create: `internal/worker/migration.go`

**Step 1: Write the worker**

```go
// internal/worker/migration.go
package worker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/brady1408/atlinks/internal/migration"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/brady1408/atlinks/internal/riverargs"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type MigrateArgs = riverargs.MigrateArgs

type MigrationWorker struct {
	river.WorkerDefaults[MigrateArgs]
	Pool         *pgxpool.Pool
	RunStore     *store.MigrationRunStore
	MSSQLDSN     string // global config: the migration MSSQL container
}

func (w *MigrationWorker) Work(ctx context.Context, job *river.Job[MigrateArgs]) error {
	args := job.Args
	runID := args.RunID

	logLine := func(line string) {
		if err := w.RunStore.AppendLog(ctx, runID, line); err != nil {
			log.Printf("warn: append migration log: %v", err)
		}
	}

	if err := w.RunStore.SetRunning(ctx, runID); err != nil {
		return fmt.Errorf("set running: %w", err)
	}
	logLine(fmt.Sprintf("Migration started at %s", time.Now().Format(time.RFC3339)))

	// Step 1: Connect to the migration MSSQL container
	logLine("Connecting to MSSQL...")
	src, err := sql.Open("sqlserver", w.MSSQLDSN)
	if err != nil {
		w.RunStore.Fail(ctx, runID, err.Error())
		return fmt.Errorf("mssql open: %w", err)
	}
	defer src.Close()
	if err := src.PingContext(ctx); err != nil {
		w.RunStore.Fail(ctx, runID, err.Error())
		return fmt.Errorf("mssql ping: %w", err)
	}
	logLine("MSSQL connected.")

	// Step 2: Restore the .bak to a uniquely named database
	dbName := fmt.Sprintf("atlinks_migration_%d", runID)
	logLine(fmt.Sprintf("Restoring backup to [%s]...", dbName))
	if err := restoreBackup(ctx, src, dbName, args.BakPath, logLine); err != nil {
		w.RunStore.Fail(ctx, runID, err.Error())
		return fmt.Errorf("restore backup: %w", err)
	}
	logLine("Backup restored.")

	// Step 3: Switch src connection to the restored database
	restoredDSN := switchDatabase(w.MSSQLDSN, dbName)
	srcDB, err := sql.Open("sqlserver", restoredDSN)
	if err != nil {
		w.cleanup(ctx, src, dbName, runID)
		return fmt.Errorf("connect restored db: %w", err)
	}
	defer srcDB.Close()

	// Step 4: Run migration phases in a PG transaction
	logLine("Starting migration phases...")
	pgConn, err := w.Pool.Acquire(ctx)
	if err != nil {
		w.cleanup(ctx, src, dbName, runID)
		return fmt.Errorf("acquire pg conn: %w", err)
	}
	defer pgConn.Release()

	tx, err := pgConn.Begin(ctx)
	if err != nil {
		w.cleanup(ctx, src, dbName, runID)
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rawStats, err := migration.RunAll(ctx, srcDB, tx, args.CompanyID, logLine)
	if err != nil {
		w.RunStore.Fail(ctx, runID, err.Error())
		w.cleanup(ctx, src, dbName, runID)
		return fmt.Errorf("migration phases: %w", err)
	}

	// Step 5: Reset sequences
	logLine("Resetting sequences...")
	migration.ResetSequences(ctx, tx)

	if err := tx.Commit(ctx); err != nil {
		w.RunStore.Fail(ctx, runID, err.Error())
		w.cleanup(ctx, src, dbName, runID)
		return fmt.Errorf("commit: %w", err)
	}
	logLine("Migration committed.")

	// Step 6: Cleanup temp DB and .bak file
	w.cleanup(ctx, src, dbName, runID)

	// Step 7: Record stats and mark complete
	stats := make([]models.MigrationTableStat, len(rawStats))
	for i, s := range rawStats {
		stats[i] = models.MigrationTableStat{
			Table: s.Table, Source: s.Source,
			Inserted: s.Inserted, Skipped: s.Skipped,
		}
	}
	if err := w.RunStore.Complete(ctx, runID, stats); err != nil {
		log.Printf("warn: mark complete: %v", err)
	}
	logLine(fmt.Sprintf("Done at %s", time.Now().Format(time.RFC3339)))
	return nil
}

func (w *MigrationWorker) cleanup(ctx context.Context, src *sql.DB, dbName string, runID int64) {
	if _, err := src.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS [%s]", dbName)); err != nil {
		log.Printf("warn: drop db %s: %v", dbName, err)
	}
}

// restoreBackup runs RESTORE FILELISTONLY then RESTORE DATABASE on the MSSQL master connection.
func restoreBackup(ctx context.Context, src *sql.DB, dbName, bakPath string, logLine func(string)) error {
	// Get logical file names from the backup
	rows, err := src.QueryContext(ctx, fmt.Sprintf("RESTORE FILELISTONLY FROM DISK = N'%s'", bakPath))
	if err != nil {
		return fmt.Errorf("filelistonly: %w", err)
	}
	defer rows.Close()

	type fileInfo struct{ logical, fileType string }
	var files []fileInfo
	cols, _ := rows.Columns()
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals { ptrs[i] = &vals[i] }
		if err := rows.Scan(ptrs...); err != nil { continue }
		var logical, fileType string
		for i, col := range cols {
			switch col {
			case "LogicalName":
				if s, ok := vals[i].(string); ok { logical = s }
			case "Type":
				if s, ok := vals[i].(string); ok { fileType = s }
			}
		}
		files = append(files, fileInfo{logical, fileType})
	}

	// Build MOVE clauses
	var moves []string
	for _, f := range files {
		var dest string
		if strings.EqualFold(f.fileType, "D") {
			dest = fmt.Sprintf("/var/opt/mssql/data/%s.mdf", dbName)
		} else {
			dest = fmt.Sprintf("/var/opt/mssql/data/%s_log.ldf", dbName)
		}
		moves = append(moves, fmt.Sprintf("MOVE N'%s' TO N'%s'", f.logical, dest))
	}

	restoreSQL := fmt.Sprintf(
		"RESTORE DATABASE [%s] FROM DISK = N'%s' WITH %s, REPLACE, STATS = 5",
		dbName, bakPath, strings.Join(moves, ", "),
	)
	logLine("Executing RESTORE DATABASE...")
	if _, err := src.ExecContext(ctx, restoreSQL); err != nil {
		return fmt.Errorf("restore database: %w", err)
	}
	return nil
}

// switchDatabase replaces the database= param in a sqlserver DSN.
func switchDatabase(dsn, dbName string) string {
	// sqlserver://user:pass@host?database=X&...
	if idx := strings.Index(dsn, "database="); idx != -1 {
		end := strings.Index(dsn[idx:], "&")
		if end == -1 {
			return dsn[:idx] + "database=" + dbName
		}
		return dsn[:idx] + "database=" + dbName + dsn[idx+end:]
	}
	sep := "?"
	if strings.Contains(dsn, "?") { sep = "&" }
	return dsn + sep + "database=" + dbName
}
```

**Step 2: Verify**

```bash
go build ./internal/worker/...
```
Expected: no errors

**Step 3: Commit**

```bash
git add internal/worker/migration.go
git commit -m "feat: add MigrationWorker River job worker"
```

---

## Task 6: Config Updates

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add new fields and env var loading**

Add to `Config` struct:
```go
MSSQLMigrationDSN string
MigrationsDir     string
```

Add to `Load()`:
```go
MSSQLMigrationDSN: getEnv("MSSQL_MIGRATION_DSN", "sqlserver://sa:ATLinks2024!@localhost:1433?encrypt=disable"),
MigrationsDir:     getEnv("MIGRATIONS_DIR", "./data/migrations"),
```

**Step 2: Verify**

```bash
go build ./internal/config/...
```

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add MSSQL_MIGRATION_DSN and MIGRATIONS_DIR config vars"
```

---

## Task 7: Wire Migration into main.go

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: Add migration queue to initRiver**

In `initRiver`, add the `MigrationWorker` and a new `"migration"` queue entry:

```go
// In initRiver(), after existing workers:
migrationRunStore := store.NewMigrationRunStore(pool)
migrationWorker := &worker.MigrationWorker{
    Pool:     pool,
    RunStore: migrationRunStore,
    MSSQLDSN: cfg.MSSQLMigrationDSN,
}
river.AddWorker(workers, migrationWorker)

// In the river.NewClient call, add to Queues map:
Queues: map[string]river.QueueConfig{
    river.QueueDefault: {MaxWorkers: 5},
    "migration":        {MaxWorkers: 1},
},
```

**Step 2: Register migration handler**

In `initRoutes`, after the admin handler registration, add a `MigrationRunStore` and register the handler:

```go
migrationRunStore := store.NewMigrationRunStore(pool)
```

Return `migrationRunStore` via `riverStores` (or a new field). Then in `main()` after `initRiver`:

```go
handler.NewMigrationHandler(
    routeStores.migrationRunStore,
    deps.CompanyStore,
    riverClient,
    cfg.MigrationsDir,
).Register(routeStores.protectedMux, middleware.RequireRole("super_admin"))
```

**Step 3: Verify**

```bash
go build ./cmd/server/...
```

**Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire migration worker and handler into main"
```

---

## Task 8: Migration Handler

**Files:**
- Create: `internal/handler/migration_handler.go`

**Step 1: Write the handler**

```go
// internal/handler/migration_handler.go
package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/brady1408/atlinks/internal/handler/components/migration"
	"github.com/brady1408/atlinks/internal/riverargs"
	"github.com/brady1408/atlinks/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type MigrationHandler struct {
	runStore    *store.MigrationRunStore
	companyStore *store.CompanyStore
	river       *river.Client[pgx.Tx]
	migrationsDir string
	deps        *Deps
}

func NewMigrationHandler(
	runStore *store.MigrationRunStore,
	companyStore *store.CompanyStore,
	riverClient *river.Client[pgx.Tx],
	migrationsDir string,
	deps *Deps,
) *MigrationHandler {
	return &MigrationHandler{
		runStore:      runStore,
		companyStore:  companyStore,
		river:         riverClient,
		migrationsDir: migrationsDir,
		deps:          deps,
	}
}

func (h *MigrationHandler) Register(mux *http.ServeMux, mw func(http.Handler) http.Handler) {
	wrap := func(fn http.HandlerFunc) http.Handler { return mw(fn) }
	mux.HandleFunc("GET /admin/migration", h.index)
	mux.Handle("GET /admin/migration/new", wrap(h.newForm))
	mux.Handle("POST /admin/migration/start", wrap(h.start))
	mux.HandleFunc("GET /admin/migration/{id}", h.show)
	mux.HandleFunc("GET /admin/migration/{id}/log", h.logPoll)
	mux.Handle("POST /admin/migration/{id}/rerun", wrap(h.rerun))
}

func (h *MigrationHandler) index(w http.ResponseWriter, r *http.Request) {
	runs, err := h.runStore.List(r.Context())
	if err != nil {
		http.Error(w, "error loading runs", http.StatusInternalServerError)
		return
	}
	h.deps.renderTempl(w, r, migration.Index(h.deps.pageContext(w, r), runs))
}

func (h *MigrationHandler) newForm(w http.ResponseWriter, r *http.Request) {
	companies, err := h.companyStore.List(r.Context())
	if err != nil {
		http.Error(w, "error loading companies", http.StatusInternalServerError)
		return
	}
	h.deps.renderTempl(w, r, migration.NewForm(h.deps.pageContext(w, r), companies))
}

func (h *MigrationHandler) start(w http.ResponseWriter, r *http.Request) {
	// 2GB max — .bak files can be large
	if err := r.ParseMultipartForm(2 << 30); err != nil {
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}
	companyID, err := parseID(r.FormValue("company_id"))
	if err != nil {
		http.Error(w, "invalid company_id", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create migration run record first to get the run ID
	run, err := h.runStore.Create(r.Context(), int64(companyID), header.Filename)
	if err != nil {
		http.Error(w, "error creating run", http.StatusInternalServerError)
		return
	}

	// Save .bak to migrations dir as <run_id>.bak
	if err := os.MkdirAll(h.migrationsDir, 0755); err != nil {
		http.Error(w, "error creating migrations dir", http.StatusInternalServerError)
		return
	}
	bakPath := filepath.Join(h.migrationsDir, fmt.Sprintf("%d.bak", run.ID))
	f, err := os.Create(bakPath)
	if err != nil {
		http.Error(w, "error saving file", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	if _, err := io.Copy(f, file); err != nil {
		http.Error(w, "error writing file", http.StatusInternalServerError)
		return
	}

	// Enqueue River job on "migration" queue
	_, err = h.river.Insert(r.Context(), riverargs.MigrateArgs{
		RunID:     run.ID,
		CompanyID: companyID,
		BakPath:   bakPath,
	}, &river.InsertOpts{Queue: "migration"})
	if err != nil {
		http.Error(w, "error enqueueing job", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/migration/%d", run.ID), http.StatusSeeOther)
}

func (h *MigrationHandler) show(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	run, err := h.runStore.Get(r.Context(), int64(id))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.deps.renderTempl(w, r, migration.Show(h.deps.pageContext(w, r), run))
}

func (h *MigrationHandler) logPoll(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	logText, status, err := h.runStore.GetLog(r.Context(), int64(id))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.deps.renderTempl(w, r, migration.LogPanel(logText, status))
}

func (h *MigrationHandler) rerun(w http.ResponseWriter, r *http.Request) {
	// Re-run: delete all company data for this run's company, then redirect to new form
	// TODO: implement data wipe + re-enqueue
	http.Redirect(w, r, "/admin/migration/new", http.StatusSeeOther)
}
```

**Step 2: Verify**

```bash
go build ./internal/handler/...
```

**Step 3: Commit**

```bash
git add internal/handler/migration_handler.go
git commit -m "feat: add MigrationHandler with file upload and River enqueue"
```

---

## Task 9: Templ Components

**Files:**
- Create: `internal/handler/components/migration/migration.templ`

**Step 1: Write the components**

```templ
package migration

import (
	"fmt"
	"github.com/brady1408/atlinks/internal/handler/components"
	"github.com/brady1408/atlinks/internal/models"
)

templ Index(pc components.PageContext, runs []models.MigrationRun) {
	@components.Layout(pc) {
		<div class="page-header">
			<h1>MSSQL Migrations</h1>
			<a href="/admin/migration/new" class="btn btn-primary">New Migration</a>
		</div>
		<table class="table">
			<thead>
				<tr>
					<th>ID</th><th>Company</th><th>File</th><th>Status</th><th>Started</th><th>Finished</th>
				</tr>
			</thead>
			<tbody>
				for _, r := range runs {
					<tr>
						<td><a href={ templ.SafeURL(fmt.Sprintf("/admin/migration/%d", r.ID)) }>{ fmt.Sprint(r.ID) }</a></td>
						<td>{ r.CompanyName }</td>
						<td>{ r.BackupFilename }</td>
						<td><span class={ "badge badge-" + r.Status }>{ r.Status }</span></td>
						<td>{ components.FormatTimePtr(r.StartedAt) }</td>
						<td>{ components.FormatTimePtr(r.FinishedAt) }</td>
					</tr>
				}
			</tbody>
		</table>
	}
}

templ NewForm(pc components.PageContext, companies []models.Company) {
	@components.Layout(pc) {
		<div class="page-header"><h1>New Migration</h1></div>
		<form method="POST" action="/admin/migration/start" enctype="multipart/form-data" class="form-card">
			<input type="hidden" name="csrf_token" value={ pc.CSRFToken }/>
			<div class="form-group">
				<label>Target Company</label>
				<select name="company_id" class="form-control" required>
					for _, c := range companies {
						<option value={ fmt.Sprint(c.ID) }>{ c.CompanyName }</option>
					}
				</select>
			</div>
			<div class="form-group">
				<label>Backup File (.bak)</label>
				<input type="file" name="backup" accept=".bak" class="form-control" required/>
			</div>
			<button type="submit" class="btn btn-primary">Start Migration</button>
		</form>
	}
}

templ Show(pc components.PageContext, run *models.MigrationRun) {
	@components.Layout(pc) {
		<div class="page-header">
			<h1>Migration #{ fmt.Sprint(run.ID) }</h1>
			<span class={ "badge badge-" + run.Status }>{ run.Status }</span>
		</div>
		<dl class="detail-list">
			<dt>Company</dt><dd>{ run.CompanyName }</dd>
			<dt>File</dt><dd>{ run.BackupFilename }</dd>
		</dl>
		@LogPanel(run.Log, run.Status)
		if len(run.Stats) > 0 {
			@StatsTable(run.Stats)
		}
		if run.Status == "failed" {
			<form method="POST" action={ templ.SafeURL(fmt.Sprintf("/admin/migration/%d/rerun", run.ID)) }>
				<input type="hidden" name="csrf_token" value={ pc.CSRFToken }/>
				<button type="submit" class="btn btn-warning">Re-run Migration</button>
			</form>
		}
	}
}

templ LogPanel(log string, status string) {
	<div
		id="log-panel"
		if status == "pending" || status == "running" {
			hx-get={ "." }
			hx-trigger="every 2s"
			hx-target="#log-panel"
			hx-swap="outerHTML"
		}
	>
		<pre class="migration-log">{ log }</pre>
	</div>
}

templ StatsTable(stats []models.MigrationTableStat) {
	<h3>Results</h3>
	<table class="table">
		<thead><tr><th>Table</th><th>Source</th><th>Inserted</th><th>Skipped</th></tr></thead>
		<tbody>
			for _, s := range stats {
				<tr>
					<td>{ s.Table }</td>
					<td>{ fmt.Sprint(s.Source) }</td>
					<td>{ fmt.Sprint(s.Inserted) }</td>
					<td>{ fmt.Sprint(s.Skipped) }</td>
				</tr>
			}
		</tbody>
	</table>
}
```

**Step 2: Generate templ code**

```bash
templ generate
```
Expected: creates `migration_templ.go`

**Step 3: Verify**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git add internal/handler/components/migration/
git commit -m "feat: add migration templ components"
```

---

## Task 10: docker-compose.prod.yml Updates

**Files:**
- Modify: `docker-compose.prod.yml`

**Step 1: Add mssql service and migrations volume**

```yaml
services:
  atlinks:
    # ... existing config ...
    volumes:
      - /volume1/docker/atlinks/uploads:/data/uploads
      - migrations:/migrations   # shared with mssql
    environment:
      PORT: "8080"
      UPLOAD_DIR: "/data/uploads"
      MIGRATIONS_DIR: "/migrations"

  mssql:
    image: mcr.microsoft.com/mssql/server:2019-latest
    container_name: atlinks-mssql
    restart: unless-stopped
    environment:
      ACCEPT_EULA: "Y"
      SA_PASSWORD: ${MSSQL_SA_PASSWORD}
    volumes:
      - migrations:/migrations
    networks:
      - tunnel

  # ... cloudflared services unchanged ...

volumes:
  migrations:
```

**Step 2: Add to .env.prod (local copy, also update on NAS)**

```
MSSQL_SA_PASSWORD=<strong-password>
MSSQL_MIGRATION_DSN=sqlserver://sa:<strong-password>@atlinks-mssql:1433?encrypt=disable
```

**Step 3: Commit**

```bash
git add docker-compose.prod.yml
git commit -m "feat: add mssql migration container and shared volume to compose"
```

---

## Task 11: Nav Link

**Files:**
- Modify: `internal/handler/components/nav.templ`

**Step 1: Add Migration link in the super_admin admin section**

Find the admin section in `nav.templ` and add:
```templ
if pc.User.Role == "super_admin" {
    // ... existing links ...
    <a href="/admin/migration" class={ navLink(pc, "/admin/migration") }>Migration</a>
}
```

**Step 2: Regenerate + verify**

```bash
templ generate && go build ./...
```

**Step 3: Commit**

```bash
git add internal/handler/components/nav.templ internal/handler/components/nav_templ.go
git commit -m "feat: add Migration nav link for super_admin"
```

---

## Task 12: End-to-End Smoke Test

**Step 1: Start local environment**

```bash
docker compose up -d
make migrate-up
make run
```

**Step 2: Start local MSSQL for testing**

```bash
docker run -d --name atlinks-mssql \
  -e 'ACCEPT_EULA=Y' -e 'SA_PASSWORD=ATLinks2024!' \
  -p 1433:1433 \
  -v $(pwd)/data/migrations:/migrations \
  mcr.microsoft.com/mssql/server:2019-latest
```

**Step 3: Set env vars**

```bash
export MSSQL_MIGRATION_DSN="sqlserver://sa:ATLinks2024!@localhost:1433?encrypt=disable"
export MIGRATIONS_DIR="./data/migrations"
```

**Step 4: Test CLI tool**

```bash
go run ./scripts/ \
  --mssql "sqlserver://sa:ATLinks2024!@localhost:1433?database=Demo&encrypt=disable" \
  --pg "postgres://atlinks:atlinks_dev@localhost:5432/atlinks" \
  --company-id 1
```
Expected: phase-by-phase log output, no fatal errors.

**Step 5: Test UI**

1. Login as super_admin → navigate to `/admin/migration`
2. Click "New Migration" → select company → upload a `.bak` file → "Start Migration"
3. Should redirect to `/admin/migration/:id`
4. Log panel should show progress updating every 2 seconds
5. When complete: status badge turns green, stats table appears

**Step 6: Final commit**

```bash
git add .
git commit -m "feat: complete MSSQL migration feature (River + admin UI + CLI)"
```
