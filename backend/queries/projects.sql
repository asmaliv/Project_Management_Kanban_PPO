-- name: CreateProject :one
INSERT INTO projects (
    code, name, description, status, owner_id, start_date, end_date, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: NextProjectCode :one
SELECT ('PRJ-' || LPAD(nextval('project_code_seq')::TEXT, 4, '0'))::text;

-- name: GetProjectByID :one
SELECT * FROM projects
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetProjectByIDIncludeDeleted :one
SELECT * FROM projects
WHERE id = $1
LIMIT 1;

-- name: ListProjectsByOwner :many
SELECT * FROM projects
WHERE owner_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListProjectsByMember :many
SELECT p.* FROM projects p
INNER JOIN project_members pm ON pm.project_id = p.id
WHERE pm.user_id = $1 AND p.deleted_at IS NULL
ORDER BY p.created_at DESC;

-- name: ListAllProjects :many
SELECT * FROM projects
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET name        = $2,
    description = $3,
    status      = $4,
    start_date  = $5,
    end_date    = $6,
    updated_at  = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ArchiveProject :one
UPDATE projects
SET status     = 'archived',
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: TransferProjectOwner :one
UPDATE projects
SET owner_id   = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: IsProjectMember :one
SELECT EXISTS (
    SELECT 1 FROM project_members
    WHERE project_id = $1 AND user_id = $2
);
