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
		SELECT p."createdAt", p."updatedAt", p."publishedAt", c."renderedAt"
		FROM posts p JOIN post_html_cache c ON c."postId" = p.id
		WHERE p.number = $1`, number).Scan(&created, &updated, &published, &rendered)
	if err != nil {
		t.Fatal(err)
	}
	err = db.conn.QueryRow(ctx, `SELECT "createdAt", "lastActivityAt" FROM sessions WHERE token = $1`, "times-test").
		Scan(&sessionCreated, &lastActivity)
	if err != nil {
		t.Fatal(err)
	}

	times := map[string]time.Time{
		"posts.createdAt":            created,
		"posts.updatedAt":            updated,
		"posts.publishedAt":          published,
		"post_html_cache.renderedAt": rendered,
		"sessions.createdAt":         sessionCreated,
		"sessions.lastActivityAt":    lastActivity,
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
