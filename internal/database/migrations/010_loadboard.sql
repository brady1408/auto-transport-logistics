-- +goose Up

-- Global sequence for listing numbers
CREATE SEQUENCE loadboard_listing_number_seq START 1000;

-- Loadboard listings: transport jobs visible to all companies
CREATE TABLE loadboard_listings (
    id              SERIAL PRIMARY KEY,
    poster_company_id INT NOT NULL REFERENCES companies(id),
    poster_user_id  INT NOT NULL REFERENCES users(id),
    source_order_id INT NOT NULL REFERENCES orders(id),
    listing_number  VARCHAR(20) NOT NULL UNIQUE,
    title           VARCHAR(200) NOT NULL,

    -- Origin (denormalized from order load customer)
    origin_name     VARCHAR(100),
    origin_city     VARCHAR(100),
    origin_state    VARCHAR(10),
    origin_zip      VARCHAR(20),

    -- Destination (denormalized from order drop customer)
    dest_name       VARCHAR(100),
    dest_city       VARCHAR(100),
    dest_state      VARCHAR(10),
    dest_zip        VARCHAR(20),

    -- Pricing
    carrier_pay     NUMERIC(9,2) NOT NULL DEFAULT 0,

    -- Date windows
    pickup_date_from  DATE,
    pickup_date_to    DATE,
    deliver_date_from DATE,
    deliver_date_to   DATE,

    -- Details
    vehicle_count     INT NOT NULL DEFAULT 0,
    equipment_type    VARCHAR(50),
    special_instructions TEXT,

    -- Claim mode
    auto_accept     BOOLEAN NOT NULL DEFAULT false,

    -- Status
    status          VARCHAR(20) NOT NULL DEFAULT 'Posted'
                    CHECK (status IN ('Posted', 'Claimed', 'Completed', 'Cancelled', 'Expired')),

    -- Expiration
    expires_at      TIMESTAMPTZ,

    -- Poster company info (denormalized for cross-company display)
    poster_company_name VARCHAR(200),
    poster_scac         VARCHAR(10),
    poster_mc_number    VARCHAR(20),

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loadboard_listings_status ON loadboard_listings(status);
CREATE INDEX idx_loadboard_listings_poster ON loadboard_listings(poster_company_id);
CREATE INDEX idx_loadboard_listings_origin_state ON loadboard_listings(origin_state);
CREATE INDEX idx_loadboard_listings_dest_state ON loadboard_listings(dest_state);
CREATE INDEX idx_loadboard_listings_expires_at ON loadboard_listings(expires_at) WHERE status = 'Posted';

-- Listing vehicles: denormalized snapshot of vehicles in a listing
CREATE TABLE loadboard_listing_vehicles (
    id                SERIAL PRIMARY KEY,
    listing_id        INT NOT NULL REFERENCES loadboard_listings(id) ON DELETE CASCADE,
    source_vehicle_id INT NOT NULL REFERENCES order_vehicles(id),
    vin               VARCHAR(20),
    year              VARCHAR(4),
    make              VARCHAR(50),
    model             VARCHAR(50),
    color             VARCHAR(30),
    weight            INT,
    category          VARCHAR(50),
    body_style        VARCHAR(50),
    operable          BOOLEAN NOT NULL DEFAULT true,
    run_drive         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loadboard_listing_vehicles_listing ON loadboard_listing_vehicles(listing_id);

-- Claims: a carrier's claim on a listing
CREATE TABLE loadboard_claims (
    id                  SERIAL PRIMARY KEY,
    listing_id          INT NOT NULL REFERENCES loadboard_listings(id),
    carrier_company_id  INT NOT NULL REFERENCES companies(id),
    carrier_user_id     INT NOT NULL REFERENCES users(id),

    -- Carrier company info (denormalized)
    carrier_company_name VARCHAR(200),
    carrier_scac         VARCHAR(10),
    carrier_mc_number    VARCHAR(20),
    carrier_dot_number   VARCHAR(20),
    carrier_insurance_exp TIMESTAMPTZ,

    -- The order created in the carrier's system after import
    carrier_order_id    INT REFERENCES orders(id),

    -- Payment
    agreed_pay          NUMERIC(9,2) NOT NULL DEFAULT 0,
    vehicle_count       INT NOT NULL DEFAULT 0,

    -- Status
    status              VARCHAR(20) NOT NULL DEFAULT 'Pending'
                        CHECK (status IN ('Pending', 'Accepted', 'Rejected', 'Cancelled', 'Completed')),

    -- Notes
    carrier_notes       TEXT,
    poster_notes        TEXT,

    -- Timestamps
    accepted_at         TIMESTAMPTZ,
    rejected_at         TIMESTAMPTZ,
    cancelled_at        TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loadboard_claims_listing ON loadboard_claims(listing_id);
CREATE INDEX idx_loadboard_claims_carrier ON loadboard_claims(carrier_company_id);
CREATE INDEX idx_loadboard_claims_status ON loadboard_claims(status);

-- +goose Down
DROP TABLE IF EXISTS loadboard_claims;
DROP TABLE IF EXISTS loadboard_listing_vehicles;
DROP TABLE IF EXISTS loadboard_listings;
DROP SEQUENCE IF EXISTS loadboard_listing_number_seq;
