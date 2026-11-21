package db

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// testDatabaseURL is the database the tests use, or "" with -short, which
// skips them.
var testDatabaseURL string

// testSeed is the data in the test database: published posts in two categories,
// two months, and with tags, plus a draft.
const testSeed = `
INSERT INTO posts (number, category, title, slug, "hideTitle", body, "createdAt", "publishedAt", "updatedAt") VALUES
	(1, 'note', NULL, NULL, FALSE, 'Prima nota', '2026-09-10 10:00Z', '2026-09-10 10:00Z', NULL),
	(2, 'til', 'Un titolo', 'un-titolo', FALSE, 'Seconda nota, con un titolo', '2026-09-20 10:00Z', '2026-09-20 10:00Z', NULL),
	(3, 'note', NULL, NULL, FALSE, 'Terza nota', '2026-10-05 10:00Z', '2026-10-05 10:00Z', NULL),
	(4, 'note', NULL, NULL, FALSE, 'Quarta nota', '2026-10-15 10:00Z', '2026-10-15 10:00Z', NULL),
	(5, 'note', NULL, NULL, FALSE, 'Una bozza', '2026-10-20 10:00Z', NULL, NULL);
INSERT INTO tags (name) VALUES ('go'), ('svelte');
INSERT INTO posts_tags ("postId", "tagName")
SELECT id, 'go' FROM posts WHERE number IN (1, 3, 5)
UNION ALL
SELECT id, 'svelte' FROM posts WHERE number = 2;
`

// TestMain starts a Postgres container with the migrations and testSeed.
// TEST_DATABASE_URL replaces it with another database, as it is, for example a
// copy of production data.
func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	if testing.Short() {
		return m.Run()
	}
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		testDatabaseURL = url
		return m.Run()
	}

	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:18",
		postgres.WithDatabase("notedimatteo"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start Postgres (run with -short to skip the database tests):", err)
		return 1
	}
	defer container.Terminate(ctx)

	testDatabaseURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err == nil {
		err = Migrate(ctx, testDatabaseURL)
	}
	if err == nil {
		err = seed(ctx, testDatabaseURL)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "prepare the test database:", err)
		return 1
	}
	return m.Run()
}

func seed(ctx context.Context, url string) error {
	db, err := Open(ctx, url)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.pool.Exec(ctx, testSeed)
	return err
}

// openTestDB connects to the test database, or skips the test with -short.
// Tests that write use openTestTx, so the tests can run against a copy of
// production data.
func openTestDB(t *testing.T) *DB {
	t.Helper()

	if testDatabaseURL == "" {
		t.Skip("database tests are skipped with -short")
	}
	db, err := Open(context.Background(), testDatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}

// openTestTx is openTestDB inside a transaction that's rolled back when the
// test ends, so the test can write.
func openTestTx(t *testing.T) *DB {
	t.Helper()

	db := openTestDB(t)
	tx, err := db.pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback(context.Background()) })
	return &DB{pool: db.pool, conn: tx, queries: New(tx)}
}

// TestPostsAndCountAgree checks that Posts and CountPosts, whose WHERE clauses
// are written twice in queries/posts.sql, select the same posts.
func TestPostsAndCountAgree(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	tags, err := db.Tags(ctx)
	if err != nil || len(tags) == 0 {
		t.Fatalf("no tags: %v", err)
	}
	categories, err := db.Categories(ctx)
	if err != nil || len(categories) == 0 {
		t.Fatalf("no categories: %v", err)
	}
	months, err := db.Archive(ctx)
	if err != nil || len(months) == 0 {
		t.Fatalf("no months: %v", err)
	}
	from := time.Date(months[0].Year, time.Month(months[0].Month), 1, 0, 0, 0, 0, time.UTC)

	filters := []PostFilter{
		{},
		{Category: categories[0].Slug},
		{Tag: tags[0].Name},
		{Search: "a"},
		{Search: "%"},
		{From: from, Until: from.AddDate(0, 1, 0)},
		{Category: categories[0].Slug, Search: "e"},
		{Tag: "no such tag"},
	}

	for _, filter := range filters {
		posts, err := db.Posts(ctx, filter, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		count, err := db.CountPosts(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		if len(posts) != count {
			t.Errorf("%+v: Posts returned %d, CountPosts %d", filter, len(posts), count)
		}
	}

	if tagged, _ := db.CountPosts(ctx, PostFilter{Tag: tags[0].Name}); tagged != tags[0].Count {
		t.Errorf("tag %s: %d posts, Tags says %d", tags[0].Name, tagged, tags[0].Count)
	}
}

func TestPostsPagination(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	all, err := db.Posts(ctx, PostFilter{}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) < 3 {
		t.Skip("not enough posts")
	}

	page, err := db.Posts(ctx, PostFilter{}, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 || page[0].ID != all[1].ID || page[1].ID != all[2].ID {
		t.Errorf("limit 2 offset 1 returned the wrong posts")
	}

	post, err := db.PostByNumber(ctx, all[0].Number)
	if err != nil || post == nil || post.ID != all[0].ID {
		t.Errorf("PostByNumber(%d) = %v, %v", all[0].Number, post, err)
	}
	for _, number := range []int{-1, 0} {
		if post, err := db.PostByNumber(ctx, number); (post != nil && post.Number != number) || err != nil {
			t.Errorf("PostByNumber(%d) = %v, %v", number, post, err)
		}
	}
}

func TestSearchEscapesWildcards(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	all, _ := db.CountPosts(ctx, PostFilter{})
	for _, wildcard := range []string{"%", "_"} {
		matches, err := db.CountPosts(ctx, PostFilter{Search: wildcard})
		if err != nil {
			t.Fatal(err)
		}
		if matches == all && all > 0 {
			t.Errorf("searching %q matched every post", wildcard)
		}
	}
}
