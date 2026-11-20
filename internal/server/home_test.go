package server

import (
	"errors"
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

func TestHomeRendersPosts(t *testing.T) {
	cached := "<p>Cached <del>body</del></p>"
	store := &fakeStore{
		total: 2,
		posts: []db.Post{
			{
				Number:   42,
				Category: "note",
				Title:    "A <title>",
				Slug:     "a-slug",
				Body:     "Markdown",
				Tags:     []string{"go", "c#"},
				// 00:30 on April 17 in Rome
				PublishedAt: at("2026-04-16T22:30:00Z"),
				CachedHTML:  &cached,
			},
			{
				Number:      41,
				Title:       "Hidden title",
				HideTitle:   true,
				Body:        "Uncached <b>body</b>",
				PublishedAt: at("2026-04-16T10:00:00Z"),
			},
		},
		categories: []db.CategoryCount{{Slug: "note", Count: 13}, {Slug: "cit", Count: 3}},
		archive: []db.MonthCount{
			{Year: 2026, Month: 4, Count: 1},
			{Year: 2026, Month: 2, Count: 1},
			{Year: 2025, Month: 12, Count: 5},
		},
		tags: []db.TagCount{{Name: "go", Count: 3}, {Name: "c#", Count: 1}},
	}

	rec := get(t, store, "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	assertContains(t, rec,
		// Head
		`<title>Note di Matteo</title>`,
		`<meta name="description" content="Note di Matteo" />`,
		`<meta property="og:type" content="website" />`,
		`<link rel="canonical" href="https://example.com/" />`,
		`href="/assets/main-abc123.css"`,
		`src="/assets/site-def456.js"`,
		`mode-watcher-mode`,
		`window.location.host !== "example.com"`,
		// Header and top bar
		`href="https://matteosonoio.it"`,
		`href="/note"`,
		`Cit.`,
		// Posts, grouped by day in Rome's time zone
		`17 aprile 2026`,
		`16 aprile 2026`,
		`00:30`,
		`12:00`,
		`A &lt;title&gt;`,
		cached,
		`href="/42/a-slug"`,
		`href="/41"`,
		`href="/archivio/c%23"`,
		`Uncached &lt;b&gt;body&lt;/b&gt;`,
		// Sidebar
		`<li class="mt-2 font-semibold">2025</li>`,
		`href="/archivio/2026/04">aprile</a>`,
		`href="/archivio/2025/12">dicembre</a>`,
		`<li data-name="c#">`,
		// Footer
		`data-theme-select`,
	)
	body := rec.Body.String()
	if strings.Contains(body, "Hidden title") {
		t.Error("response contains a hidden title")
	}
	if strings.Contains(body, "Pagina") {
		t.Error("response shows pagination for a single page")
	}
	if strings.Contains(body, "modifica") {
		t.Error("response shows edit links to a visitor")
	}
	if strings.Count(body, "<li class=\"mt-2 font-semibold\">2026</li>") != 1 {
		t.Error("response doesn't show 2026 exactly once in the archive")
	}
}

func TestHomePagination(t *testing.T) {
	store := &fakeStore{total: 45}

	rec := get(t, store, "/?page=2")
	assertContains(t, rec,
		`Pagina 2 di 3`,
		`href="/">← Precedente`,
		`href="/?page=3">Successiva →`,
	)
	if store.lastOffset != 20 {
		t.Errorf("offset = %d", store.lastOffset)
	}

	rec = get(t, store, "/?page=3")
	assertContains(t, rec, `href="/?page=2">← Precedente`)
	if strings.Contains(rec.Body.String(), "Successiva") {
		t.Error("the last page links to a next page")
	}

	for _, page := range []string{"0", "-1", "abc", ""} {
		get(t, store, "/?page="+page)
		if store.lastOffset != 0 {
			t.Errorf("page %q: offset = %d", page, store.lastOffset)
		}
	}
}

func TestHomeWithoutPosts(t *testing.T) {
	rec := get(t, &fakeStore{}, "/")

	assertContains(t, rec, "Non c'è niente qui.")
}

func TestHomeKeepsSearchQuery(t *testing.T) {
	rec := get(t, &fakeStore{}, `/?q=a"b`)

	assertContains(t, rec, `value="a&#34;b"`)
}

func TestHomeDatabaseError(t *testing.T) {
	rec := get(t, &fakeStore{err: errors.New("secret connection details")}, "/")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Error("response leaks the error")
	}
}
