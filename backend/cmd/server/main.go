package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"ppo-kanban/internal/httpapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Config loading is deferred until database is needed (Phase 1).
	// For Phase 0, we only need the port.
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	router := httpapi.NewRouter(logger)

	addr := fmt.Sprintf(":%s", port)
	logger.Info("server starting", slog.String("addr", addr))

	if err := http.ListenAndServe(addr, router); err != nil {
		logger.Error("server failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
