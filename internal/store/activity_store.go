package store

import (
	"context"
	"fmt"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ActivityStore struct {
	pool *pgxpool.Pool
}

func NewActivityStore(pool *pgxpool.Pool) *ActivityStore {
	return &ActivityStore{pool: pool}
}

func (s *ActivityStore) Insert(ctx context.Context, l models.ActivityLog) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO activity_log (user_id, username, company_id, method, path, status_code, duration_ms, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		l.UserID, l.Username, l.CompanyID, l.Method, l.Path, l.StatusCode, l.DurationMS, l.IPAddress, l.UserAgent,
	)
	return err
}

func scanActivityLog(row interface{ Scan(dest ...any) error }) (models.ActivityLog, error) {
	var l models.ActivityLog
	err := row.Scan(
		&l.ID, &l.UserID, &l.Username, &l.CompanyID,
		&l.Method, &l.Path, &l.StatusCode, &l.DurationMS,
		&l.IPAddress, &l.UserAgent, &l.CreatedAt,
	)
	return l, err
}

const activityColumns = `id, user_id, username, company_id, method, path, status_code, duration_ms, ip_address, user_agent, created_at`

func (s *ActivityStore) List(ctx context.Context, f models.ActivityFilter) (*models.ActivityListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 50
	}

	qb := newQueryBuilder()
	if f.UserID != 0 {
		qb.Add("user_id = ?", f.UserID)
	}
	if f.CompanyID != 0 {
		qb.Add("company_id = ?", f.CompanyID)
	}
	if f.Method != "" {
		qb.Add("method = ?", f.Method)
	}
	if f.Path != "" {
		qb.Add("path ILIKE ?", "%"+f.Path+"%")
	}
	if f.DateFrom != "" {
		qb.Add("created_at >= ?", f.DateFrom)
	}
	if f.DateTo != "" {
		qb.Add("created_at <= ?", f.DateTo)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM activity_log "+qb.Where(), qb.Args()...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count activity: %w", err)
	}

	paginate := qb.Paginate(f.PageSize, f.Page)
	query := fmt.Sprintf("SELECT %s FROM activity_log %s ORDER BY id DESC %s", activityColumns, qb.Where(), paginate)
	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list activity: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.ActivityLog, error) {
		return scanActivityLog(row)
	})
	if err != nil {
		return nil, fmt.Errorf("scan activity: %w", err)
	}

	return &models.ActivityListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *ActivityStore) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.pool.Exec(ctx, "DELETE FROM activity_log WHERE created_at < $1", cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old activity: %w", err)
	}
	return result.RowsAffected(), nil
}

// GetStats returns aggregated statistics for the activity dashboard.
func (s *ActivityStore) GetStats(ctx context.Context, since time.Time) (*models.ActivityStats, error) {
	stats := &models.ActivityStats{}

	// Total requests + unique users
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*), COUNT(DISTINCT user_id) FROM activity_log WHERE created_at >= $1`, since,
	).Scan(&stats.TotalRequests, &stats.UniqueUsers)
	if err != nil {
		return nil, fmt.Errorf("activity totals: %w", err)
	}

	// Top paths (top 10 by count)
	rows, err := s.pool.Query(ctx,
		`SELECT path, COUNT(*) AS cnt FROM activity_log WHERE created_at >= $1
		GROUP BY path ORDER BY cnt DESC LIMIT 10`, since,
	)
	if err != nil {
		return nil, fmt.Errorf("top paths: %w", err)
	}
	stats.TopPaths, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.PathCount, error) {
		var pc models.PathCount
		return pc, row.Scan(&pc.Path, &pc.Count)
	})
	if err != nil {
		return nil, fmt.Errorf("scan top paths: %w", err)
	}

	// Active users (top 20 by request count)
	rows, err = s.pool.Query(ctx,
		`SELECT user_id, COALESCE(username, 'unknown') AS uname, COUNT(*) AS cnt
		FROM activity_log WHERE created_at >= $1 AND user_id IS NOT NULL
		GROUP BY user_id, username ORDER BY cnt DESC LIMIT 20`, since,
	)
	if err != nil {
		return nil, fmt.Errorf("active users: %w", err)
	}
	stats.ActiveUsers, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.UserActivity, error) {
		var ua models.UserActivity
		return ua, row.Scan(&ua.UserID, &ua.Username, &ua.Count)
	})
	if err != nil {
		return nil, fmt.Errorf("scan active users: %w", err)
	}

	// Recent logins — first authenticated request per user per day in period
	rows, err = s.pool.Query(ctx,
		`SELECT DISTINCT ON (user_id, created_at::date)
			id, user_id, username, company_id, method, path, status_code, duration_ms, ip_address, user_agent, created_at
		FROM activity_log
		WHERE created_at >= $1 AND user_id IS NOT NULL
		ORDER BY user_id, created_at::date DESC, created_at ASC
		LIMIT 20`, since,
	)
	if err != nil {
		return nil, fmt.Errorf("recent logins: %w", err)
	}
	stats.RecentLogins, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.ActivityLog, error) {
		return scanActivityLog(row)
	})
	if err != nil {
		return nil, fmt.Errorf("scan recent logins: %w", err)
	}

	// Hourly distribution (0–23) for the since period
	rows, err = s.pool.Query(ctx,
		`SELECT EXTRACT(HOUR FROM created_at)::int AS hr, COUNT(*) AS cnt
		FROM activity_log WHERE created_at >= $1
		GROUP BY hr ORDER BY hr`, since,
	)
	if err != nil {
		return nil, fmt.Errorf("hourly requests: %w", err)
	}
	stats.HourlyRequests, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.HourlyCount, error) {
		var hc models.HourlyCount
		return hc, row.Scan(&hc.Hour, &hc.Count)
	})
	if err != nil {
		return nil, fmt.Errorf("scan hourly: %w", err)
	}

	return stats, nil
}

// GetUserTimeline returns recent activity for a single user.
func (s *ActivityStore) GetUserTimeline(ctx context.Context, userID, limit int) ([]models.ActivityLog, error) {
	rows, err := s.pool.Query(ctx,
		fmt.Sprintf("SELECT %s FROM activity_log WHERE user_id = $1 ORDER BY id DESC LIMIT $2", activityColumns),
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("user timeline: %w", err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.ActivityLog, error) {
		return scanActivityLog(row)
	})
	if err != nil {
		return nil, fmt.Errorf("scan user timeline: %w", err)
	}
	return items, nil
}
