package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	// CookieName is the HTTP cookie key for the session.
	CookieName = "ppo_session"

	// IdleTimeout is 30 minutes of inactivity.
	IdleTimeout = 30 * time.Minute

	// AbsoluteTimeout is 8 hours from session creation.
	AbsoluteTimeout = 8 * time.Hour
)

// GenerateSessionToken produces a cryptographically secure 32-byte URL-safe string.
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read secure random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashSessionToken computes the SHA-256 hex digest of the raw session token.
// The raw token is stored in the user's cookie, while only this hash is stored in PostgreSQL.
func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// IsSessionExpired checks both idle (30m) and absolute (8h) timeouts against the given current time.
func IsSessionExpired(lastSeenAt, expiresAt time.Time, now time.Time) (bool, string) {
	if !expiresAt.IsZero() && now.After(expiresAt) {
		return true, "absolute_timeout"
	}
	if !lastSeenAt.IsZero() && now.Sub(lastSeenAt) > IdleTimeout {
		return true, "idle_timeout"
	}
	return false, ""
}
