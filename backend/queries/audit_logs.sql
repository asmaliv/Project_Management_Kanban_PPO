-- name: InsertAuditLog :one
INSERT INTO audit_logs (
    actor_id,
    actor_role,
    action,
    entity_type,
    entity_id,
    project_id,
    before,
    after,
    ip,
    request_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE 
    (sqlc.narg('actor_id')::UUID IS NULL OR actor_id = sqlc.narg('actor_id')::UUID)
    AND (sqlc.narg('action')::TEXT IS NULL OR action = sqlc.narg('action')::TEXT)
    AND (sqlc.narg('entity_type')::TEXT IS NULL OR entity_type = sqlc.narg('entity_type')::TEXT)
    AND (sqlc.narg('entity_id')::TEXT IS NULL OR entity_id = sqlc.narg('entity_id')::TEXT)
    AND (sqlc.narg('project_id')::UUID IS NULL OR project_id = sqlc.narg('project_id')::UUID)
    AND (sqlc.narg('date_from')::TIMESTAMPTZ IS NULL OR occurred_at >= sqlc.narg('date_from')::TIMESTAMPTZ)
    AND (sqlc.narg('date_to')::TIMESTAMPTZ IS NULL OR occurred_at <= sqlc.narg('date_to')::TIMESTAMPTZ)
ORDER BY occurred_at DESC
LIMIT $1 OFFSET $2;
