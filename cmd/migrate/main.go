// Command migrate applies the database migrations in internal/db/migrations to
// the database at DATABASE_URL. The app doesn't apply migrations at startup.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL is required")
	}
	if err := db.Migrate(ctx, url); err != nil {
		return err
	}
	slog.Info("the database is up to date")
	return nil
}
