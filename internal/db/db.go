// Package db holds the SQLite database and its SQL queries.
//
// The queries are in queries/*.sql. sqlc generates their Go code (the *.sql.go
// and sqlc*.go files) with `go tool sqlc generate`, checking them against the
// schema in sqlc.yaml. The other files wrap the generated code.
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/unicode"
)

// maxConns is the number of connections. One can read while another writes.
const maxConns = 2

type DB struct {
	sql     *sql.DB
	queries *Queries
}

// Open opens the database file at path, creating it if it doesn't exist.
func Open(ctx context.Context, path string) (*DB, error) {
	sqlDB, err := openSQL(path)
	if err != nil {
		return nil, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &DB{sql: sqlDB, queries: New(sqlDB)}, nil
}

// openSQL opens path with the settings every connection needs.
func openSQL(path string) (*sql.DB, error) {
	params := url.Values{}
	// Transactions take the write lock when they begin, so a transaction that
	// reads and then writes never fails because another one wrote in between.
	params.Set("_txlock", "immediate")
	params.Add("_pragma", "busy_timeout(5000)")
	params.Add("_pragma", "journal_mode(wal)")
	params.Add("_pragma", "synchronous(normal)")
	// Without foreign keys on, deletes don't cascade.
	params.Add("_pragma", "foreign_keys(on)")

	sqlDB, err := driver.Open("file:"+path+"?"+params.Encode(), func(conn *sqlite3.Conn) error {
		// LIKE, upper, and lower handle every letter, not only ASCII ones
		return unicode.Register(conn)
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB.SetMaxOpenConns(maxConns)
	sqlDB.SetMaxIdleConns(maxConns)
	return sqlDB, nil
}

func (db *DB) Close() {
	db.sql.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.sql.PingContext(ctx)
}

// decodeTags decodes the JSON array of tag names that queries return.
func decodeTags(tags string) ([]string, error) {
	var names []string
	if err := json.Unmarshal([]byte(tags), &names); err != nil {
		return nil, fmt.Errorf("decode tags: %w", err)
	}
	return names, nil
}
