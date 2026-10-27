package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"ppo-kanban/internal/auditlog"
	"ppo-kanban/internal/auth"
	"ppo-kanban/internal/config"
	"ppo-kanban/internal/db"
	"ppo-kanban/internal/finding"
	"ppo-kanban/internal/httpapi"
	"ppo-kanban/internal/project"
	"ppo-kanban/internal/risk"
	"ppo-kanban/internal/task"
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

	// Create audit log store and service
	auditStore := auditlog.NewStore(queries)
	auditService := auditlog.NewService(auditStore, logger)

	// Bootstrap initial admin if configured
	if err := auth.BootstrapAdmin(ctx, queries, cfg.BootstrapAdminEmail, cfg.BootstrapAdminPassword, logger); err != nil {
		logger.Error("failed to bootstrap admin", slog.String("error", err.Error()))
	}

	authService := auth.NewService(queries, cfg.SessionSecret, logger, auditService)
	projectService := project.NewService(dbPool, queries, auditService, logger)
	taskService := task.NewService(dbPool, queries, auditService, logger)
	riskService := risk.NewService(dbPool, queries, auditService, logger)
	findingService := finding.NewService(dbPool, queries, auditService, logger)

	router := httpapi.NewRouter(httpapi.RouterParams{
		Logger:         logger,
		AuthService:    authService,
		AuditService:   auditService,
		ProjectService: projectService,
		TaskService:    taskService,
		RiskService:    riskService,
		FindingService: findingService,
		SessionSecret:  cfg.SessionSecret,
		CookieSecure:   cfg.CookieSecure,
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Info("server starting", slog.String("addr", addr), slog.String("env", cfg.Env))

	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
