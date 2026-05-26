package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pulse/stone/internal/api"
	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/scheduler"
	"github.com/pulse/stone/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.IsProduction() {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}

	db, err := store.NewPostgres(cfg)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}

	rdb, err := store.NewRedis(cfg)
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}

	storage, err := store.NewLocalStorage(cfg.StoragePath, cfg.StorageBaseURL)
	if err != nil {
		slog.Error("failed to init storage", "error", err)
		os.Exit(1)
	}

	srv := api.NewServer(cfg, db, rdb, storage)
	mediaSvc := srv.MediaService()
	pathSvc := srv.PathService()

	// Start the background scheduler for periodic cleanup tasks
	// (expired rooms, stale tokens, media recovery, old events, path generation).
	schedulerCtx, schedulerCancel := context.WithCancel(context.Background())
	sched := scheduler.New(db, cfg, mediaSvc.RecoverAssets, pathSvc)
	go sched.Start(schedulerCtx)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      srv.Router(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		slog.Info("starting server", "addr", httpServer.Addr, "env", cfg.Env)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down server")

	// Stop the background scheduler first so no new work is enqueued.
	schedulerCancel()

	// Gracefully shutdown the HTTP server with a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}

	// Drain the media processing queue so in-flight jobs complete.
	mediaSvc.Shutdown()

	slog.Info("server stopped")
}
