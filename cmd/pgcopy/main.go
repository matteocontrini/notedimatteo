// Command pgcopy copies the site's PostgreSQL database into a new SQLite file,
// to move the data from PostgreSQL to SQLite:
//
//	go run ./cmd/pgcopy <postgres URL> <sqlite file>
//
// It creates the file with the server's migrations, copies every table in one
// transaction, and checks the result.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ncruces/go-sqlite3/driver"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

// tables are copied in this order, so foreign keys always refer to rows that
// are already there. The columns have the same names in both databases.
var tables = []struct {
	name    string
	columns []string
	orderBy string
}{
	{"posts", []string{"id", "number", "category", "created_at", "updated_at", "published_at",
		"title", "slug", "hide_title", "body", "body_revision"}, "id"},
	{"tags", []string{"name"}, "name"},
	{"posts_tags", []string{"post_id", "tag_name", "position"}, "post_id, position"},
	{"sessions", []string{"id", "token", "created_at", "last_activity_at"}, "id"},
	{"post_html_cache", []string{"post_id", "renderer_version", "body_revision", "html", "rendered_at"}, "post_id"},
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: pgcopy <postgres URL> <sqlite file>")
		os.Exit(2)
	}
	if err := run(context.Background(), os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, "pgcopy:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, postgresURL, path string) error {
	start := time.Now()

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s already exists", path)
	}
	if err := db.Migrate(ctx, path); err != nil {
		return err
	}

	pg, err := pgx.Connect(ctx, postgresURL)
	if err != nil {
		return err
	}
	defer pg.Close(ctx)
	// One read-only snapshot, so the copy is consistent even if something writes
	pgTx, err := pg.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return err
	}
	defer pgTx.Rollback(ctx)

	lite, err := driver.Open("file:" + path + "?_pragma=foreign_keys(on)")
	if err != nil {
		return err
	}
	defer lite.Close()
	tx, err := lite.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range tables {
		count, err := copyTable(ctx, pgTx, tx, table.name, table.columns, table.orderBy)
		if err != nil {
			return fmt.Errorf("copy %s: %w", table.name, err)
		}
		fmt.Printf("%s: %d rows\n", table.name, count)
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if err := check(ctx, pgTx, lite); err != nil {
		return err
	}
	// Move the WAL file's content into the database, so the file is complete on
	// its own
	if _, err := lite.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return err
	}
	fmt.Printf("copied and checked in %v\n", time.Since(start).Round(time.Millisecond))
	return nil
}

func copyTable(ctx context.Context, pgTx pgx.Tx, tx *sql.Tx, name string, columns []string, orderBy string) (int, error) {
	list := strings.Join(columns, ", ")
	rows, err := pgTx.Query(ctx, fmt.Sprintf("SELECT %s FROM %s ORDER BY %s", list, name, orderBy))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", ")
	insert, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", name, list, placeholders))
	if err != nil {
		return 0, err
	}
	defer insert.Close()

	count := 0
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return count, err
		}
		for i, value := range values {
			// Times go in the stored format
			if t, ok := value.(time.Time); ok {
				values[i] = db.Time{Time: t}
			}
		}
		if _, err := insert.ExecContext(ctx, values...); err != nil {
			return count, fmt.Errorf("row %v: %w", values[0], err)
		}
		count++
	}
	return count, rows.Err()
}

// check compares the row counts, and checks the file's integrity and foreign
// keys.
func check(ctx context.Context, pgTx pgx.Tx, lite *sql.DB) error {
	for _, table := range tables {
		var want, got int
		if err := pgTx.QueryRow(ctx, "SELECT count(*) FROM "+table.name).Scan(&want); err != nil {
			return err
		}
		if err := lite.QueryRowContext(ctx, "SELECT count(*) FROM "+table.name).Scan(&got); err != nil {
			return err
		}
		if got != want {
			return fmt.Errorf("%s: %d rows in PostgreSQL, %d in SQLite", table.name, want, got)
		}
	}

	var integrity string
	if err := lite.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil {
		return err
	}
	if integrity != "ok" {
		return fmt.Errorf("integrity check: %s", integrity)
	}

	rows, err := lite.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("foreign key check: rows refer to missing rows")
	}
	return rows.Err()
}
