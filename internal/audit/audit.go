package audit

import (
	"context"
	"encoding/json"
	"log"

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
	if s == nil || s.pool == nil {
		return // no-op when used without a real database (e.g. tests)
	}
	user, _ := auth.GetUser(ctx)

	var oldJSON, newJSON []byte
	if oldValues != nil {
		var err error
		oldJSON, err = json.Marshal(oldValues)
		if err != nil {
			log.Printf("audit: marshal old values for %s/%d: %v", tableName, recordID, err)
		}
	}
	if newValues != nil {
		var err error
		newJSON, err = json.Marshal(newValues)
		if err != nil {
			log.Printf("audit: marshal new values for %s/%d: %v", tableName, recordID, err)
		}
	}

	// Fire-and-forget audit logging — don't block the request on audit insert failure
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO audit_log (table_name, record_id, action, old_values, new_values, user_id, username, company_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		tableName, recordID, action, oldJSON, newJSON, nilIfZero(user.ID), user.Username, nilIfZero(user.CompanyID),
	); err != nil {
		log.Printf("audit: insert for %s/%d/%s: %v", tableName, recordID, action, err)
	}
}

func nilIfZero(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}
