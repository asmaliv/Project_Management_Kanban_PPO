package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
)

var (
	ErrCSRFTokenMissing = errors.New("header X-CSRF-Token tidak ditemukan")
	ErrCSRFTokenInvalid = errors.New("token CSRF tidak valid")
)

const (
	// CSRFHeader is the required request header for write operations.
	CSRFHeader = "X-CSRF-Token"
)

// GenerateCSRFToken generates an HMAC-SHA256 hex string bound to the raw session token and secret.
func GenerateCSRFToken(sessionToken, sessionSecret string) string {
	mac := hmac.New(sha256.New, []byte(sessionSecret))
	mac.Write([]byte(sessionToken))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyCSRFToken validates the given token against the session token in constant time.
func VerifyCSRFToken(providedToken, sessionToken, sessionSecret string) bool {
	if providedToken == "" || sessionToken == "" || sessionSecret == "" {
		return false
	}

	expected := GenerateCSRFToken(sessionToken, sessionSecret)
	return subtle.ConstantTimeCompare([]byte(providedToken), []byte(expected)) == 1
}
