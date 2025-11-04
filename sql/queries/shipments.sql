-- name: ListShipments :many
SELECT
    s.*,
    c.name as customer_name,
    ca.company_name as carrier_name
FROM shipments s
LEFT JOIN customers c ON s.customer_id = c.id
LEFT JOIN carriers ca ON s.carrier_id = ca.id
WHERE s.organization_id = $1
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetShipment :one
SELECT
    s.*,
    c.name as customer_name,
    c.email as customer_email,
    c.phone as customer_phone,
    ca.company_name as carrier_name,
    ca.contact_name as carrier_contact_name,
    ca.phone as carrier_phone
FROM shipments s
LEFT JOIN customers c ON s.customer_id = c.id
LEFT JOIN carriers ca ON s.carrier_id = ca.id
WHERE s.id = $1 AND s.organization_id = $2;

-- name: CreateShipment :one
INSERT INTO shipments (
    organization_id,
    customer_id,
    pickup_address_line1,
    pickup_city,
    pickup_state,
    pickup_zip_code,
    delivery_address_line1,
    delivery_city,
    delivery_state,
    delivery_zip_code,
    pickup_date_requested,
    delivery_date_estimated,
    price_quoted,
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING *;

-- name: UpdateShipment :one
UPDATE shipments
SET
    carrier_id = COALESCE($3, carrier_id),
    status = COALESCE($4, status),
    pickup_date_actual = COALESCE($5, pickup_date_actual),
    delivery_date_actual = COALESCE($6, delivery_date_actual),
    price_actual = COALESCE($7, price_actual),
    notes = COALESCE($8, notes)
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpdateShipmentStatus :one
UPDATE shipments
SET status = $3
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: AssignCarrier :one
UPDATE shipments
SET
    carrier_id = $3,
    status = 'assigned'
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteShipment :exec
DELETE FROM shipments
WHERE id = $1 AND organization_id = $2;

-- name: CountShipments :one
SELECT COUNT(*) FROM shipments
WHERE organization_id = $1;

-- name: CountShipmentsByStatus :one
SELECT COUNT(*) FROM shipments
WHERE organization_id = $1 AND status = $2;
