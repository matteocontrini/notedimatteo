package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

func at(value string) *time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &t
}

func publishedPost() *db.Post {
	return &db.Post{ID: 7, Number: 670, Slug: "a-slug", Body: "Body", PublishedAt: at("2026-11-19T13:30:31Z")}
}

func TestStatusAndRedirects(t *testing.T) {
	tests := []struct {
		path     string
		store    *fakeStore
		status   int
		location string
		message  string
	}{
		{"/about", &fakeStore{}, http.StatusOK, "", ""},
		{"/cerca", &fakeStore{}, http.StatusOK, "", ""},
		{"/note", &fakeStore{}, http.StatusOK, "", ""},
		{"/video", &fakeStore{}, http.StatusOK, "", ""},
		{"/unknown-category", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		{"/a/b/c", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		{"/note/x", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		{"/archivio", &fakeStore{}, http.StatusNotFound, "", "Not Found"},

		// Trailing slashes
		{"/note/", &fakeStore{}, http.StatusPermanentRedirect, "/note", ""},
		{"/note/?page=2", &fakeStore{}, http.StatusPermanentRedirect, "/note?page=2", ""},
		{"//example.org/", &fakeStore{}, http.StatusPermanentRedirect, "/example.org", ""},
		{"/note//", &fakeStore{}, http.StatusPermanentRedirect, "/note", ""},

		// The router cleans other paths
		{"/archivio//ai", &fakeStore{}, http.StatusTemporaryRedirect, "/archivio/ai", ""},
		{"/cit/../note", &fakeStore{}, http.StatusTemporaryRedirect, "/note", ""},

		// Posts
		{"/670", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		{"/670", &fakeStore{post: &db.Post{Number: 670}}, http.StatusNotFound, "", "Not Found"},
		{"/670", &fakeStore{post: publishedPost()}, http.StatusTemporaryRedirect, "/670/a-slug", ""},
		{"/670/old-slug", &fakeStore{post: publishedPost()}, http.StatusTemporaryRedirect, "/670/a-slug", ""},
		{"/670/a-slug", &fakeStore{post: publishedPost()}, http.StatusOK, "", ""},
		{"/670/a-slug", &fakeStore{post: &db.Post{Number: 670, PublishedAt: at("2026-01-01T00:00:00Z")}}, http.StatusTemporaryRedirect, "/670", ""},
		{"/99999999999999999999", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		// Too big for the database's integer column
		{"/3000000000", &fakeStore{post: publishedPost()}, http.StatusNotFound, "", "Not Found"},

		// Tags
		{"/archivio/ai", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		{"/archivio/ai", &fakeStore{total: 3}, http.StatusOK, "", ""},
		{"/archivio/%20", &fakeStore{total: 3}, http.StatusNotFound, "", "Not Found"},

		// Months
		{"/archivio/2026/13", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		{"/archivio/2026/ab", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		{"/archivio/2026/04", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		{"/archivio/2026/04", &fakeStore{posts: []db.Post{*publishedPost()}}, http.StatusOK, "", ""},

		{"/archivio/2026/4?x=1", &fakeStore{}, http.StatusTemporaryRedirect, "/archivio/2026/04?x=1", ""},
		{"/archivio/02026/04", &fakeStore{}, http.StatusTemporaryRedirect, "/archivio/2026/04", ""},
		{"/archivio/2026/4.0", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
		{"/archivio/0/01", &fakeStore{}, http.StatusNotFound, "", "Not Found"},

		// Invalid pages
		{"/note?page=abc", &fakeStore{total: 3}, http.StatusBadRequest, "", "Bad Request"},
		{"/?page=0", &fakeStore{total: 3}, http.StatusBadRequest, "", "Bad Request"},
		{"/?page=2abc", &fakeStore{total: 3}, http.StatusBadRequest, "", "Bad Request"},
		{"/archivio/ai?page=abc", &fakeStore{}, http.StatusNotFound, "", "Not Found"},
	}

	for _, tt := range tests {
		rec := get(t, tt.store, tt.path)

		if rec.Code != tt.status {
			t.Errorf("%s: status = %d, want %d", tt.path, rec.Code, tt.status)
		}
		if got := rec.Header().Get("Location"); got != tt.location {
			t.Errorf("%s: Location = %q, want %q", tt.path, got, tt.location)
		}
		if tt.message != "" && !strings.Contains(rec.Body.String(), "<p>"+tt.message+"</p>") {
			t.Errorf("%s: error page doesn't show %q", tt.path, tt.message)
		}
	}
}

func TestErrorPageHasLayout(t *testing.T) {
	rec := get(t, &fakeStore{categories: []db.CategoryCount{{Slug: "note", Count: 1}}}, "/nope")

	assertContains(t, rec, "<h1>404</h1>", `href="/note"`, `rel="canonical" href="https://example.com/nope"`)
}

func TestFilters(t *testing.T) {
	tests := []struct {
		path string
		want db.PostFilter
	}{
		{"/note", db.PostFilter{Category: "note"}},
		{"/archivio/open%20source", db.PostFilter{Tag: "open source"}},
		{"/cerca?q=+%25go+", db.PostFilter{Search: "%go"}},
		{"/archivio/2026/02", db.PostFilter{
			From:  time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			Until: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		}},
		{"/archivio/2026/12", db.PostFilter{
			From:  time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
			Until: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		}},
	}

	for _, tt := range tests {
		store := &fakeStore{total: 1, posts: []db.Post{*publishedPost()}}
		get(t, store, tt.path)
		if store.lastFilter != tt.want {
			t.Errorf("%s: filter = %+v, want %+v", tt.path, store.lastFilter, tt.want)
		}
	}
}

func TestSearchWithoutQuery(t *testing.T) {
	store := &fakeStore{}
	rec := get(t, store, "/cerca?q=+")

	assertContains(t, rec, `Nessun risultato per ""`, "<title>Cerca in Note di Matteo</title>")
	if store.lastFilter != (db.PostFilter{}) {
		t.Error("an empty query searched posts")
	}
}

func TestPostMeta(t *testing.T) {
	post := db.Post{
		Number:      670,
		Body:        strings.Repeat("a\n\n b ", 40),
		PublishedAt: at("2026-11-19T13:30:31.349Z"),
	}

	meta, err := postMeta(post)
	if err != nil {
		t.Fatal(err)
	}

	if meta.Title != "Nota #670 del 19/11/2026" {
		t.Errorf("title = %q", meta.Title)
	}
	if !strings.HasSuffix(meta.Description, "…") || strings.Contains(meta.Description, "  ") {
		t.Errorf("description = %q", meta.Description)
	}
	want := `{"@context":"https://schema.org","@type":"NewsArticle","headline":"Nota #670 del 19/11/2026","datePublished":"2026-11-19T13:30:31.349Z"}`
	if meta.JSONLD != want {
		t.Errorf("JSON-LD = %s", meta.JSONLD)
	}

	post.Title = "</script> & co"
	post.UpdatedAt = at("2026-11-20T00:00:00Z")
	meta, _ = postMeta(post)
	if strings.Contains(meta.JSONLD, "</script>") || !strings.Contains(meta.JSONLD, `"dateModified":"2026-11-20T00:00:00.000Z"`) {
		t.Errorf("JSON-LD = %s", meta.JSONLD)
	}
}

func TestFeed(t *testing.T) {
	html := "<p>Contains ]]> marker</p>"
	store := &fakeStore{posts: []db.Post{{
		ID:          1,
		Number:      670,
		Title:       "Title & more",
		Slug:        "a-slug",
		Category:    "cit",
		Tags:        []string{"ai"},
		Body:        strings.Repeat("x", 250),
		PublishedAt: at("2026-11-19T13:30:31Z"),
		CachedHTML:  &html,
	}}}

	rec := get(t, store, "/feed.xml")

	if got := rec.Header().Get("Content-Type"); got != "application/xml; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}

	var feed struct {
		Items []struct {
			Title       string   `xml:"title"`
			Link        string   `xml:"link"`
			PubDate     string   `xml:"pubDate"`
			Categories  []string `xml:"category"`
			Description string   `xml:"description"`
			Content     string   `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
		} `xml:"channel>item"`
	}
	if err := xml.Unmarshal(rec.Body.Bytes(), &feed); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}

	item := feed.Items[0]
	if item.Title != "#670 / Title & more" || item.Link != "https://example.com/670/a-slug" {
		t.Errorf("title and link = %q, %q", item.Title, item.Link)
	}
	if item.PubDate != "Thu, 19 Nov 2026 13:30:31 GMT" {
		t.Errorf("pubDate = %q", item.PubDate)
	}
	if strings.Join(item.Categories, ",") != "cit,ai" {
		t.Errorf("categories = %v", item.Categories)
	}
	if item.Description != strings.Repeat("x", 200)+"…" {
		t.Errorf("description = %q", item.Description)
	}
	if strings.TrimSpace(item.Content) != html {
		t.Errorf("content = %q", item.Content)
	}
}
