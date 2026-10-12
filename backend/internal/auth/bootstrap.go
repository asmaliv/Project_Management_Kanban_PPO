package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"ppo-kanban/internal/db"
)

// BootstrapAdmin creates the initial system administrator if not already existing.
func BootstrapAdmin(ctx context.Context, queries db.Querier, email, password string, logger *slog.Logger) error {
	if email == "" || password == "" {
		logger.Info("no bootstrap admin credentials configured, skipping")
		return nil
	}

	cleanEmail := strings.ToLower(strings.TrimSpace(email))

	count, err := queries.CountUsersByEmail(ctx, cleanEmail)
	if err != nil {
		return fmt.Errorf("check existing bootstrap admin: %w", err)
	}

	if count > 0 {
		logger.Info("bootstrap admin already exists", slog.String("email", cleanEmail))
		return nil
	}

	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash bootstrap admin password: %w", err)
	}

	_, err = queries.CreateUser(ctx, db.CreateUserParams{
		Email:              cleanEmail,
		FullName:           "System Administrator",
		PasswordHash:       hash,
		Role:               db.UserRoleAdmin,
		IsActive:           true,
		MustChangePassword: true,
	})
	if err != nil {
		return fmt.Errorf("insert bootstrap admin: %w", err)
	}

	logger.Info("bootstrap admin created successfully", slog.String("email", cleanEmail))
	return nil
}
