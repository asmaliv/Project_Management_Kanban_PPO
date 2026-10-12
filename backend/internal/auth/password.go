package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrPasswordTooShort     = errors.New("password minimal 12 karakter")
	ErrPasswordSameAsEmail  = errors.New("password tidak boleh sama dengan email")
	ErrPasswordCommon       = errors.New("password terlalu umum atau mudah ditebak")
	ErrInvalidPasswordHash  = errors.New("format hash password tidak valid")
	ErrPasswordIncompatible = errors.New("versi atau algoritma hash tidak kompatibel")
)

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

var defaultParams = &argon2Params{
	memory:      64 * 1024, // 64 MB
	iterations:  1,
	parallelism: 4,
	saltLength:  16,
	keyLength:   32,
}

var commonPasswords = map[string]struct{}{
	"password1234":     {},
	"password12345":    {},
	"123456789012":     {},
	"administrator":    {},
	"admin1234567":     {},
	"qwerty123456":     {},
	"welcome123456":    {},
	"bankmandiri123":   {},
	"bcapassword12":    {},
	"indonesia123":     {},
	"rahasia12345":     {},
	"projectmgmt12":    {},
}

// ValidatePasswordPolicy checks that the password meets the banking security policy:
// - At least 12 characters
// - Not identical to email or email user part
// - Not in list of common weak passwords
func ValidatePasswordPolicy(password, email string) error {
	if len(password) < 12 {
		return ErrPasswordTooShort
	}

	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanPass := strings.ToLower(strings.TrimSpace(password))

	if cleanPass == cleanEmail {
		return ErrPasswordSameAsEmail
	}

	parts := strings.Split(cleanEmail, "@")
	if len(parts) > 0 && len(parts[0]) >= 3 && cleanPass == parts[0] {
		return ErrPasswordSameAsEmail
	}

	if _, found := commonPasswords[cleanPass]; found {
		return ErrPasswordCommon
	}

	return nil
}

// HashPassword hashes a plaintext password using Argon2id.
func HashPassword(password string) (string, error) {
	salt := make([]byte, defaultParams.saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		defaultParams.iterations,
		defaultParams.memory,
		defaultParams.parallelism,
		defaultParams.keyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		defaultParams.memory,
		defaultParams.iterations,
		defaultParams.parallelism,
		b64Salt,
		b64Hash,
	)

	return encoded, nil
}

// VerifyPassword checks if a password matches an Argon2id hash in constant time.
func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, ErrInvalidPasswordHash
	}

	if parts[1] != "argon2id" {
		return false, ErrPasswordIncompatible
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrPasswordIncompatible
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, ErrInvalidPasswordHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidPasswordHash
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrInvalidPasswordHash
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(expectedHash)),
	)

	if subtle.ConstantTimeCompare(actualHash, expectedHash) == 1 {
		return true, nil
	}

	return false, nil
}
