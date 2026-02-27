# MSSQL Migration Feature — Design

**Date:** 2026-02-27
**Status:** Approved

## Problem

Customers on the legacy Clarion/MSSQL ATLinks system need a way to import their historical data into the new Go/PostgreSQL app. Today this requires SSH access and manual CLI execution. The goal is a self-service in-app workflow: upload a `.bak` file, select a target company, watch it run.

## Approach

River background job on a dedicated `migration` queue, with an admin UI for upload + live progress monitoring. A dedicated MSSQL Docker container handles all restores. The CLI tool is updated as a parallel power-user path.

## Architecture

```
Admin uploads .bak + selects company
  → POST /admin/migration/start
  → .bak saved to shared volume (/migrations/<run_id>.bak)
  → migration_runs row inserted (status: pending)
  → MigrateArgs enqueued on "migration" River queue
  → redirect to /admin/migration/:id

River worker (migration queue, concurrency: 1):
  1. RESTORE FILELISTONLY → get logical file names
  2. mkdir /var/opt/mssql/data/migrations/<run_id>/
  3. RESTORE DATABASE [atlinks_migration_<run_id>]
       FROM DISK = '/migrations/<run_id>.bak'
       WITH MOVE ... (predictable container paths)
  4. Run migration phases 1-4 against atlinks_migration_<run_id>
       → all records inserted with target company_id
  5. DROP DATABASE [atlinks_migration_<run_id>]
  6. Delete /migrations/<run_id>.bak
  7. Reset PG sequences
  8. Mark run complete, write stats JSON

Admin UI polls /admin/migration/:id/log every 2s while running
  → stops polling when status = complete | failed
```

## Infrastructure

**New River queue:** `migration` (separate from `default` used by QBO workers)
**Concurrency:** 1 — never run two migrations simultaneously

**Shared volume:** mounted at `/migrations/` in both `atlinks` and `mssql` containers
**MSSQL data path:** `/var/opt/mssql/data/migrations/<run_id>/` (created per-run, cleaned up after)

**New env vars:**
- `MSSQL_DSN` — connection string to dedicated migration MSSQL container
- `MIGRATIONS_DIR` — shared volume mount path (default: `/migrations`)

**`docker-compose.prod.yml` changes:**
- Add `mssql` service
- Add `migrations` named volume shared between `atlinks` and `mssql`

## Data Model

```sql
-- Migration: 027_migration_runs.sql
CREATE TABLE migration_runs (
    id              BIGSERIAL PRIMARY KEY,
    company_id      BIGINT NOT NULL REFERENCES companies(id),
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending | running | complete | failed
    backup_filename TEXT NOT NULL,   -- original filename (display only)
    log             TEXT NOT NULL DEFAULT '',
    stats           JSONB,           -- {table: {source, inserted, skipped}}
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

No changes to existing tables. All migrated records already have `company_id` columns.

## Migration Logic

**Target company is always fresh (no existing data).** Plain INSERTs — no upsert complexity.

**Phase order:**
1. Lookup tables: zones, regions, dispatch codes, equipment types, items, vehicle makes, VIN definitions, color codes, hold codes, declination codes, field codes (5 tables), damage areas/types/severities, chart of accounts, terms, tax codes, vendors, vendor groups, carriers, zone pricing
2. Core entities: customers, employees, trucks
3. Dispatch: orders, order vehicles, trips, load details, order charges, vehicle damage, damage details, vehicle notes, trip fuel, trip expenses, trip routes, split loads
4. Accounting: invoices, invoice details, credit memos, payments, payment details, damage claims, accounts payable
5. Sequence reset: `setval('<table>_id_seq', MAX(id) + 1)` for all tables

**Per-table log line** (appended to `migration_runs.log` after each table):
```
[Phase 1] customers       src=1842  ins=1842  skip=0   341ms
```

**Re-run after failure:** A "Re-run" button on the detail page deletes all data for the `company_id` (safe — company was empty when migration started), then enqueues a new job.

## New Files

| File | Purpose |
|------|---------|
| `internal/database/migrations/027_migration_runs.sql` | New table |
| `internal/models/migration_run.go` | MigrationRun struct |
| `internal/store/migration_run_store.go` | CRUD + log append |
| `internal/worker/migration.go` | River worker (restore + migrate + cleanup) |
| `internal/riverargs/migration_args.go` | MigrateArgs struct |
| `internal/handler/migration_handler.go` | Admin UI handler |
| `internal/handler/components/migration/` | Templ components (list, show, log partial) |

## Modified Files

| File | Change |
|------|--------|
| `internal/config/config.go` | Add MSSQL_DSN, MIGRATIONS_DIR |
| `cmd/server/main.go` | Register migration queue + worker, register handler routes |
| `docker-compose.prod.yml` | Add mssql service + migrations volume |
| `scripts/migrate_mssql.go` | Add --company-id flag, switch from truncate to company-scoped inserts |
| `internal/handler/components/nav.templ` | Add Migration link under super_admin admin section |

## UI

**`GET /admin/migration`** — Migration index
- Table of all past runs: company, filename, status, started, finished
- "New Migration" button → `/admin/migration/new`

**`GET /admin/migration/new`** — Start form
- Company dropdown (all companies)
- `.bak` file upload
- "Start Migration" submit

**`GET /admin/migration/:id`** — Run detail
- Header: company, status badge, timestamps
- Live log: monospace panel, HTMX polls `GET /admin/migration/:id/log` every 2s while running
- Stats table (shown on complete): per-table source / inserted / skipped
- "Re-run" button (shown on failed)

## CLI Tool Update

`scripts/migrate_mssql.go` gains:
- `--company-id` flag (required)
- Removes `truncateAll()` call
- All inserts include `company_id = <flag value>`
- Sequence reset scoped only to affected tables

Useful for local testing and power users with direct DB access.
