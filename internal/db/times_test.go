package db

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestStoredTimes checks that the queries store the current time in every
// time column, as text in timeFormat: in UTC, with milliseconds.
func TestStoredTimes(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	start := time.Now().Add(-time.Second)

	number, err := db.CreateDraft(ctx)
	if err != nil {
		t.Fatal(err)
	}
	edit := PostEdit{Number: number, Category: "note", Body: "Body", HTML: "<p>Body</p>\n", Intent: Publish}
	if _, err := db.SavePost(ctx, number, edit); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateSession(ctx, "times-test"); err != nil {
		t.Fatal(err)
	}
	session, err := db.SessionByToken(ctx, "times-test")
	if err != nil || session == nil {
		t.Fatalf("session %v, %v", session, err)
	}
	if err := db.TouchSession(ctx, session.ID); err != nil {
		t.Fatal(err)
	}

	var created, updated, published, rendered, sessionCreated, lastActivity string
	err = db.sql.QueryRowContext(ctx, `
		SELECT p.created_at, p.updated_at, p.published_at, c.rendered_at
		FROM posts p JOIN post_html_cache c ON c.post_id = p.id
		WHERE p.number = ?`, number).Scan(&created, &updated, &published, &rendered)
	if err != nil {
		t.Fatal(err)
	}
	err = db.sql.QueryRowContext(ctx, `SELECT created_at, last_activity_at FROM sessions WHERE token = ?`, "times-test").
		Scan(&sessionCreated, &lastActivity)
	if err != nil {
		t.Fatal(err)
	}

	stored := map[string]string{
		"posts.created_at":            created,
		"posts.updated_at":            updated,
		"posts.published_at":          published,
		"post_html_cache.rendered_at": rendered,
		"sessions.created_at":         sessionCreated,
		"sessions.last_activity_at":   lastActivity,
	}
	for column, text := range stored {
		value, err := time.Parse(timeFormat, text)
		if err != nil || value.Format(timeFormat) != text {
			t.Errorf("%s = %q, not in the format %q", column, text, timeFormat)
			continue
		}
		if value.Before(start) || value.After(time.Now()) {
			t.Errorf("%s = %v, not the current time in UTC", column, value)
		}
	}
}

// TestTimeColumns checks that the tables are STRICT, and that every column
// named *_at has the CHECK that rejects other time formats.
func TestTimeColumns(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	rows, err := db.sql.QueryContext(ctx, `
		SELECT l.name, l.strict, s.sql, c.name
		FROM pragma_table_list l
		JOIN sqlite_schema s ON s.name = l.name
		LEFT JOIN pragma_table_info(l.name) c ON c.name LIKE '%\_at' ESCAPE '\'
		WHERE l.schema = 'main' AND l.type = 'table'
		  AND l.name NOT LIKE 'sqlite\_%' ESCAPE '\' AND l.name != 'goose_db_version'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	columns := 0
	for rows.Next() {
		var table, schema string
		var strict bool
		var column *string
		if err := rows.Scan(&table, &strict, &schema, &column); err != nil {
			t.Fatal(err)
		}
		if !strict {
			t.Errorf("table %s isn't STRICT", table)
		}
		if column == nil {
			continue
		}
		columns++
		check := fmt.Sprintf("CHECK (%[1]s IS strftime('%%Y-%%m-%%d %%H:%%M:%%f', %[1]s))", *column)
		if !strings.Contains(schema, check) {
			t.Errorf("%s.%s has no %s", table, *column, check)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if columns == 0 {
		t.Error("no time columns found")
	}

	// The CHECK rejects a time in another format
	_, err = db.sql.ExecContext(ctx, `UPDATE posts SET created_at = '2026-11-23T10:00:00Z'`)
	if err == nil || !strings.Contains(err.Error(), "CHECK constraint failed") {
		t.Errorf("a time in another format: %v", err)
	}
}
