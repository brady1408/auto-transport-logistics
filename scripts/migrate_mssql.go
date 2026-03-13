package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/brady1408/auto-transport-logistics/internal/migration"
	"github.com/jackc/pgx/v5"
)

func main() {
	mssqlDSN := flag.String("mssql", "sqlserver://sa:ATLinks2024!@localhost:1433?database=Demo&encrypt=disable", "MSSQL connection string")
	pgDSN := flag.String("pg", "postgres://atlinks:atlinks_dev@localhost:5432/atlinks", "PostgreSQL connection string")
	companyID := flag.Int("company-id", 0, "Target company ID (required)")
	flag.Parse()

	if *companyID == 0 {
		log.Fatal("--company-id is required")
	}

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

	start := time.Now()

	// Run all migration phases
	logger := func(line string) { log.Println(" ", line) }
	stats, err := migration.RunAll(ctx, src, tx, *companyID, logger)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Reset sequences
	log.Println("\n=== Resetting Sequences ===")
	migration.ResetSequences(ctx, tx)

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
	for _, s := range stats {
		log.Printf("%-25s %8d %8d %8d", s.Table, s.Source, s.Inserted, s.Skipped)
		totalSrc += s.Source
		totalIns += s.Inserted
		totalSkip += s.Skipped
	}
	log.Printf("%s", strings.Repeat("-", 55))
	log.Printf("%-25s %8d %8d %8d", "TOTAL", totalSrc, totalIns, totalSkip)
}
