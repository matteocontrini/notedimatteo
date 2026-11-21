package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// The migrations are goose SQL files named like 00002_add_revisions.sql. sqlc
// reads them as the schema.
//
//go:embed migrations/*.sql
var migrations embed.FS

// Migrate applies the migrations that the database doesn't have yet. goose
// records the applied ones in the goose_db_version table.
func Migrate(ctx context.Context, url string) error {
	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer sqlDB.Close()

	provider, err := newMigrationProvider(sqlDB)
	if err != nil {
		return err
	}

	results, err := provider.Up(ctx)
	for _, result := range results {
		slog.Info("applied migration", "file", result.Source.Path, "duration", result.Duration)
	}
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// newMigrationProvider reads the embedded migrations. It doesn't connect to
// the database.
func newMigrationProvider(sqlDB *sql.DB) (*goose.Provider, error) {
	files, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, files)
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	return provider, nil
}
