package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/config"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/httpapi"
	"ppo-kanban/internal/users"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		logger.Error("invalid database url", slog.String("error", err.Error()))
		os.Exit(1)
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.Error("failed to connect to database pool", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		logger.Warn("could not ping database at startup", slog.String("error", err.Error()))
	} else {
		logger.Info("connected to database successfully")
	}

	queries := db.New(dbPool)

	// Bootstrap initial admin if configured
	if err := auth.BootstrapAdmin(ctx, queries, cfg.BootstrapAdminEmail, cfg.BootstrapAdminPassword, logger); err != nil {
		logger.Error("failed to bootstrap admin", slog.String("error", err.Error()))
	}

	authService := auth.NewService(queries, cfg.SessionSecret, logger)
	userService := users.NewService(queries, logger)

	router := httpapi.NewRouter(httpapi.RouterParams{
		Logger:        logger,
		AuthService:   authService,
		UserService:   userService,
		SessionSecret: cfg.SessionSecret,
		CookieSecure:  cfg.CookieSecure,
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Info("server starting", slog.String("addr", addr), slog.String("env", cfg.Env))

	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
