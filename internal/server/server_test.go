package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/web"
)

type fakeStore struct {
	posts []db.Post
	err   error
}

func (f fakeStore) Ping(context.Context) error {
	return f.err
}

func (f fakeStore) LatestPosts(context.Context, int) ([]db.Post, error) {
	return f.posts, f.err
}

// testDist stands in for the Vite output, so tests don't depend on its hashes.
func testDist() fstest.MapFS {
	return fstest.MapFS{
		".vite/manifest.json":    {Data: []byte(`{"assets/main.css":{"file":"assets/main-abc123.css"}}`)},
		"assets/main-abc123.css": {Data: []byte("body{}")},
		"favicon.png":            {Data: []byte("png")},
	}
}

func get(t *testing.T, store Store, path string) *httptest.ResponseRecorder {
	t.Helper()

	handler, err := New(store, web.Templates(), testDist())
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHomeRendersPosts(t *testing.T) {
	cached := "<p>Cached <del>body</del></p>"
	store := fakeStore{posts: []db.Post{
		{
			Number:      42,
			Category:    "note",
			Title:       "A <title>",
			Slug:        "a-slug",
			Body:        "Markdown",
			PublishedAt: time.Now(),
			CachedHTML:  &cached,
		},
		{Number: 41, Title: "Hidden title", HideTitle: true, Body: "Uncached <b>body</b>"},
	}}

	rec := get(t, store, "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		`href="/assets/main-abc123.css"`,
		`<title>Note di Matteo</title>`,
		`data-category="note"`,
		`A &lt;title&gt;`,
		cached,
		`href="/42/a-slug"`,
		`href="/41"`,
		`Uncached &lt;b&gt;body&lt;/b&gt;`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response is missing %q", want)
		}
	}
	if strings.Contains(body, "Hidden title") {
		t.Error("response contains a hidden title")
	}
}

func TestHomeWithoutPosts(t *testing.T) {
	rec := get(t, fakeStore{}, "/")

	if !strings.Contains(rec.Body.String(), "Non c'è niente qui.") {
		t.Errorf("response is missing the empty message: %s", rec.Body)
	}
}

func TestHomeDatabaseError(t *testing.T) {
	rec := get(t, fakeStore{err: errors.New("secret connection details")}, "/")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Error("response leaks the error")
	}
}

func TestHealthz(t *testing.T) {
	if rec := get(t, fakeStore{}, "/healthz"); rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}

	rec := get(t, fakeStore{err: errors.New("down")}, "/healthz")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status with a failing database = %d", rec.Code)
	}
}

func TestAssets(t *testing.T) {
	rec := get(t, fakeStore{}, "/assets/main-abc123.css")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("Cache-Control = %q", got)
	}

	rec = get(t, fakeStore{}, "/assets/missing.css")
	if rec.Code != http.StatusNotFound {
		t.Errorf("missing asset status = %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); strings.Contains(got, "immutable") {
		t.Errorf("missing asset Cache-Control = %q", got)
	}

	if rec := get(t, fakeStore{}, "/assets/"); rec.Code != http.StatusNotFound {
		t.Errorf("directory listing status = %d", rec.Code)
	}
	if rec := get(t, fakeStore{}, "/.vite/manifest.json"); rec.Code != http.StatusNotFound {
		t.Errorf("manifest status = %d", rec.Code)
	}
	if rec := get(t, fakeStore{}, "/favicon.png"); rec.Code != http.StatusOK {
		t.Errorf("favicon status = %d", rec.Code)
	}
}

func TestUnknownPath(t *testing.T) {
	if rec := get(t, fakeStore{}, "/does/not/exist"); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestPanicBecomes500(t *testing.T) {
	handler := recoverPanics(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.0.2.1:1234"
	if got := clientIP(r); got != "192.0.2.1" {
		t.Errorf("without header = %q", got)
	}

	r.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	if got := clientIP(r); got != "203.0.113.5" {
		t.Errorf("with header = %q", got)
	}
}
