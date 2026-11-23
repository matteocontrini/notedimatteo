// Package db holds the PostgreSQL connection pool and SQL queries.
//
// The queries are in queries/*.sql. sqlc generates their Go code (the *.sql.go
// and sqlc*.go files) with `go tool sqlc generate`, checking them against the
// schema in sqlc.yaml. The other files wrap the generated code.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxConns        = 4
	maxConnIdleTime = 5 * time.Minute
	connectTimeout  = 10 * time.Second
)

// currentTime returns the current time truncated to milliseconds, the precision
// of every stored time.
func currentTime() time.Time {
	return time.Now().Truncate(time.Millisecond)
}

type DB struct {
	pool *pgxpool.Pool
	// conn runs queries and starts transactions. It's the pool, except in tests,
	// which use a transaction they roll back at the end.
	conn    conn
	queries *Queries
}

type conn interface {
	DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Open connects to the database and fails if it can't be reached.
func Open(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.MaxConnIdleTime = maxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	return &DB{pool: pool, conn: pool, queries: New(pool)}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}
