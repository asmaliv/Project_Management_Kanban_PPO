-- name: CreateTask :one
INSERT INTO tasks (
    project_id, column_id, title, description, priority, assignee_id, reporter_id, due_date, progress_percent, position, labels
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetTaskByID :one
SELECT * FROM tasks
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetTaskByIDWithDetails :one
SELECT 
    t.*,
    u_assignee.full_name AS assignee_name,
    u_reporter.full_name AS reporter_name,
    c.name AS column_name,
    c.is_done_column,
    c.wip_limit,
    p.code AS project_code,
    p.name AS project_name,
    p.owner_id AS project_owner_id
FROM tasks t
LEFT JOIN users u_assignee ON u_assignee.id = t.assignee_id
INNER JOIN users u_reporter ON u_reporter.id = t.reporter_id
INNER JOIN board_columns c ON c.id = t.column_id
INNER JOIN projects p ON p.id = t.project_id
WHERE t.id = $1 AND t.deleted_at IS NULL
LIMIT 1;

-- name: UpdateTask :one
UPDATE tasks
SET title            = $3,
    description      = $4,
    priority         = $5,
    assignee_id      = $6,
    due_date         = $7,
    progress_percent = $8,
    labels           = $9,
    version          = version + 1,
    updated_at       = NOW()
WHERE id = $1 AND version = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteTask :one
UPDATE tasks
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CountActiveTasksInColumn :one
SELECT COUNT(*)::int FROM tasks
WHERE column_id = $1 AND deleted_at IS NULL;

-- name: MaxTaskPositionInColumn :one
SELECT COALESCE(MAX(position), 0)::int FROM tasks
WHERE column_id = $1 AND deleted_at IS NULL;

-- name: ListTasksByProject :many
SELECT 
    t.*,
    u_assignee.full_name AS assignee_name,
    u_reporter.full_name AS reporter_name
FROM tasks t
LEFT JOIN users u_assignee ON u_assignee.id = t.assignee_id
INNER JOIN users u_reporter ON u_reporter.id = t.reporter_id
WHERE t.project_id = $1 AND t.deleted_at IS NULL
ORDER BY t.column_id, t.position ASC;

-- name: ListTasksByColumn :many
SELECT 
    t.*,
    u_assignee.full_name AS assignee_name,
    u_reporter.full_name AS reporter_name
FROM tasks t
LEFT JOIN users u_assignee ON u_assignee.id = t.assignee_id
INNER JOIN users u_reporter ON u_reporter.id = t.reporter_id
WHERE t.column_id = $1 AND t.deleted_at IS NULL
ORDER BY t.position ASC;

-- name: UpdateTaskPositionAndColumn :one
UPDATE tasks
SET column_id    = $2,
    position     = $3,
    version      = version + 1,
    completed_at = $4,
    updated_at   = NOW()
WHERE id = $1 AND version = $5 AND deleted_at IS NULL
RETURNING *;

-- name: ShiftTasksDownInColumn :exec
UPDATE tasks
SET position = position - 1
WHERE column_id = $1 AND position > $2 AND deleted_at IS NULL;

-- name: ShiftTasksUpInColumn :exec
UPDATE tasks
SET position = position + 1
WHERE column_id = $1 AND position >= $2 AND deleted_at IS NULL;

-- name: ShiftTasksBetweenPositionsDown :exec
UPDATE tasks
SET position = position - 1
WHERE column_id = $1 AND position > $2 AND position <= $3 AND deleted_at IS NULL;

-- name: ShiftTasksBetweenPositionsUp :exec
UPDATE tasks
SET position = position + 1
WHERE column_id = $1 AND position >= $2 AND position < $3 AND deleted_at IS NULL;

-- name: ListTasksByProjectsFiltered :many
SELECT 
    t.*,
    u_assignee.full_name AS assignee_name,
    u_reporter.full_name AS reporter_name,
    c.name AS column_name,
    p.code AS project_code,
    p.name AS project_name
FROM tasks t
LEFT JOIN users u_assignee ON u_assignee.id = t.assignee_id
INNER JOIN users u_reporter ON u_reporter.id = t.reporter_id
INNER JOIN board_columns c ON c.id = t.column_id
INNER JOIN projects p ON p.id = t.project_id
WHERE t.deleted_at IS NULL
  AND p.deleted_at IS NULL
  AND (sqlc.narg('project_ids')::uuid[] IS NULL OR t.project_id = ANY(sqlc.narg('project_ids')::uuid[]))
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR t.assignee_id = sqlc.narg('assignee_id'))
  AND (sqlc.narg('column_id')::uuid IS NULL OR t.column_id = sqlc.narg('column_id'))
  AND (sqlc.narg('is_overdue')::boolean IS NULL OR (sqlc.narg('is_overdue')::boolean = TRUE AND t.due_date < CURRENT_DATE AND t.completed_at IS NULL))
ORDER BY t.created_at DESC
LIMIT sqlc.arg('limit_val')::int OFFSET sqlc.arg('offset_val')::int;

-- name: CountTasksByProjectsFiltered :one
SELECT COUNT(*)::bigint
FROM tasks t
INNER JOIN projects p ON p.id = t.project_id
WHERE t.deleted_at IS NULL
  AND p.deleted_at IS NULL
  AND (sqlc.narg('project_ids')::uuid[] IS NULL OR t.project_id = ANY(sqlc.narg('project_ids')::uuid[]))
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR t.assignee_id = sqlc.narg('assignee_id'))
  AND (sqlc.narg('column_id')::uuid IS NULL OR t.column_id = sqlc.narg('column_id'))
  AND (sqlc.narg('is_overdue')::boolean IS NULL OR (sqlc.narg('is_overdue')::boolean = TRUE AND t.due_date < CURRENT_DATE AND t.completed_at IS NULL));
