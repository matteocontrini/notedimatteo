package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matteocontrini/notedimatteo/internal/config"
	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/images"
	"github.com/matteocontrini/notedimatteo/internal/images/libvips"
	"github.com/matteocontrini/notedimatteo/internal/markdown"
	"github.com/matteocontrini/notedimatteo/internal/server"
	"github.com/matteocontrini/notedimatteo/internal/storage"
	"github.com/matteocontrini/notedimatteo/web"
)

const shutdownTimeout = 10 * time.Second

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := db.Migrate(ctx, cfg.DatabasePath); err != nil {
		return err
	}

	database, err := db.Open(ctx, cfg.DatabasePath)
	if err != nil {
		return err
	}
	defer database.Close()

	renderer, err := markdown.New(markdown.Options{StorageBaseURL: cfg.StoragePublicBaseURL})
	if err != nil {
		return err
	}

	var uploader *images.Uploader
	if cfg.S3 != nil {
		stopLibvips, err := libvips.Start()
		if err != nil {
			return err
		}
		defer stopLibvips()
		uploader = images.NewUploader(libvips.Processor{}, storage.NewS3(*cfg.S3))
	} else {
		slog.Warn("uploads are off: the STORAGE_S3_* variables aren't set")
	}

	handler, err := server.New(database, server.Options{
		SiteURL:  cfg.SiteURL,
		Dist:     web.Dist(),
		Markdown: renderer,

		StorageBaseURL: cfg.StoragePublicBaseURL,

		AdminUsername: cfg.AdminUsername,
		AdminPassword: cfg.AdminPassword,
		Uploader:      uploader,
	})
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              net.JoinHostPort("", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	slog.Info("server shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shut down: %w", err)
	}
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
