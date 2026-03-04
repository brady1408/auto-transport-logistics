-- +goose Up
CREATE TABLE truck_checkins (
    id          BIGSERIAL PRIMARY KEY,
    truck_id    INTEGER NOT NULL REFERENCES trucks(id),
    driver_id   INTEGER NOT NULL REFERENCES users(id),
    company_id  INTEGER NOT NULL,
    latitude    DOUBLE PRECISION NOT NULL,
    longitude   DOUBLE PRECISION NOT NULL,
    accuracy    DOUBLE PRECISION,
    speed       DOUBLE PRECISION,
    heading     DOUBLE PRECISION,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_truck_checkins_truck ON truck_checkins (truck_id, created_at DESC);
CREATE INDEX idx_truck_checkins_company ON truck_checkins (company_id, created_at DESC);

ALTER TABLE load_details ADD COLUMN status_latitude DOUBLE PRECISION;
ALTER TABLE load_details ADD COLUMN status_longitude DOUBLE PRECISION;
ALTER TABLE load_details ADD COLUMN status_location_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE load_details DROP COLUMN IF EXISTS status_latitude;
ALTER TABLE load_details DROP COLUMN IF EXISTS status_longitude;
ALTER TABLE load_details DROP COLUMN IF EXISTS status_location_at;
DROP TABLE IF EXISTS truck_checkins;
