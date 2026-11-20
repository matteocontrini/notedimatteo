package server

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/views"
)

func TestNewPage(t *testing.T) {
	s := &Server{
		store: &fakeStore{
			categories: []db.CategoryCount{{Slug: "note", Count: 13}, {Slug: "cit", Count: 3}},
		},
		siteURL: "https://example.com",
	}

	page, err := s.newPage(httptest.NewRequest(http.MethodGet, "/cit?q=go&page=2", nil), views.Meta{Title: "Cit."})
	if err != nil {
		t.Fatal(err)
	}

	wantMeta := views.Meta{Title: "Cit.", Description: siteTitle, OGTitle: "Cit.", OGType: "website"}
	if page.Meta != wantMeta {
		t.Errorf("meta = %+v", page.Meta)
	}
	if page.Canonical != "https://example.com/cit" {
		t.Errorf("canonical = %q", page.Canonical)
	}
	if page.SearchQuery != "go" {
		t.Errorf("search query = %q", page.SearchQuery)
	}
	wantCategories := []views.CategoryLink{
		{Label: "Note", Count: 13, URL: "/note"},
		{Label: "Cit.", Count: 3, URL: "/cit", Current: true},
	}
	if !reflect.DeepEqual(page.Categories, wantCategories) {
		t.Errorf("categories = %+v", page.Categories)
	}
}

func TestNewPageDefaultTitle(t *testing.T) {
	s := &Server{store: &fakeStore{}}

	page, err := s.newPage(httptest.NewRequest(http.MethodGet, "/", nil), views.Meta{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Meta.Title != siteTitle || page.Meta.OGTitle != siteTitle {
		t.Errorf("meta = %+v", page.Meta)
	}
}

func TestArchiveYears(t *testing.T) {
	months := []db.MonthCount{
		{Year: 2026, Month: 4, Count: 1},
		{Year: 2026, Month: 2, Count: 1},
		{Year: 2025, Month: 12, Count: 5},
	}

	want := []views.ArchiveYear{
		{Year: 2026, Months: months[:2]},
		{Year: 2025, Months: months[2:]},
	}
	if got := archiveYears(months); !reflect.DeepEqual(got, want) {
		t.Errorf("archiveYears = %+v", got)
	}
}

func TestNewPagination(t *testing.T) {
	tests := []struct {
		current, total int
		want           views.Pagination
	}{
		{1, 1, views.Pagination{Current: 1, Total: 1}},
		{1, 3, views.Pagination{Current: 1, Total: 3, Next: "/note?page=2"}},
		{2, 3, views.Pagination{Current: 2, Total: 3, Previous: "/note", Next: "/note?page=3"}},
		{3, 3, views.Pagination{Current: 3, Total: 3, Previous: "/note?page=2"}},
		// Past the last page, like the SvelteKit app
		{5, 3, views.Pagination{Current: 5, Total: 3, Previous: "/note?page=4"}},
	}

	for _, tt := range tests {
		if got := newPagination("/note", tt.current, tt.total); got != tt.want {
			t.Errorf("newPagination(%d, %d) = %+v", tt.current, tt.total, got)
		}
	}
}

func TestPageNumber(t *testing.T) {
	tests := map[string]int{"": 1, "page=2": 2, "page=0": 1, "page=-1": 1, "page=abc": 1}

	for query, want := range tests {
		r := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
		if got := pageNumber(r); got != want {
			t.Errorf("pageNumber(%q) = %d, want %d", query, got, want)
		}
	}
}

func TestNewPostItem(t *testing.T) {
	cached := "<p>Cached</p>"
	if got := newPostItem(db.Post{CachedHTML: &cached}); got.HTML != cached {
		t.Errorf("cached HTML = %q", got.HTML)
	}

	got := newPostItem(db.Post{Body: "Uncached <b>body</b>"})
	if want := "<pre>Uncached &lt;b&gt;body&lt;/b&gt;</pre>"; got.HTML != want {
		t.Errorf("fallback HTML = %q, want %q", got.HTML, want)
	}
}
