-- name: CreateTaskComment :one
INSERT INTO task_comments (
    task_id, author_id, body
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: ListTaskComments :many
SELECT 
    tc.*,
    u.full_name AS author_name,
    u.role AS author_role
FROM task_comments tc
INNER JOIN users u ON u.id = tc.author_id
WHERE tc.task_id = $1
ORDER BY tc.created_at ASC;

-- name: CountTaskComments :one
SELECT COUNT(*)::int
FROM task_comments
WHERE task_id = $1;
