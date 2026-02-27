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
