package audit

import (
	"context"
	"encoding/json"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Log(ctx context.Context, tableName string, recordID int, action string, oldValues, newValues any) {
	user, _ := auth.GetUser(ctx)

	var oldJSON, newJSON []byte
	if oldValues != nil {
		oldJSON, _ = json.Marshal(oldValues)
	}
	if newValues != nil {
		newJSON, _ = json.Marshal(newValues)
	}

	// Fire-and-forget audit logging — don't block the request on audit insert failure
	_, _ = s.pool.Exec(ctx,
		`INSERT INTO audit_log (table_name, record_id, action, old_values, new_values, user_id, username, company_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		tableName, recordID, action, oldJSON, newJSON, nilIfZero(user.ID), user.Username, nilIfZero(user.CompanyID),
	)
}

func nilIfZero(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}
