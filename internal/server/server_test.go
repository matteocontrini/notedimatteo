package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/markdown"
)

type fakeStore struct {
	posts      []db.Post
	total      int
	categories []db.CategoryCount
	archive    []db.MonthCount
	tags       []db.TagCount
	err        error

	// lastOffset records the offset of the last PublishedPosts call
	lastOffset int
	// saved records the HTML cached by SavePostHTML, by post ID
	saved   map[int]string
	saveErr error
}

func (f *fakeStore) Ping(context.Context) error {
	return f.err
}

func (f *fakeStore) PublishedPosts(_ context.Context, _, offset int) ([]db.Post, error) {
	f.lastOffset = offset
	return f.posts, f.err
}

func (f *fakeStore) CountPublishedPosts(context.Context) (int, error) {
	return f.total, f.err
}

func (f *fakeStore) Categories(context.Context) ([]db.CategoryCount, error) {
	return f.categories, f.err
}

func (f *fakeStore) Archive(context.Context) ([]db.MonthCount, error) {
	return f.archive, f.err
}

func (f *fakeStore) Tags(context.Context) ([]db.TagCount, error) {
	return f.tags, f.err
}

func (f *fakeStore) SavePostHTML(_ context.Context, postID, _ int, html string) error {
	if f.saved == nil {
		f.saved = map[int]string{}
	}
	f.saved[postID] = html
	return f.saveErr
}

func testRenderer(t *testing.T) *markdown.Renderer {
	t.Helper()

	r, err := markdown.New(markdown.Options{StorageBaseURL: "https://storage.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// testDist stands in for the Vite output, so tests don't depend on its hashes.
func testDist() fstest.MapFS {
	return fstest.MapFS{
		".vite/manifest.json": {Data: []byte(`{
			"assets/main.css": {"file": "assets/main-abc123.css"},
			"assets/site.ts": {"file": "assets/site-def456.js"}
		}`)},
		"assets/main-abc123.css": {Data: []byte("body{}")},
		"assets/site-def456.js":  {Data: []byte("")},
		"favicon.png":            {Data: []byte("png")},
	}
}

func newTestHandler(t *testing.T, store Store) http.Handler {
	t.Helper()

	handler, err := New(store, Options{SiteURL: "https://example.com/", Dist: testDist(), Markdown: testRenderer(t)})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func get(t *testing.T, store Store, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	newTestHandler(t, store).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

// assertContains fails for each string missing from the response body.
func assertContains(t *testing.T, rec *httptest.ResponseRecorder, wants ...string) {
	t.Helper()

	body := rec.Body.String()
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("response is missing %q", want)
		}
	}
}

func TestInvalidSiteURL(t *testing.T) {
	for _, siteURL := range []string{"", "notedimatteo.it", "/path"} {
		_, err := New(&fakeStore{}, Options{SiteURL: siteURL, Dist: testDist(), Markdown: testRenderer(t)})
		if err == nil {
			t.Errorf("site URL %q was accepted", siteURL)
		}
	}
}

func TestMissingAsset(t *testing.T) {
	dist := testDist()
	dist[".vite/manifest.json"] = &fstest.MapFile{Data: []byte(`{"assets/main.css": {"file": "assets/main.css"}}`)}

	if _, err := New(&fakeStore{}, Options{SiteURL: "https://example.com", Dist: dist, Markdown: testRenderer(t)}); err == nil {
		t.Error("a manifest without the script was accepted")
	}
}

func TestHealthz(t *testing.T) {
	if rec := get(t, &fakeStore{}, "/healthz"); rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}

	rec := get(t, &fakeStore{err: errors.New("down")}, "/healthz")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status with a failing database = %d", rec.Code)
	}
}

func TestAssets(t *testing.T) {
	rec := get(t, &fakeStore{}, "/assets/main-abc123.css")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("Cache-Control = %q", got)
	}

	rec = get(t, &fakeStore{}, "/assets/missing.css")
	if rec.Code != http.StatusNotFound {
		t.Errorf("missing asset status = %d", rec.Code)
	}
	if got := rec.Header().Get("Cache-Control"); strings.Contains(got, "immutable") {
		t.Errorf("missing asset Cache-Control = %q", got)
	}

	if rec := get(t, &fakeStore{}, "/assets/"); rec.Code != http.StatusNotFound {
		t.Errorf("directory listing status = %d", rec.Code)
	}
	if rec := get(t, &fakeStore{}, "/.vite/manifest.json"); rec.Code != http.StatusNotFound {
		t.Errorf("manifest status = %d", rec.Code)
	}
	if rec := get(t, &fakeStore{}, "/favicon.png"); rec.Code != http.StatusOK {
		t.Errorf("favicon status = %d", rec.Code)
	}
}

func TestUnknownPath(t *testing.T) {
	if rec := get(t, &fakeStore{}, "/does/not/exist"); rec.Code != http.StatusNotFound {
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
