package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedbackStore struct {
	pool *pgxpool.Pool
}

func NewFeedbackStore(pool *pgxpool.Pool) *FeedbackStore {
	return &FeedbackStore{pool: pool}
}

const feedbackColumns = `f.id, f.user_id, u.username, f.page_url, f.category,
	f.message, f.status, f.admin_notes, f.created_at, f.updated_at`

func scanFeedback(row interface{ Scan(dest ...any) error }) (*models.Feedback, error) {
	var fb models.Feedback
	err := row.Scan(
		&fb.ID, &fb.UserID, &fb.Username, &fb.PageURL, &fb.Category,
		&fb.Message, &fb.Status, &fb.AdminNotes, &fb.CreatedAt, &fb.UpdatedAt,
	)
	return &fb, err
}

func (s *FeedbackStore) List(ctx context.Context, f models.FeedbackFilter) (*models.FeedbackListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}

	var where []string
	var args []any
	argN := 1

	if f.Status != "" {
		where = append(where, fmt.Sprintf("f.status = $%d", argN))
		args = append(args, f.Status)
		argN++
	}
	if f.Category != "" {
		where = append(where, fmt.Sprintf("f.category = $%d", argN))
		args = append(args, f.Category)
		argN++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM feedback f " + whereClause
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count feedback: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf(
		"SELECT %s FROM feedback f JOIN users u ON u.id = f.user_id %s ORDER BY f.id DESC LIMIT $%d OFFSET $%d",
		feedbackColumns, whereClause, argN, argN+1,
	)
	args = append(args, f.PageSize, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list feedback: %w", err)
	}
	defer rows.Close()

	var items []models.Feedback
	for rows.Next() {
		fb, err := scanFeedback(rows)
		if err != nil {
			return nil, fmt.Errorf("scan feedback: %w", err)
		}
		items = append(items, *fb)
	}

	return &models.FeedbackListResult{
		Items:      items,
		TotalCount: total,
		Page:       f.Page,
		PageSize:   f.PageSize,
	}, nil
}

func (s *FeedbackStore) GetByID(ctx context.Context, id int) (*models.Feedback, error) {
	query := fmt.Sprintf("SELECT %s FROM feedback f JOIN users u ON u.id = f.user_id WHERE f.id = $1", feedbackColumns)
	fb, err := scanFeedback(s.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("get feedback %d: %w", id, err)
	}
	return fb, nil
}

func (s *FeedbackStore) Create(ctx context.Context, fb *models.Feedback) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO feedback (user_id, page_url, category, message)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status, created_at, updated_at`,
		fb.UserID, fb.PageURL, fb.Category, fb.Message,
	).Scan(&fb.ID, &fb.Status, &fb.CreatedAt, &fb.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create feedback: %w", err)
	}
	return nil
}

func (s *FeedbackStore) Update(ctx context.Context, fb *models.Feedback) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE feedback SET status = $1, admin_notes = $2, updated_at = NOW() WHERE id = $3`,
		fb.Status, fb.AdminNotes, fb.ID,
	)
	if err != nil {
		return fmt.Errorf("update feedback %d: %w", fb.ID, err)
	}
	return nil
}

func (s *FeedbackStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM feedback WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete feedback %d: %w", id, err)
	}
	return nil
}

func (s *FeedbackStore) CountOpen(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM feedback WHERE status = 'open'").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count open feedback: %w", err)
	}
	return count, nil
}
