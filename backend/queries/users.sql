-- name: GetUserByEmail :one
SELECT id, email, full_name, password_hash, role, is_active, must_change_password, failed_login_count, locked_until, last_login_at, created_at, updated_at
FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT id, email, full_name, password_hash, role, is_active, must_change_password, failed_login_count, locked_until, last_login_at, created_at, updated_at
FROM users
WHERE id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
    email, full_name, password_hash, role, is_active, must_change_password
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING id, email, full_name, password_hash, role, is_active, must_change_password, failed_login_count, locked_until, last_login_at, created_at, updated_at;

-- name: UpdatePassword :exec
UPDATE users
SET password_hash = $2,
    must_change_password = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: IncrementFailedLogin :one
UPDATE users
SET failed_login_count = failed_login_count + 1,
    updated_at = NOW()
WHERE id = $1
RETURNING failed_login_count;

-- name: LockAccount :exec
UPDATE users
SET locked_until = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: ResetFailedLoginAndTouchLastLogin :exec
UPDATE users
SET failed_login_count = 0,
    locked_until = NULL,
    last_login_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: CountUsersByEmail :one
SELECT COUNT(*) FROM users WHERE email = $1;

-- name: ListUsers :many
SELECT id, email, full_name, role, is_active, must_change_password, failed_login_count, locked_until, last_login_at, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: ListUsersSummary :many
SELECT id, full_name, role
FROM users
WHERE is_active = true
ORDER BY full_name ASC;

-- name: UpdateUser :one
UPDATE users
SET full_name = $2,
    role = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING id, email, full_name, role, is_active, must_change_password, failed_login_count, locked_until, last_login_at, created_at, updated_at;

-- name: SetUserActiveStatus :one
UPDATE users
SET is_active = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING id, email, full_name, role, is_active, must_change_password, failed_login_count, locked_until, last_login_at, created_at, updated_at;

-- name: ResetUserPassword :exec
UPDATE users
SET password_hash = $2,
    must_change_password = true,
    failed_login_count = 0,
    locked_until = NULL,
    updated_at = NOW()
WHERE id = $1;

