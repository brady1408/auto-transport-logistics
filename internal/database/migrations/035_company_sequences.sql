-- +goose Up
CREATE TABLE company_sequences (
    company_id  INT NOT NULL REFERENCES companies(id),
    seq_name    VARCHAR(30) NOT NULL,
    current_val BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (company_id, seq_name)
);

-- Seed from existing data for every company
INSERT INTO company_sequences (company_id, seq_name, current_val)
SELECT id, 'order_number',   COALESCE((SELECT MAX(order_number::int)   FROM orders       WHERE order_number ~ '^\d+$'   AND company_id = c.id), 0) FROM companies c
UNION ALL
SELECT id, 'load_number',    COALESCE((SELECT MAX(load_number::int)    FROM trips        WHERE load_number ~ '^\d+$'    AND company_id = c.id), 0) FROM companies c
UNION ALL
SELECT id, 'invoice_number', COALESCE((SELECT MAX(invoice_number::int) FROM invoices     WHERE invoice_number ~ '^\d+$' AND company_id = c.id), 0) FROM companies c
UNION ALL
SELECT id, 'credit_number',  COALESCE((SELECT MAX(SUBSTRING(credit_number FROM '\d+')::int) FROM credit_memos WHERE credit_number ~ '\d+' AND company_id = c.id), 0) FROM companies c
UNION ALL
SELECT id, 'claim_number',   COALESCE((SELECT MAX(SUBSTRING(claim_number FROM '\d+')::int)  FROM damage_claims WHERE claim_number ~ '\d+' AND company_id = c.id), 0) FROM companies c;

-- +goose Down
DROP TABLE company_sequences;
