package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	Env                    string
	Port                   int
	DatabaseURL            string
	SessionSecret          string
	CookieSecure           bool
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

// Load reads configuration from environment variables.
// Returns an error if required variables are missing.
func Load() (*Config, error) {
	cfg := &Config{
		Env:                    getEnv("APP_ENV", "development"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		SessionSecret:          os.Getenv("SESSION_SECRET"),
		BootstrapAdminEmail:    os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
	}

	port, err := strconv.Atoi(getEnv("APP_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("parse APP_PORT: %w", err)
	}
	cfg.Port = port

	cfg.CookieSecure = getEnv("COOKIE_SECURE", "false") == "true"

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.SessionSecret == "" {
		return nil, fmt.Errorf("SESSION_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
