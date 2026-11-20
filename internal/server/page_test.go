package server

import (
	"context"
	"errors"
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
	valid := map[string]int{"": 1, "page=": 1, "page=1": 1, "page=2": 2}
	for query, want := range valid {
		r := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
		if got, err := pageNumber(r); got != want || err != nil {
			t.Errorf("pageNumber(%q) = %d, %v, want %d", query, got, err, want)
		}
	}

	for _, query := range []string{"page=0", "page=-1", "page=abc", "page=2abc", "page=1.5", "page=99999999999"} {
		r := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
		if _, err := pageNumber(r); err != errBadRequest {
			t.Errorf("pageNumber(%q) error = %v", query, err)
		}
	}
}

func TestSidebarOnlyWhenNeeded(t *testing.T) {
	s := &Server{store: &fakeStore{archive: []db.MonthCount{{Year: 2026, Month: 4, Count: 1}}, tags: []db.TagCount{{Name: "ai", Count: 1}}}}
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	page, err := s.newPage(r, views.Meta{})
	if err != nil || page.Archive != nil || page.Tags != nil {
		t.Errorf("newPage loaded the sidebar: %+v, %v", page, err)
	}
	page, err = s.newSidebarPage(r, views.Meta{})
	if err != nil || len(page.Archive) != 1 || len(page.Tags) != 1 {
		t.Errorf("newSidebarPage didn't load the sidebar: %+v, %v", page, err)
	}
}

func TestPostItems(t *testing.T) {
	cached := "<p>Cached</p>"
	store := &fakeStore{}
	s := &Server{store: store, markdown: testRenderer(t)}

	items, err := s.postItems(context.Background(), []db.Post{
		{ID: 1, CachedHTML: &cached, Body: "Not rendered"},
		{ID: 2, Body: "~~Rendered~~"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if items[0].HTML != cached {
		t.Errorf("cached post HTML = %q", items[0].HTML)
	}
	want := "<p><del>Rendered</del></p>\n"
	if items[1].HTML != want {
		t.Errorf("rendered post HTML = %q, want %q", items[1].HTML, want)
	}
	if len(store.saved) != 1 || store.saved[2] != want {
		t.Errorf("saved = %v, want only post 2", store.saved)
	}
}

func TestPostItemsWhenCachingFails(t *testing.T) {
	s := &Server{store: &fakeStore{saveErr: errors.New("read-only")}, markdown: testRenderer(t)}

	items, err := s.postItems(context.Background(), []db.Post{{ID: 1, Body: "Body"}})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].HTML != "<p>Body</p>\n" {
		t.Errorf("HTML = %q", items[0].HTML)
	}
}
