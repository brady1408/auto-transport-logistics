package store

import (
	"context"
	"fmt"
	"time"

	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QBOStore struct {
	pool *pgxpool.Pool
}

func NewQBOStore(pool *pgxpool.Pool) *QBOStore {
	return &QBOStore{pool: pool}
}

func (s *QBOStore) GetConnection(ctx context.Context, companyID int) (*models.QBOConnection, error) {
	var c models.QBOConnection
	err := s.pool.QueryRow(ctx, `
		SELECT id, company_id, realm_id, access_token, refresh_token,
		       token_expiry, connected_by, connected_at, created_at, updated_at
		FROM qbo_connections WHERE company_id = $1`, companyID,
	).Scan(&c.ID, &c.CompanyID, &c.RealmID, &c.AccessToken, &c.RefreshToken,
		&c.TokenExpiry, &c.ConnectedBy, &c.ConnectedAt, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get qbo connection: %w", err)
	}
	return &c, nil
}

func (s *QBOStore) UpsertConnection(ctx context.Context, c *models.QBOConnection) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO qbo_connections
		    (company_id, realm_id, access_token, refresh_token, token_expiry, connected_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (company_id) DO UPDATE SET
		    realm_id      = EXCLUDED.realm_id,
		    access_token  = EXCLUDED.access_token,
		    refresh_token = EXCLUDED.refresh_token,
		    token_expiry  = EXCLUDED.token_expiry,
		    connected_by  = EXCLUDED.connected_by,
		    connected_at  = NOW(),
		    updated_at    = NOW()`,
		c.CompanyID, c.RealmID, c.AccessToken, c.RefreshToken, c.TokenExpiry, c.ConnectedBy,
	)
	return err
}

func (s *QBOStore) UpdateTokens(ctx context.Context, companyID int, accessToken, refreshToken string, expiry time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE qbo_connections
		SET access_token = $2, refresh_token = $3, token_expiry = $4, updated_at = NOW()
		WHERE company_id = $1`,
		companyID, accessToken, refreshToken, expiry,
	)
	return err
}

func (s *QBOStore) DeleteConnection(ctx context.Context, companyID int) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM qbo_connections WHERE company_id = $1`, companyID)
	return err
}

func (s *QBOStore) Log(ctx context.Context, entry *models.QBOSyncLog) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO qbo_sync_log
		    (company_id, entity_type, entity_id, qbo_id, action, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.CompanyID, entry.EntityType, entry.EntityID, entry.QBOID,
		entry.Action, entry.Status, entry.ErrorMessage,
	)
	return err
}

func (s *QBOStore) RecentFailures(ctx context.Context, companyID int) ([]models.QBOSyncLog, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, company_id, entity_type, entity_id, qbo_id, action, status,
		       error_message, attempted_at, completed_at
		FROM qbo_sync_log
		WHERE company_id = $1 AND status = 'failed'
		ORDER BY attempted_at DESC LIMIT 20`,
		companyID,
	)
	if err != nil {
		return nil, fmt.Errorf("recent failures: %w", err)
	}
	defer rows.Close()

	var results []models.QBOSyncLog
	for rows.Next() {
		var e models.QBOSyncLog
		if err := rows.Scan(&e.ID, &e.CompanyID, &e.EntityType, &e.EntityID, &e.QBOID,
			&e.Action, &e.Status, &e.ErrorMessage, &e.AttemptedAt, &e.CompletedAt); err != nil {
			return nil, err
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

func (s *QBOStore) UpdateCustomerQBOID(ctx context.Context, customerID int, qboID string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE customers SET qbo_customer_id = $2 WHERE id = $1`,
		customerID, qboID,
	)
	return err
}

func (s *QBOStore) UpdateInvoiceQBO(ctx context.Context, invoiceID int, qboID, syncToken string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE invoices SET qbo_invoice_id = $2, qbo_sync_token = $3, qbo_synced_at = NOW() WHERE id = $1`,
		invoiceID, qboID, syncToken,
	)
	return err
}

func (s *QBOStore) UpdatePaymentQBO(ctx context.Context, paymentID int, qboID, syncToken string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE payments SET qbo_payment_id = $2, qbo_sync_token = $3, qbo_synced_at = NOW() WHERE id = $1`,
		paymentID, qboID, syncToken,
	)
	return err
}
