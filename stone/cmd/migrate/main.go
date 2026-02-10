package main

import (
	"database/sql"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/db"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	goose.SetBaseFS(db.Migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set goose dialect", "error", err)
		os.Exit(1)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	var args []string
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	if err := goose.RunWithOptions(command, sqlDB, "migrations", args, goose.WithAllowMissing()); err != nil {
		slog.Error("migration failed", "command", command, "error", err)
		os.Exit(1)
	}
}
