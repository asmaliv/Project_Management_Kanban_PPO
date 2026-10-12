package auth

import (
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	raw := "SecurePassword#2026!Bank"

	hash, err := HashPassword(raw)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if hash == "" || hash == raw {
		t.Fatalf("hash should not be empty or equal to raw password")
	}

	match, err := VerifyPassword(raw, hash)
	if err != nil {
		t.Fatalf("unexpected error verifying password: %v", err)
	}
	if !match {
		t.Errorf("expected password to match hash")
	}

	wrongMatch, err := VerifyPassword("WrongPassword#999", hash)
	if err != nil {
		t.Fatalf("unexpected error on wrong password: %v", err)
	}
	if wrongMatch {
		t.Errorf("wrong password should not match")
	}
}

func TestValidatePasswordPolicy(t *testing.T) {
	tests := []struct {
		name     string
		password string
		email    string
		wantErr  error
	}{
		{
			name:     "valid complex password",
			password: "PPOKanbanSecure#2026",
			email:    "admin@ppo.bank.local",
			wantErr:  nil,
		},
		{
			name:     "too short (<12 chars)",
			password: "ShortPass12",
			email:    "user@bank.com",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "same as email",
			password: "admin@ppo.bank.local",
			email:    "admin@ppo.bank.local",
			wantErr:  ErrPasswordSameAsEmail,
		},
		{
			name:     "same as username prefix",
			password: "administrator",
			email:    "administrator@bank.com",
			wantErr:  ErrPasswordSameAsEmail,
		},
		{
			name:     "common weak password",
			password: "password12345",
			email:    "random@bank.com",
			wantErr:  ErrPasswordCommon,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordPolicy(tt.password, tt.email)
			if tt.wantErr == nil && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if tt.wantErr != nil && err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}
