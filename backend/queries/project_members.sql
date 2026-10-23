-- name: AddProjectMember :exec
INSERT INTO project_members (project_id, user_id, added_by)
VALUES ($1, $2, $3)
ON CONFLICT (project_id, user_id) DO NOTHING;

-- name: RemoveProjectMember :exec
DELETE FROM project_members
WHERE project_id = $1 AND user_id = $2;

-- name: ListProjectMembers :many
SELECT u.id, u.full_name, u.email, u.role, pm.added_at
FROM project_members pm
INNER JOIN users u ON u.id = pm.user_id
WHERE pm.project_id = $1
ORDER BY pm.added_at;

-- name: CountProjectMembers :one
SELECT COUNT(*) FROM project_members WHERE project_id = $1;
