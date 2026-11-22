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
	"github.com/matteocontrini/notedimatteo/internal/markdown"
)

type fakeStore struct {
	posts      []db.Post
	total      int
	post       *db.Post
	categories []db.CategoryCount
	archive    []db.MonthCount
	tags       []db.TagCount
	err        error

	// lastFilter and lastOffset record the last Posts call
	lastFilter db.PostFilter
	lastOffset int
	// saved records the HTML cached by SavePostHTML, by post ID
	saved   map[int]string
	saveErr error

	// sessions maps tokens to sessions; touched and deleted record changes
	sessions map[string]*db.Session
	touched  []int
	deleted  []string

	drafts        []db.Post
	createdDraft  int
	deletedDrafts []int

	// editPost is what PostForEdit returns
	editPost *db.EditablePost
	// savedEdit records the last SavePost call, which fails with savePostErr
	savedEdit   *db.PostEdit
	savePostErr error
}

func (f *fakeStore) Ping(context.Context) error {
	return f.err
}

func (f *fakeStore) Posts(_ context.Context, filter db.PostFilter, _, offset int) ([]db.Post, error) {
	f.lastFilter = filter
	f.lastOffset = offset
	return f.posts, f.err
}

func (f *fakeStore) CountPosts(context.Context, db.PostFilter) (int, error) {
	return f.total, f.err
}

func (f *fakeStore) PostByNumber(context.Context, int) (*db.Post, error) {
	return f.post, f.err
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

func (f *fakeStore) SessionByToken(_ context.Context, token string) (*db.Session, error) {
	return f.sessions[token], f.err
}

func (f *fakeStore) CreateSession(_ context.Context, token string) error {
	if f.sessions == nil {
		f.sessions = map[string]*db.Session{}
	}
	f.sessions[token] = &db.Session{ID: len(f.sessions) + 1, LastActivityAt: time.Now()}
	return f.err
}

func (f *fakeStore) TouchSession(_ context.Context, id int) error {
	f.touched = append(f.touched, id)
	return f.err
}

func (f *fakeStore) DeleteSession(_ context.Context, token string) error {
	f.deleted = append(f.deleted, token)
	delete(f.sessions, token)
	return f.err
}

func (f *fakeStore) Drafts(context.Context) ([]db.Post, error) {
	return f.drafts, f.err
}

func (f *fakeStore) CountDrafts(context.Context) (int, error) {
	return len(f.drafts), f.err
}

func (f *fakeStore) CreateDraft(context.Context) (int, error) {
	f.createdDraft = 671
	return f.createdDraft, f.err
}

func (f *fakeStore) DeleteDraft(_ context.Context, number int) (bool, error) {
	for _, d := range f.drafts {
		if d.Number == number {
			f.deletedDrafts = append(f.deletedDrafts, number)
			return true, f.err
		}
	}
	return false, f.err
}

func (f *fakeStore) PostForEdit(context.Context, int) (*db.EditablePost, error) {
	return f.editPost, f.err
}

func (f *fakeStore) SavePost(_ context.Context, number int, edit db.PostEdit) (*db.EditablePost, error) {
	f.savedEdit = &edit
	if f.savePostErr != nil || f.post == nil || f.post.Number != number {
		return nil, f.savePostErr
	}
	return &db.EditablePost{Number: edit.Number, Body: edit.Body, Tags: edit.Tags}, nil
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
			"assets/site.ts": {"file": "assets/site-def456.js"},
			"editor/main.ts": {"file": "assets/editor-789abc.js"}
		}`)},
		"assets/main-abc123.css": {Data: []byte("body{}")},
		"assets/site-def456.js":  {Data: []byte("")},
		"favicon.png":            {Data: []byte("png")},
	}
}

func testOptions(t *testing.T) Options {
	return Options{
		SiteURL:        "https://example.com/",
		Dist:           testDist(),
		Markdown:       testRenderer(t),
		StorageBaseURL: "https://storage.example.com",
		AdminUsername:  "admin",
		AdminPassword:  "secret",
	}
}

func newTestHandler(t *testing.T, store Store) http.Handler {
	t.Helper()

	handler, err := New(store, testOptions(t))
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
		_, err := New(&fakeStore{}, Options{SiteURL: siteURL, Dist: testDist(), Markdown: testRenderer(t), AdminUsername: "admin", AdminPassword: "secret"})
		if err == nil {
			t.Errorf("site URL %q was accepted", siteURL)
		}
	}
}

func TestMissingAsset(t *testing.T) {
	manifests := map[string]string{
		"without the script": `{"assets/main.css": {"file": "assets/main.css"}}`,
		// The editor page would miss this CSS: it loads only the editor's script
		"with editor CSS": `{
			"assets/main.css": {"file": "assets/main.css"},
			"assets/site.ts": {"file": "assets/site.js"},
			"editor/main.ts": {"file": "assets/main.js", "css": ["assets/main.css"]}
		}`,
	}
	for name, manifest := range manifests {
		dist := testDist()
		dist[".vite/manifest.json"] = &fstest.MapFile{Data: []byte(manifest)}
		if _, err := New(&fakeStore{}, Options{SiteURL: "https://example.com", Dist: dist, Markdown: testRenderer(t), AdminUsername: "admin", AdminPassword: "secret"}); err == nil {
			t.Errorf("a manifest %s was accepted", name)
		}
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

	// No directory listing: /assets/ redirects to /assets, which isn't a category
	if rec := get(t, &fakeStore{}, "/assets"); rec.Code != http.StatusNotFound {
		t.Errorf("/assets status = %d", rec.Code)
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
