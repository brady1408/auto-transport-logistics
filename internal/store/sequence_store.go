package store

import (
	"context"
	"fmt"

	"github.com/brady1408/auto-transport-logistics/internal/auth"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SequenceStore struct {
	pool *pgxpool.Pool
}

func NewSequenceStore(pool *pgxpool.Pool) *SequenceStore {
	return &SequenceStore{pool: pool}
}

// NextVal atomically increments and returns the next value for the given sequence name.
// Uses the company_id from context. If no row exists, upserts starting at 1.
func (s *SequenceStore) NextVal(ctx context.Context, seqName string) (int64, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return 0, err
	}
	return s.nextVal(ctx, s.pool, companyID, seqName)
}

// NextValTx is like NextVal but runs on an existing transaction.
func (s *SequenceStore) NextValTx(ctx context.Context, tx pgx.Tx, seqName string) (int64, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return 0, err
	}
	return s.nextVal(ctx, tx, companyID, seqName)
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *SequenceStore) nextVal(ctx context.Context, q querier, companyID int, seqName string) (int64, error) {
	var val int64
	err := q.QueryRow(ctx,
		`INSERT INTO company_sequences (company_id, seq_name, current_val)
		 VALUES ($1, $2, 1)
		 ON CONFLICT (company_id, seq_name)
		 DO UPDATE SET current_val = company_sequences.current_val + 1
		 RETURNING current_val`,
		companyID, seqName,
	).Scan(&val)
	if err != nil {
		return 0, fmt.Errorf("next sequence val %q: %w", seqName, err)
	}
	return val, nil
}
