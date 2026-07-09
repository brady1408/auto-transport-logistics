-- +goose Up
-- Trailers are soft-deleted, so the unique trailer-number index must ignore
-- deleted rows or a deleted trailer's number can never be reused.
DROP INDEX idx_trailers_company_trailer_number;
CREATE UNIQUE INDEX idx_trailers_company_trailer_number ON trailers (company_id, trailer_number) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX idx_trailers_company_trailer_number;
CREATE UNIQUE INDEX idx_trailers_company_trailer_number ON trailers (company_id, trailer_number);
