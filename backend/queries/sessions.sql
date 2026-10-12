-- name: CreateSession :one
INSERT INTO sessions (
    user_id, token_hash, expires_at, ip, user_agent
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id, user_id, token_hash, created_at, last_seen_at, expires_at, ip, user_agent;

-- name: GetSessionWithUser :one
SELECT
    s.id AS session_id,
    s.user_id,
    s.token_hash,
    s.created_at AS session_created_at,
    s.last_seen_at,
    s.expires_at,
    s.ip,
    s.user_agent,
    u.email,
    u.full_name,
    u.role,
    u.is_active,
    u.must_change_password
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1
LIMIT 1;

-- name: TouchSessionLastSeen :exec
UPDATE sessions
SET last_seen_at = NOW()
WHERE id = $1;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions
WHERE user_id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE expires_at < NOW();
