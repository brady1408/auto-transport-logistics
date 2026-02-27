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

// MigrateArgs is re-exported from riverargs for convenience.
type MigrateArgs = riverargs.MigrateArgs

type MigrationWorker struct {
	river.WorkerDefaults[MigrateArgs]
	Pool     *pgxpool.Pool
	RunStore *store.MigrationRunStore
	MSSQLDSN string // DSN pointing at the migration MSSQL container (master db)
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

	// Step 1: Connect to the migration MSSQL container (master)
	logLine("Connecting to MSSQL...")
	src, err := sql.Open("sqlserver", w.MSSQLDSN)
	if err != nil {
		_ = w.RunStore.Fail(ctx, runID, err.Error())
		return fmt.Errorf("mssql open: %w", err)
	}
	defer src.Close()
	if err := src.PingContext(ctx); err != nil {
		_ = w.RunStore.Fail(ctx, runID, err.Error())
		return fmt.Errorf("mssql ping: %w", err)
	}
	logLine("MSSQL connected.")

	// Step 2: Restore the .bak to a uniquely named database
	dbName := fmt.Sprintf("atlinks_migration_%d", runID)
	logLine(fmt.Sprintf("Restoring backup to [%s]...", dbName))
	if err := restoreBackup(ctx, src, dbName, args.BakPath, logLine); err != nil {
		_ = w.RunStore.Fail(ctx, runID, err.Error())
		return fmt.Errorf("restore backup: %w", err)
	}
	logLine("Backup restored.")

	// Step 3: Open a new connection targeting the restored database
	restoredDSN := switchDatabase(w.MSSQLDSN, dbName)
	srcDB, err := sql.Open("sqlserver", restoredDSN)
	if err != nil {
		w.cleanup(ctx, src, dbName)
		return fmt.Errorf("connect restored db: %w", err)
	}
	defer srcDB.Close()

	// Step 4: Run migration phases inside a PostgreSQL transaction
	logLine("Starting migration phases...")
	pgConn, err := w.Pool.Acquire(ctx)
	if err != nil {
		w.cleanup(ctx, src, dbName)
		return fmt.Errorf("acquire pg conn: %w", err)
	}
	defer pgConn.Release()

	tx, err := pgConn.Begin(ctx)
	if err != nil {
		w.cleanup(ctx, src, dbName)
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rawStats, err := migration.RunAll(ctx, srcDB, tx, args.CompanyID, logLine)
	if err != nil {
		_ = w.RunStore.Fail(ctx, runID, err.Error())
		w.cleanup(ctx, src, dbName)
		return fmt.Errorf("migration phases: %w", err)
	}

	// Step 5: Reset PostgreSQL sequences
	logLine("Resetting sequences...")
	migration.ResetSequences(ctx, tx)

	if err := tx.Commit(ctx); err != nil {
		_ = w.RunStore.Fail(ctx, runID, err.Error())
		w.cleanup(ctx, src, dbName)
		return fmt.Errorf("commit: %w", err)
	}
	logLine("Migration committed.")

	// Step 6: Drop temp MSSQL database
	w.cleanup(ctx, src, dbName)

	// Step 7: Record stats and mark complete
	stats := make([]models.MigrationTableStat, len(rawStats))
	for i, s := range rawStats {
		stats[i] = models.MigrationTableStat{
			Table:    s.Table,
			Source:   s.Source,
			Inserted: s.Inserted,
			Skipped:  s.Skipped,
		}
	}
	if err := w.RunStore.Complete(ctx, runID, stats); err != nil {
		log.Printf("warn: mark complete: %v", err)
	}
	logLine(fmt.Sprintf("Done at %s", time.Now().Format(time.RFC3339)))
	return nil
}

func (w *MigrationWorker) cleanup(ctx context.Context, src *sql.DB, dbName string) {
	if _, err := src.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS [%s]", dbName)); err != nil {
		log.Printf("warn: drop temp db %s: %v", dbName, err)
	}
}

// restoreBackup runs RESTORE FILELISTONLY then RESTORE DATABASE on the master MSSQL connection.
func restoreBackup(ctx context.Context, src *sql.DB, dbName, bakPath string, logLine func(string)) error {
	rows, err := src.QueryContext(ctx, fmt.Sprintf("RESTORE FILELISTONLY FROM DISK = N'%s'", bakPath))
	if err != nil {
		return fmt.Errorf("filelistonly: %w", err)
	}
	defer rows.Close()

	type fileInfo struct{ logical, fileType string }
	var files []fileInfo
	cols, _ := rows.Columns()
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		var logical, fileType string
		for i, col := range cols {
			switch col {
			case "LogicalName":
				if s, ok := vals[i].(string); ok {
					logical = s
				}
			case "Type":
				if s, ok := vals[i].(string); ok {
					fileType = s
				}
			}
		}
		files = append(files, fileInfo{logical, fileType})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("filelistonly rows: %w", err)
	}

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

// switchDatabase replaces or appends the database= parameter in a sqlserver DSN.
func switchDatabase(dsn, dbName string) string {
	if idx := strings.Index(dsn, "database="); idx != -1 {
		end := strings.Index(dsn[idx:], "&")
		if end == -1 {
			return dsn[:idx] + "database=" + dbName
		}
		return dsn[:idx] + "database=" + dbName + dsn[idx+end:]
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "database=" + dbName
}
