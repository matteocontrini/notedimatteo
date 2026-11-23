package db

import (
	"context"
	"testing"
	"time"
)

// TestStoredTimes checks that the queries store the current time, truncated to
// milliseconds, in every time column.
func TestStoredTimes(t *testing.T) {
	db := openTestTx(t)
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

	var created, updated, published, rendered, sessionCreated, lastActivity time.Time
	err = db.conn.QueryRow(ctx, `
		SELECT p.created_at, p.updated_at, p.published_at, c.rendered_at
		FROM posts p JOIN post_html_cache c ON c.post_id = p.id
		WHERE p.number = $1`, number).Scan(&created, &updated, &published, &rendered)
	if err != nil {
		t.Fatal(err)
	}
	err = db.conn.QueryRow(ctx, `SELECT created_at, last_activity_at FROM sessions WHERE token = $1`, "times-test").
		Scan(&sessionCreated, &lastActivity)
	if err != nil {
		t.Fatal(err)
	}

	times := map[string]time.Time{
		"posts.created_at":            created,
		"posts.updated_at":            updated,
		"posts.published_at":          published,
		"post_html_cache.rendered_at": rendered,
		"sessions.created_at":         sessionCreated,
		"sessions.last_activity_at":   lastActivity,
	}
	for column, value := range times {
		if value.Before(start) || value.After(time.Now()) {
			t.Errorf("%s = %v, not the current time", column, value)
		}
		if value.Nanosecond()%int(time.Millisecond) != 0 {
			t.Errorf("%s = %v, not in milliseconds", column, value)
		}
	}
}
