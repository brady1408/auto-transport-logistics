-- name: GetOrganization :one
SELECT * FROM organizations
WHERE id = $1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations
WHERE slug = $1;

-- name: CreateOrganization :one
INSERT INTO organizations (
    name,
    slug,
    active
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: UpdateOrganization :one
UPDATE organizations
SET
    name = COALESCE($2, name),
    slug = COALESCE($3, slug),
    active = COALESCE($4, active)
WHERE id = $1
RETURNING *;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY name ASC
LIMIT $1 OFFSET $2;

-- name: CountActiveOrganizations :one
SELECT COUNT(*) FROM organizations
WHERE active = true;
