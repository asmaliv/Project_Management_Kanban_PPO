-- name: GetNextFindingCode :one
SELECT ('FND-' || LPAD(nextval('audit_finding_code_seq')::TEXT, 4, '0'))::text AS code;

-- name: CreateAuditFinding :one
INSERT INTO audit_findings (
    code, project_id, task_id, title, description, severity, status, raised_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetAuditFindingByID :one
SELECT * FROM audit_findings
WHERE id = $1
LIMIT 1;

-- name: GetAuditFindingByIDWithDetails :one
SELECT 
    f.*,
    p.name AS project_name,
    p.code AS project_code,
    p.owner_id AS project_owner_id,
    t.title AS task_title,
    u_raised.full_name AS raised_by_name,
    u_responded.full_name AS responded_by_name,
    u_closed.full_name AS closed_by_name
FROM audit_findings f
INNER JOIN projects p ON p.id = f.project_id
LEFT JOIN tasks t ON t.id = f.task_id
INNER JOIN users u_raised ON u_raised.id = f.raised_by
LEFT JOIN users u_responded ON u_responded.id = f.responded_by
LEFT JOIN users u_closed ON u_closed.id = f.closed_by
WHERE f.id = $1 AND p.deleted_at IS NULL
LIMIT 1;

-- name: RespondAuditFinding :one
UPDATE audit_findings
SET response     = $2,
    responded_by = $3,
    responded_at = NOW(),
    status       = 'responded',
    updated_at   = NOW()
WHERE id = $1
RETURNING *;

-- name: ReopenAuditFinding :one
UPDATE audit_findings
SET status     = 'open',
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: CloseAuditFinding :one
UPDATE audit_findings
SET status     = 'closed',
    closed_by  = $2,
    closed_at  = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListAuditFindingsFiltered :many
SELECT 
    f.*,
    p.name AS project_name,
    p.code AS project_code,
    p.owner_id AS project_owner_id,
    t.title AS task_title,
    u_raised.full_name AS raised_by_name,
    u_responded.full_name AS responded_by_name,
    u_closed.full_name AS closed_by_name
FROM audit_findings f
INNER JOIN projects p ON p.id = f.project_id
LEFT JOIN tasks t ON t.id = f.task_id
INNER JOIN users u_raised ON u_raised.id = f.raised_by
LEFT JOIN users u_responded ON u_responded.id = f.responded_by
LEFT JOIN users u_closed ON u_closed.id = f.closed_by
WHERE p.deleted_at IS NULL
  AND (sqlc.narg('project_ids')::uuid[] IS NULL OR f.project_id = ANY(sqlc.narg('project_ids')::uuid[]))
  AND (sqlc.narg('project_id')::uuid IS NULL OR f.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('task_id')::uuid IS NULL OR f.task_id = sqlc.narg('task_id'))
  AND (sqlc.narg('severity')::finding_severity IS NULL OR f.severity = sqlc.narg('severity'))
  AND (sqlc.narg('status')::finding_status IS NULL OR f.status = sqlc.narg('status'))
  AND (sqlc.narg('search')::text IS NULL OR (
      f.title ILIKE ('%' || sqlc.narg('search')::text || '%') OR
      f.code ILIKE ('%' || sqlc.narg('search')::text || '%') OR
      f.description ILIKE ('%' || sqlc.narg('search')::text || '%')
  ))
ORDER BY f.created_at DESC
LIMIT sqlc.arg('limit_val')::int OFFSET sqlc.arg('offset_val')::int;

-- name: CountAuditFindingsFiltered :one
SELECT COUNT(*)::bigint
FROM audit_findings f
INNER JOIN projects p ON p.id = f.project_id
WHERE p.deleted_at IS NULL
  AND (sqlc.narg('project_ids')::uuid[] IS NULL OR f.project_id = ANY(sqlc.narg('project_ids')::uuid[]))
  AND (sqlc.narg('project_id')::uuid IS NULL OR f.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('task_id')::uuid IS NULL OR f.task_id = sqlc.narg('task_id'))
  AND (sqlc.narg('severity')::finding_severity IS NULL OR f.severity = sqlc.narg('severity'))
  AND (sqlc.narg('status')::finding_status IS NULL OR f.status = sqlc.narg('status'))
  AND (sqlc.narg('search')::text IS NULL OR (
      f.title ILIKE ('%' || sqlc.narg('search')::text || '%') OR
      f.code ILIKE ('%' || sqlc.narg('search')::text || '%') OR
      f.description ILIKE ('%' || sqlc.narg('search')::text || '%')
  ));

-- name: GetAuditFindingsSummaryStats :one
SELECT 
    COUNT(*)::bigint AS total_findings,
    COUNT(*) FILTER (WHERE f.status != 'closed')::bigint AS open_findings,
    COUNT(*) FILTER (WHERE f.severity = 'critical' AND f.status != 'closed')::bigint AS critical_count,
    COUNT(*) FILTER (WHERE f.severity = 'high' AND f.status != 'closed')::bigint AS high_count,
    COUNT(*) FILTER (WHERE f.severity = 'medium' AND f.status != 'closed')::bigint AS medium_count,
    COUNT(*) FILTER (WHERE f.severity = 'low' AND f.status != 'closed')::bigint AS low_count,
    COUNT(*) FILTER (WHERE f.status = 'responded')::bigint AS responded_count
FROM audit_findings f
INNER JOIN projects p ON p.id = f.project_id
WHERE p.deleted_at IS NULL
  AND (sqlc.narg('project_ids')::uuid[] IS NULL OR f.project_id = ANY(sqlc.narg('project_ids')::uuid[]))
  AND (sqlc.narg('project_id')::uuid IS NULL OR f.project_id = sqlc.narg('project_id'));
