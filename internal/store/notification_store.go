package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/brady1408/auto-transport-logistics/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationStore struct {
	pool *pgxpool.Pool
}

func NewNotificationStore(pool *pgxpool.Pool) *NotificationStore {
	return &NotificationStore{pool: pool}
}

func (s *NotificationStore) getLastChecked(ctx context.Context, userID int) (time.Time, error) {
	var lastChecked time.Time
	err := s.pool.QueryRow(ctx,
		"SELECT COALESCE(notifications_last_checked_at, '1970-01-01'::timestamptz) FROM users WHERE id = $1",
		userID,
	).Scan(&lastChecked)
	if err != nil {
		return time.Time{}, fmt.Errorf("get last checked: %w", err)
	}
	return lastChecked, nil
}

// CountUnchecked returns the total number of unread notification items.
// Loadboard: claims with unread messages (per poster/carrier last_read_at).
// Feedback: comments from others on the user's feedback.
func (s *NotificationStore) CountUnchecked(ctx context.Context, userID, companyID int) (int, error) {
	// Count claims with unread loadboard messages sent by others
	var lbCount int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT c.id)
		FROM loadboard_messages m
		JOIN loadboard_claims c ON c.id = m.claim_id
		JOIN loadboard_listings l ON l.id = c.listing_id
		WHERE m.sender_company_id != $1
		  AND (
		    (l.poster_company_id = $1 AND (c.poster_last_read_at IS NULL OR m.created_at > c.poster_last_read_at))
		    OR
		    (c.carrier_company_id = $1 AND (c.carrier_last_read_at IS NULL OR m.created_at > c.carrier_last_read_at))
		  )`,
		companyID,
	).Scan(&lbCount)
	if err != nil {
		return 0, fmt.Errorf("count loadboard notifications: %w", err)
	}

	// Count feedback comments from others on my feedback
	var fbCount int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM feedback_comments fc
		JOIN feedback f ON f.id = fc.feedback_id
		WHERE f.user_id = $1
		  AND fc.user_id != $1
		  AND fc.internal = false`,
		userID,
	).Scan(&fbCount)
	if err != nil {
		return 0, fmt.Errorf("count feedback notifications: %w", err)
	}

	return lbCount + fbCount, nil
}

// ListRecent returns recent notification items (both read and unread).
// Items are sorted by created_at DESC, limited to the specified count.
func (s *NotificationStore) ListRecent(ctx context.Context, userID, companyID, limit int) ([]models.NotificationItem, error) {
	lastChecked, err := s.getLastChecked(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Loadboard: one item per claim with the latest unread message
	lbRows, err := s.pool.Query(ctx, `
		SELECT
			c.id AS entity_id,
			l.listing_number,
			latest.body,
			latest.sender_name,
			CASE
				WHEN l.poster_company_id = $1 THEN '/loadboard/my-listings/' || l.id::text
				ELSE '/loadboard/my-claims/' || c.id::text
			END AS url,
			latest.created_at
		FROM loadboard_claims c
		JOIN loadboard_listings l ON l.id = c.listing_id
		JOIN LATERAL (
			SELECT m.body, m.sender_name, m.created_at
			FROM loadboard_messages m
			WHERE m.claim_id = c.id
			  AND m.sender_company_id != $1
			  AND (
			    (l.poster_company_id = $1 AND (c.poster_last_read_at IS NULL OR m.created_at > c.poster_last_read_at))
			    OR
			    (c.carrier_company_id = $1 AND (c.carrier_last_read_at IS NULL OR m.created_at > c.carrier_last_read_at))
			  )
			ORDER BY m.created_at DESC
			LIMIT 1
		) latest ON true
		WHERE (l.poster_company_id = $1 OR c.carrier_company_id = $1)
		ORDER BY latest.created_at DESC
		LIMIT $2`,
		companyID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list loadboard notifications: %w", err)
	}
	lbItems, err := pgx.CollectRows(lbRows, func(row pgx.CollectableRow) (models.NotificationItem, error) {
		var n models.NotificationItem
		var listingNumber, body string
		if err := row.Scan(&n.EntityID, &listingNumber, &body, &n.Author, &n.URL, &n.CreatedAt); err != nil {
			return models.NotificationItem{}, err
		}
		n.Type = "loadboard_message"
		n.Title = "Message on " + listingNumber
		n.Description = truncate(body, 100)
		n.IsNew = n.CreatedAt.After(lastChecked)
		return n, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan loadboard notifications: %w", err)
	}

	// Feedback comments from others on my feedback
	fbRows, err := s.pool.Query(ctx, `
		SELECT
			f.id AS entity_id,
			f.category,
			fc.message,
			u.username,
			fc.created_at
		FROM feedback_comments fc
		JOIN feedback f ON f.id = fc.feedback_id
		JOIN users u ON u.id = fc.user_id
		WHERE f.user_id = $1
		  AND fc.user_id != $1
		  AND fc.internal = false
		ORDER BY fc.created_at DESC
		LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list feedback notifications: %w", err)
	}
	fbItems, err := pgx.CollectRows(fbRows, func(row pgx.CollectableRow) (models.NotificationItem, error) {
		var n models.NotificationItem
		var category, message string
		if err := row.Scan(&n.EntityID, &category, &message, &n.Author, &n.CreatedAt); err != nil {
			return models.NotificationItem{}, err
		}
		n.Type = "feedback_comment"
		n.Title = "Comment on " + feedbackCategoryLabel(category)
		n.Description = truncate(message, 100)
		n.URL = fmt.Sprintf("/feedback/%d", n.EntityID)
		n.IsNew = n.CreatedAt.After(lastChecked)
		return n, nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan feedback notifications: %w", err)
	}

	// Merge and sort by created_at DESC
	items := append(lbItems, fbItems...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

// MarkChecked updates the user's notifications_last_checked_at to NOW().
func (s *NotificationStore) MarkChecked(ctx context.Context, userID int) error {
	_, err := s.pool.Exec(ctx,
		"UPDATE users SET notifications_last_checked_at = NOW() WHERE id = $1",
		userID,
	)
	if err != nil {
		return fmt.Errorf("mark notifications checked: %w", err)
	}
	return nil
}

func feedbackCategoryLabel(category string) string {
	switch category {
	case "bug":
		return "Bug Report"
	case "feature":
		return "Feature Request"
	case "question":
		return "Question"
	default:
		return "Feedback"
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
