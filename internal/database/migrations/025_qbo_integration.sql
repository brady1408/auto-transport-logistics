-- +goose Up

-- qbo_connections: one row per company, stores OAuth tokens
CREATE TABLE qbo_connections (
    id              bigserial PRIMARY KEY,
    company_id      integer NOT NULL UNIQUE REFERENCES companies(id) ON DELETE CASCADE,
    realm_id        text    NOT NULL,
    access_token    text    NOT NULL,
    refresh_token   text    NOT NULL,
    token_expiry    timestamptz NOT NULL,
    connected_by    text    NOT NULL,
    connected_at    timestamptz NOT NULL DEFAULT NOW(),
    created_at      timestamptz NOT NULL DEFAULT NOW(),
    updated_at      timestamptz NOT NULL DEFAULT NOW()
);

-- qbo_sync_log: audit trail for every sync attempt
CREATE TABLE qbo_sync_log (
    id            bigserial PRIMARY KEY,
    company_id    integer NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    entity_type   text    NOT NULL CHECK (entity_type IN ('customer', 'invoice', 'payment')),
    entity_id     integer NOT NULL,
    qbo_id        text,
    action        text    NOT NULL CHECK (action IN ('create', 'update', 'void')),
    status        text    NOT NULL CHECK (status IN ('success', 'failed')),
    error_message text,
    attempted_at  timestamptz NOT NULL DEFAULT NOW(),
    completed_at  timestamptz
);

CREATE INDEX idx_qbo_sync_log_company_entity ON qbo_sync_log(company_id, entity_type, entity_id);

-- Add QBO columns to existing tables
ALTER TABLE customers ADD COLUMN qbo_customer_id text;

ALTER TABLE invoices ADD COLUMN qbo_invoice_id  text;
ALTER TABLE invoices ADD COLUMN qbo_sync_token  text;
ALTER TABLE invoices ADD COLUMN qbo_synced_at   timestamptz;

ALTER TABLE payments ADD COLUMN qbo_payment_id  text;
ALTER TABLE payments ADD COLUMN qbo_sync_token  text;
ALTER TABLE payments ADD COLUMN qbo_synced_at   timestamptz;

-- +goose Down

ALTER TABLE payments DROP COLUMN IF EXISTS qbo_payment_id;
ALTER TABLE payments DROP COLUMN IF EXISTS qbo_sync_token;
ALTER TABLE payments DROP COLUMN IF EXISTS qbo_synced_at;

ALTER TABLE invoices DROP COLUMN IF EXISTS qbo_invoice_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS qbo_sync_token;
ALTER TABLE invoices DROP COLUMN IF EXISTS qbo_synced_at;

ALTER TABLE customers DROP COLUMN IF EXISTS qbo_customer_id;

DROP TABLE IF EXISTS qbo_sync_log;
DROP TABLE IF EXISTS qbo_connections;
