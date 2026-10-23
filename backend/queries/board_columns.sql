-- name: CreateBoardColumn :one
INSERT INTO board_columns (project_id, name, position, is_done_column, wip_limit)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListBoardColumns :many
SELECT * FROM board_columns
WHERE project_id = $1
ORDER BY position;

-- name: GetBoardColumnByID :one
SELECT * FROM board_columns
WHERE id = $1
LIMIT 1;

-- name: UpdateBoardColumn :one
UPDATE board_columns
SET name      = $2,
    position  = $3,
    wip_limit = $4
WHERE id = $1
RETURNING *;

-- name: DeleteBoardColumn :exec
DELETE FROM board_columns
WHERE id = $1;

-- name: GetDoneColumn :one
SELECT * FROM board_columns
WHERE project_id = $1 AND is_done_column = TRUE
LIMIT 1;

-- name: MaxColumnPosition :one
SELECT COALESCE(MAX(position), 0)::int FROM board_columns WHERE project_id = $1;
