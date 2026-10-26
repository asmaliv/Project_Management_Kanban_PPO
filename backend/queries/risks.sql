-- name: GetNextRiskCode :one
SELECT ('RSK-' || LPAD(nextval('risk_code_seq')::TEXT, 4, '0'))::text AS code;

-- name: CreateRisk :one
INSERT INTO risks (
    code, project_id, task_id, title, description, category,
    likelihood, impact, level, status, mitigation_plan, owner_id, raised_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetRiskByID :one
SELECT * FROM risks
WHERE id = $1
LIMIT 1;

-- name: GetRiskByIDWithDetails :one
SELECT 
    r.*,
    p.name AS project_name,
    p.code AS project_code,
    p.owner_id AS project_owner_id,
    t.title AS task_title,
    u_owner.full_name AS owner_name,
    u_raised.full_name AS raised_by_name
FROM risks r
INNER JOIN projects p ON p.id = r.project_id
LEFT JOIN tasks t ON t.id = r.task_id
LEFT JOIN users u_owner ON u_owner.id = r.owner_id
INNER JOIN users u_raised ON u_raised.id = r.raised_by
WHERE r.id = $1 AND p.deleted_at IS NULL
LIMIT 1;

-- name: UpdateRiskByRiskManagement :one
UPDATE risks
SET title           = $2,
    description     = $3,
    category        = $4,
    task_id         = $5,
    owner_id        = $6,
    status          = $7,
    mitigation_plan = $8,
    updated_at      = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateRiskMitigationPlan :one
UPDATE risks
SET mitigation_plan = $2,
    status          = CASE WHEN status = 'identified' THEN 'mitigating'::risk_status ELSE status END,
    updated_at      = NOW()
WHERE id = $1
RETURNING *;

-- name: AssessRisk :one
UPDATE risks
SET likelihood      = $2,
    impact          = $3,
    level           = $4,
    status          = CASE 
                        WHEN status = 'closed' THEN 'closed'::risk_status
                        WHEN LENGTH(TRIM(COALESCE($5, mitigation_plan))) > 0 THEN 'mitigating'::risk_status
                        ELSE 'assessed'::risk_status
                      END,
    mitigation_plan = COALESCE($5, mitigation_plan),
    updated_at      = NOW()
WHERE id = $1
RETURNING *;

-- name: CloseRisk :one
UPDATE risks
SET status     = 'closed',
    closed_at  = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListRisksFiltered :many
SELECT 
    r.*,
    p.name AS project_name,
    p.code AS project_code,
    p.owner_id AS project_owner_id,
    t.title AS task_title,
    u_owner.full_name AS owner_name,
    u_raised.full_name AS raised_by_name
FROM risks r
INNER JOIN projects p ON p.id = r.project_id
LEFT JOIN tasks t ON t.id = r.task_id
LEFT JOIN users u_owner ON u_owner.id = r.owner_id
INNER JOIN users u_raised ON u_raised.id = r.raised_by
WHERE p.deleted_at IS NULL
  AND (sqlc.narg('project_ids')::uuid[] IS NULL OR r.project_id = ANY(sqlc.narg('project_ids')::uuid[]))
  AND (sqlc.narg('project_id')::uuid IS NULL OR r.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('task_id')::uuid IS NULL OR r.task_id = sqlc.narg('task_id'))
  AND (sqlc.narg('level')::risk_level IS NULL OR r.level = sqlc.narg('level'))
  AND (sqlc.narg('status')::risk_status IS NULL OR r.status = sqlc.narg('status'))
  AND (sqlc.narg('category')::risk_category IS NULL OR r.category = sqlc.narg('category'))
  AND (sqlc.narg('search')::text IS NULL OR (
      r.title ILIKE ('%' || sqlc.narg('search')::text || '%') OR
      r.code ILIKE ('%' || sqlc.narg('search')::text || '%') OR
      r.description ILIKE ('%' || sqlc.narg('search')::text || '%')
  ))
ORDER BY r.created_at DESC
LIMIT sqlc.arg('limit_val')::int OFFSET sqlc.arg('offset_val')::int;

-- name: CountRisksFiltered :one
SELECT COUNT(*)::bigint
FROM risks r
INNER JOIN projects p ON p.id = r.project_id
WHERE p.deleted_at IS NULL
  AND (sqlc.narg('project_ids')::uuid[] IS NULL OR r.project_id = ANY(sqlc.narg('project_ids')::uuid[]))
  AND (sqlc.narg('project_id')::uuid IS NULL OR r.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('task_id')::uuid IS NULL OR r.task_id = sqlc.narg('task_id'))
  AND (sqlc.narg('level')::risk_level IS NULL OR r.level = sqlc.narg('level'))
  AND (sqlc.narg('status')::risk_status IS NULL OR r.status = sqlc.narg('status'))
  AND (sqlc.narg('category')::risk_category IS NULL OR r.category = sqlc.narg('category'))
  AND (sqlc.narg('search')::text IS NULL OR (
      r.title ILIKE ('%' || sqlc.narg('search')::text || '%') OR
      r.code ILIKE ('%' || sqlc.narg('search')::text || '%') OR
      r.description ILIKE ('%' || sqlc.narg('search')::text || '%')
  ));

-- name: GetRisksHeatmapCounts :many
SELECT 
    r.likelihood,
    r.impact,
    COUNT(*)::bigint AS count
FROM risks r
INNER JOIN projects p ON p.id = r.project_id
WHERE p.deleted_at IS NULL
  AND r.likelihood IS NOT NULL
  AND r.impact IS NOT NULL
  AND (sqlc.narg('project_ids')::uuid[] IS NULL OR r.project_id = ANY(sqlc.narg('project_ids')::uuid[]))
  AND (sqlc.narg('project_id')::uuid IS NULL OR r.project_id = sqlc.narg('project_id'))
  AND (sqlc.narg('include_closed')::boolean = TRUE OR r.status != 'closed')
GROUP BY r.likelihood, r.impact;

-- name: GetRisksSummaryStats :one
SELECT 
    COUNT(*)::bigint AS total_risks,
    COUNT(*) FILTER (WHERE r.status != 'closed')::bigint AS open_risks,
    COUNT(*) FILTER (WHERE r.level IN ('high', 'critical') AND r.status != 'closed')::bigint AS high_critical_risks,
    COUNT(*) FILTER (WHERE (r.mitigation_plan = '' OR r.mitigation_plan IS NULL) AND r.status != 'closed')::bigint AS unmitigated_risks,
    COUNT(*) FILTER (WHERE r.level = 'low' AND r.status != 'closed')::bigint AS low_level_count,
    COUNT(*) FILTER (WHERE r.level = 'medium' AND r.status != 'closed')::bigint AS medium_level_count,
    COUNT(*) FILTER (WHERE r.level = 'high' AND r.status != 'closed')::bigint AS high_level_count,
    COUNT(*) FILTER (WHERE r.level = 'critical' AND r.status != 'closed')::bigint AS critical_level_count
FROM risks r
INNER JOIN projects p ON p.id = r.project_id
WHERE p.deleted_at IS NULL
  AND (sqlc.narg('project_ids')::uuid[] IS NULL OR r.project_id = ANY(sqlc.narg('project_ids')::uuid[]))
  AND (sqlc.narg('project_id')::uuid IS NULL OR r.project_id = sqlc.narg('project_id'));

-- name: CountRisksByTaskIDs :many
SELECT 
    r.task_id,
    COUNT(*)::bigint AS count,
    COUNT(*) FILTER (WHERE r.level IN ('high', 'critical'))::bigint AS high_critical_count
FROM risks r
WHERE r.task_id = ANY(sqlc.arg('task_ids')::uuid[])
GROUP BY r.task_id;
