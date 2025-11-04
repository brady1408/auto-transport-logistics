-- name: ListCustomers :many
SELECT * FROM customers
WHERE organization_id = $1
ORDER BY name ASC
LIMIT $2 OFFSET $3;

-- name: GetCustomer :one
SELECT * FROM customers
WHERE id = $1 AND organization_id = $2;

-- name: GetCustomerByEmail :one
SELECT * FROM customers
WHERE email = $1 AND organization_id = $2;

-- name: CreateCustomer :one
INSERT INTO customers (
    organization_id,
    name,
    contact_name,
    email,
    phone,
    address_line1,
    address_line2,
    city,
    state,
    zip_code,
    country
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET
    name = COALESCE($3, name),
    contact_name = COALESCE($4, contact_name),
    email = COALESCE($5, email),
    phone = COALESCE($6, phone),
    address_line1 = COALESCE($7, address_line1),
    address_line2 = COALESCE($8, address_line2),
    city = COALESCE($9, city),
    state = COALESCE($10, state),
    zip_code = COALESCE($11, zip_code),
    country = COALESCE($12, country)
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customers
WHERE id = $1 AND organization_id = $2;

-- name: SearchCustomers :many
SELECT * FROM customers
WHERE
    organization_id = $1
    AND (
        name ILIKE '%' || $2 || '%'
        OR email ILIKE '%' || $2 || '%'
        OR phone ILIKE '%' || $2 || '%'
    )
ORDER BY name ASC
LIMIT $3 OFFSET $4;
