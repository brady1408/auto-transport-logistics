package store

import (
	"context"
	"fmt"

	"github.com/brady1408/atlinks/internal/auth"
	"github.com/brady1408/atlinks/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AttachmentStore struct {
	pool *pgxpool.Pool
}

func NewAttachmentStore(pool *pgxpool.Pool) *AttachmentStore {
	return &AttachmentStore{pool: pool}
}

func (s *AttachmentStore) Create(ctx context.Context, att *models.Attachment) error {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO attachments (company_id, category, entity_id, filename, storage_key, content_type, size_bytes, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`,
		att.CompanyID, att.Category, att.EntityID, att.Filename, att.StorageKey,
		att.ContentType, att.SizeBytes, att.UploadedBy,
	).Scan(&att.ID, &att.CreatedAt)
	if err != nil {
		return fmt.Errorf("create attachment: %w", err)
	}
	return nil
}

func (s *AttachmentStore) GetByID(ctx context.Context, id int) (*models.Attachment, error) {
	var att models.Attachment
	err := s.pool.QueryRow(ctx,
		`SELECT id, company_id, category, entity_id, filename, storage_key, content_type, size_bytes, uploaded_by, created_at
		FROM attachments WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&att.ID, &att.CompanyID, &att.Category, &att.EntityID, &att.Filename,
		&att.StorageKey, &att.ContentType, &att.SizeBytes, &att.UploadedBy, &att.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get attachment %d: %w", id, err)
	}
	return &att, nil
}

func (s *AttachmentStore) ListByEntity(ctx context.Context, category string, entityID int) ([]models.Attachment, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}

	qb := newQueryBuilder()
	qb.Add("category = ?", category)
	qb.Add("entity_id = ?", entityID)
	if companyID != 0 {
		qb.Add("company_id = ?", companyID)
	}

	qb.AddRaw("deleted_at IS NULL")
	query := fmt.Sprintf(
		`SELECT id, company_id, category, entity_id, filename, storage_key, content_type, size_bytes, uploaded_by, created_at
		FROM attachments %s ORDER BY created_at ASC`, qb.Where(),
	)

	rows, err := s.pool.Query(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Attachment, error) {
		var a models.Attachment
		err := row.Scan(&a.ID, &a.CompanyID, &a.Category, &a.EntityID, &a.Filename,
			&a.StorageKey, &a.ContentType, &a.SizeBytes, &a.UploadedBy, &a.CreatedAt)
		return a, err
	})
}

func (s *AttachmentStore) ListBackups(ctx context.Context) ([]models.Attachment, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, company_id, category, entity_id, filename, storage_key, content_type, size_bytes, uploaded_by, created_at
		FROM attachments WHERE category = 'backup' AND deleted_at IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Attachment, error) {
		var a models.Attachment
		err := row.Scan(&a.ID, &a.CompanyID, &a.Category, &a.EntityID, &a.Filename,
			&a.StorageKey, &a.ContentType, &a.SizeBytes, &a.UploadedBy, &a.CreatedAt)
		return a, err
	})
}

func (s *AttachmentStore) Delete(ctx context.Context, id int) error {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return err
	}
	qb := newQueryBuilder()
	qb.Add("id = ?", id)
	if companyID != 0 {
		qb.Add("company_id = ?", companyID)
	}
	qb.AddRaw("deleted_at IS NULL")
	result, err := s.pool.Exec(ctx, "UPDATE attachments SET deleted_at = NOW() "+qb.Where(), qb.Args()...)
	if err != nil {
		return fmt.Errorf("delete attachment %d: %w", id, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("attachment %d not found", id)
	}
	return nil
}

// DeleteByEntity deletes all attachments for a given entity and returns storage keys for disk cleanup.
func (s *AttachmentStore) DeleteByEntity(ctx context.Context, category string, entityID int) ([]string, error) {
	companyID, err := auth.GetCompanyID(ctx)
	if err != nil {
		return nil, err
	}
	qb := newQueryBuilder()
	qb.Add("category = ?", category)
	qb.Add("entity_id = ?", entityID)
	if companyID != 0 {
		qb.Add("company_id = ?", companyID)
	}

	qb.AddRaw("deleted_at IS NULL")
	query := "UPDATE attachments SET deleted_at = NOW() " + qb.Where()
	_, err = s.pool.Exec(ctx, query, qb.Args()...)
	if err != nil {
		return nil, fmt.Errorf("delete attachments: %w", err)
	}
	return []string{}, nil
}
