-- +goose Up
CREATE TABLE subscriptions (
    id              SERIAL PRIMARY KEY,
    company_id      INT NOT NULL UNIQUE REFERENCES companies(id) ON DELETE CASCADE,
    tier            VARCHAR(20) NOT NULL DEFAULT 'basic',
    addon_edi       BOOLEAN NOT NULL DEFAULT false,
    edi_monthly_limit INT,
    external_id     VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Grandfather all existing companies at Enterprise + EDI
INSERT INTO subscriptions (company_id, tier, addon_edi)
SELECT id, 'enterprise', true FROM companies;

-- +goose Down
DROP TABLE IF EXISTS subscriptions;
