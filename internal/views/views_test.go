package views

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/a-h/templ"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

func render(t *testing.T, component templ.Component) string {
	t.Helper()

	var b strings.Builder
	if err := component.Render(context.Background(), &b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func at(value string) *time.Time {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestGroupByDay(t *testing.T) {
	posts := []PostItem{
		// 00:30 on April 17 in Rome
		{Post: db.Post{Number: 3, PublishedAt: at("2026-04-16T22:30:00Z")}},
		{Post: db.Post{Number: 2, PublishedAt: at("2026-04-16T10:00:00Z")}},
		{Post: db.Post{Number: 1, PublishedAt: at("2026-04-16T08:00:00Z")}},
	}

	groups := GroupByDay(posts)

	if len(groups) != 2 {
		t.Fatalf("got %d groups", len(groups))
	}
	if groups[0].Heading != "17 aprile 2026" || len(groups[0].Posts) != 1 {
		t.Errorf("first group = %s with %d posts", groups[0].Heading, len(groups[0].Posts))
	}
	if groups[1].Heading != "16 aprile 2026" || len(groups[1].Posts) != 2 {
		t.Errorf("second group = %s with %d posts", groups[1].Heading, len(groups[1].Posts))
	}
}

func TestPost(t *testing.T) {
	item := PostItem{
		Post: db.Post{Number: 42, Title: "Title", PublishedAt: at("2026-04-16T12:00:00Z")},
		HTML: "<p>Rendered <del>body</del></p>",
	}

	html := render(t, post(item, false, false, false))
	if !strings.Contains(html, item.HTML) {
		t.Error("body HTML is escaped or missing")
	}
	if !strings.Contains(html, "Title") {
		t.Error("title is missing")
	}
	if strings.Contains(html, "16 aprile 2026") {
		t.Error("date is shown without showDate")
	}
	if strings.Contains(html, "modifica") {
		t.Error("edit link is shown without canEdit")
	}

	html = render(t, post(item, true, true, false))
	if !strings.Contains(html, "16 aprile 2026") {
		t.Error("date is missing with showDate")
	}
	if !strings.Contains(html, `href="/42/edit"`) {
		t.Error("edit link is missing with canEdit")
	}

	item.HideTitle = true
	if strings.Contains(render(t, post(item, false, false, false)), "Title") {
		t.Error("hidden title is shown")
	}
}

func TestPagination(t *testing.T) {
	if html := render(t, pagination(Pagination{Current: 1, Total: 1})); strings.TrimSpace(html) != "" {
		t.Errorf("single page renders %q", html)
	}

	html := render(t, pagination(Pagination{Current: 1, Total: 2, Next: "/?page=2"}))
	if strings.Contains(html, "Precedente") || !strings.Contains(html, "Successiva") {
		t.Errorf("first page renders %q", html)
	}
}

func TestHomeWithoutPosts(t *testing.T) {
	if html := render(t, Home(Page{}, nil, Pagination{})); !strings.Contains(html, "Non c'è niente qui.") {
		t.Error("empty message is missing")
	}
}
