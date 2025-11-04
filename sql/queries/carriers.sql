-- name: ListCarriers :many
SELECT * FROM carriers
WHERE organization_id = $1
ORDER BY company_name ASC
LIMIT $2 OFFSET $3;

-- name: ListActiveCarriers :many
SELECT * FROM carriers
WHERE organization_id = $1 AND active = true
ORDER BY company_name ASC;

-- name: GetCarrier :one
SELECT * FROM carriers
WHERE id = $1 AND organization_id = $2;

-- name: CreateCarrier :one
INSERT INTO carriers (
    organization_id,
    company_name,
    contact_name,
    email,
    phone,
    mc_number,
    dot_number,
    insurance_provider,
    insurance_policy_number,
    insurance_expiry,
    active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: UpdateCarrier :one
UPDATE carriers
SET
    company_name = COALESCE($3, company_name),
    contact_name = COALESCE($4, contact_name),
    email = COALESCE($5, email),
    phone = COALESCE($6, phone),
    mc_number = COALESCE($7, mc_number),
    dot_number = COALESCE($8, dot_number),
    insurance_provider = COALESCE($9, insurance_provider),
    insurance_policy_number = COALESCE($10, insurance_policy_number),
    insurance_expiry = COALESCE($11, insurance_expiry),
    active = COALESCE($12, active)
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeactivateCarrier :one
UPDATE carriers
SET active = false
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: ActivateCarrier :one
UPDATE carriers
SET active = true
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteCarrier :exec
DELETE FROM carriers
WHERE id = $1 AND organization_id = $2;

-- name: SearchCarriers :many
SELECT * FROM carriers
WHERE
    organization_id = $1
    AND (
        company_name ILIKE '%' || $2 || '%'
        OR email ILIKE '%' || $2 || '%'
        OR mc_number ILIKE '%' || $2 || '%'
        OR dot_number ILIKE '%' || $2 || '%'
    )
ORDER BY company_name ASC
LIMIT $3 OFFSET $4;
