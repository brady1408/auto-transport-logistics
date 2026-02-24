package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FeedbackStore struct {
	pool *pgxpool.Pool
}

func NewFeedbackStore(pool *pgxpool.Pool) *FeedbackStore {
	return &FeedbackStore{pool: pool}
}

const feedbackColumns = `f.id, f.company_id, f.user_id, u.username, f.page_url, f.category,
	f.message, f.status, COALESCE(c.company_name, ''), COALESCE(cc.cnt, 0), f.created_at, f.updated_at`

func scanFeedback(row interface{ Scan(dest ...any) error }) (*models.Feedback, error) {
	var fb models.Feedback
	err := row.Scan(
		&fb.ID, &fb.CompanyID, &fb.UserID, &fb.Username, &fb.PageURL, &fb.Category,
		&fb.Message, &fb.Status, &fb.CompanyName, &fb.CommentCount, &fb.CreatedAt, &fb.UpdatedAt,
	)
	return &fb, err
}

const feedbackJoins = `FROM feedback f
	JOIN users u ON u.id = f.user_id
	LEFT JOIN companies c ON c.id = f.company_id
	LEFT JOIN (SELECT feedback_id, COUNT(*) AS cnt FROM feedback_comments GROUP BY feedback_id) cc ON cc.feedback_id = f.id`

func (s *FeedbackStore) List(ctx context.Context, f models.FeedbackFilter) (*models.FeedbackListResult, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 25
	}

	companyID := auth.GetCompanyID(ctx)

	var where []string
	var args []any
	argN := 1

	// super_admin (companyID=0) sees all; others see only their company
	if companyID != 0 {
		where = append(where, fmt.Sprintf("f.company_id = $%d", argN))
		args = append(args, companyID)
		argN++
	}

	if f.Status == "active" {
		where = append(where, fmt.Sprintf("f.status IN ($%d, $%d)", argN, argN+1))
		args = append(args, "open", "reviewed")
		argN += 2
	} else if f.Status != "" && f.Status != "all" {
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
	countQuery := "SELECT COUNT(*) " + feedbackJoins + " " + whereClause
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count feedback: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf(
		"SELECT %s %s %s ORDER BY f.id DESC LIMIT $%d OFFSET $%d",
		feedbackColumns, feedbackJoins, whereClause, argN, argN+1,
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
	companyID := auth.GetCompanyID(ctx)
	where := "WHERE f.id = $1"
	args := []any{id}
	if companyID != 0 {
		where += " AND f.company_id = $2"
		args = append(args, companyID)
	}
	query := fmt.Sprintf("SELECT %s %s %s", feedbackColumns, feedbackJoins, where)
	fb, err := scanFeedback(s.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("get feedback %d: %w", id, err)
	}
	return fb, nil
}

func (s *FeedbackStore) Create(ctx context.Context, fb *models.Feedback) error {
	fb.CompanyID = auth.GetCompanyID(ctx)
	err := s.pool.QueryRow(ctx,
		`INSERT INTO feedback (company_id, user_id, page_url, category, message)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at, updated_at`,
		fb.CompanyID, fb.UserID, fb.PageURL, fb.Category, fb.Message,
	).Scan(&fb.ID, &fb.Status, &fb.CreatedAt, &fb.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create feedback: %w", err)
	}
	return nil
}

func (s *FeedbackStore) Update(ctx context.Context, fb *models.Feedback) error {
	companyID := auth.GetCompanyID(ctx)
	where := "WHERE id = $2"
	args := []any{fb.Status, fb.ID}
	if companyID != 0 {
		where += " AND company_id = $3"
		args = append(args, companyID)
	}
	_, err := s.pool.Exec(ctx,
		"UPDATE feedback SET status = $1, updated_at = NOW() "+where,
		args...,
	)
	if err != nil {
		return fmt.Errorf("update feedback %d: %w", fb.ID, err)
	}
	return nil
}

func (s *FeedbackStore) Delete(ctx context.Context, id int) error {
	companyID := auth.GetCompanyID(ctx)
	where := "WHERE id = $1"
	args := []any{id}
	if companyID != 0 {
		where += " AND company_id = $2"
		args = append(args, companyID)
	}
	_, err := s.pool.Exec(ctx, "DELETE FROM feedback "+where, args...)
	if err != nil {
		return fmt.Errorf("delete feedback %d: %w", id, err)
	}
	return nil
}

func (s *FeedbackStore) CountOpen(ctx context.Context) (int, error) {
	companyID := auth.GetCompanyID(ctx)
	var count int
	if companyID == 0 {
		err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM feedback WHERE status = 'open'").Scan(&count)
		return count, err
	}
	err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM feedback WHERE status = 'open' AND company_id = $1", companyID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count open feedback: %w", err)
	}
	return count, nil
}

// ListComments returns comments for a feedback item.
// If includeInternal is false, internal comments are excluded.
func (s *FeedbackStore) ListComments(ctx context.Context, feedbackID int, includeInternal bool) ([]models.FeedbackComment, error) {
	where := "WHERE fc.feedback_id = $1"
	if !includeInternal {
		where += " AND fc.internal = false"
	}
	query := fmt.Sprintf(`SELECT fc.id, fc.feedback_id, fc.user_id, u.username, u.role, fc.company_id,
		fc.message, fc.internal, fc.created_at
		FROM feedback_comments fc
		JOIN users u ON u.id = fc.user_id
		%s ORDER BY fc.created_at ASC`, where)

	rows, err := s.pool.Query(ctx, query, feedbackID)
	if err != nil {
		return nil, fmt.Errorf("list feedback comments: %w", err)
	}
	defer rows.Close()

	var comments []models.FeedbackComment
	for rows.Next() {
		var c models.FeedbackComment
		if err := rows.Scan(&c.ID, &c.FeedbackID, &c.UserID, &c.Username, &c.UserRole,
			&c.CompanyID, &c.Message, &c.Internal, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan feedback comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, nil
}

// CreateComment inserts a new feedback comment.
func (s *FeedbackStore) CreateComment(ctx context.Context, c *models.FeedbackComment) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO feedback_comments (feedback_id, user_id, company_id, message, internal)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`,
		c.FeedbackID, c.UserID, c.CompanyID, c.Message, c.Internal,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("create feedback comment: %w", err)
	}
	return nil
}
