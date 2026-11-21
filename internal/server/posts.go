package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/urls"
	"github.com/matteocontrini/notedimatteo/internal/views"
)

const descriptionLength = 160

// postItems pairs posts with their HTML, rendering and caching the bodies whose
// cached HTML is missing or out of date.
func (s *Server) postItems(ctx context.Context, posts []db.Post) ([]views.PostItem, error) {
	items := make([]views.PostItem, 0, len(posts))
	for _, post := range posts {
		if post.CachedHTML != nil {
			items = append(items, views.PostItem{Post: post, HTML: *post.CachedHTML})
			continue
		}

		html, err := s.markdown.Render(post.Body)
		if err != nil {
			return nil, fmt.Errorf("render post %d: %w", post.Number, err)
		}
		// A failed cache write only costs rendering again, so the page is still shown
		if err := s.store.SavePostHTML(ctx, post.ID, post.BodyRevision, html); err != nil {
			slog.Error("caching post HTML failed", "post", post.Number, "error", err)
		}
		items = append(items, views.PostItem{Post: post, HTML: html})
	}
	return items, nil
}

// postOrCategory serves /{segment}: a post number, like /670, or a category.
func (s *Server) postOrCategory(w http.ResponseWriter, r *http.Request) error {
	segment := r.PathValue("segment")
	if number, ok := postNumber(segment); ok {
		return s.post(w, r, number, "")
	}
	return s.category(w, r, segment)
}

// postWithSlug serves /{number}/{slug}, and the editor at /{number}/edit.
// The router would reject a separate GET /{number}/edit pattern. That pattern
// and GET /archivio/{tag} both match /archivio/edit, and neither is more
// specific than the other.
func (s *Server) postWithSlug(w http.ResponseWriter, r *http.Request) error {
	if r.PathValue("slug") == "edit" {
		return requireLogin(s.editPage)(w, r)
	}
	number, ok := postNumber(r.PathValue("number"))
	if !ok {
		return errNotFound
	}
	return s.post(w, r, number, r.PathValue("slug"))
}

// postNumber parses a post number written with digits only, like 670. It
// refuses numbers too big for the database's integer column.
func postNumber(segment string) (int, bool) {
	if segment == "" || strings.Trim(segment, "0123456789") != "" {
		return 0, false
	}
	number, err := strconv.ParseInt(segment, 10, 32)
	return int(number), err == nil
}

func (s *Server) post(w http.ResponseWriter, r *http.Request, number int, slug string) error {
	post, err := s.store.PostByNumber(r.Context(), number)
	if err != nil {
		return err
	}
	if post == nil || post.PublishedAt == nil {
		return errNotFound
	}

	// Redirect to the post's current URL
	if slug != post.Slug {
		redirect(w, urls.Post(post.Number, post.Slug), http.StatusTemporaryRedirect)
		return nil
	}

	items, err := s.postItems(r.Context(), []db.Post{*post})
	if err != nil {
		return err
	}
	meta, err := postMeta(*post)
	if err != nil {
		return err
	}
	page, err := s.newPage(r, meta)
	if err != nil {
		return err
	}

	return s.render(w, r, http.StatusOK, views.PostPage(page, items[0]))
}

// postMeta returns a post page's metadata: its title, the start of its body as
// description, and NewsArticle structured data.
func postMeta(post db.Post) (views.Meta, error) {
	title := views.PostTitle(post)

	description := strings.Join(strings.Fields(truncate(post.Body, descriptionLength)), " ")
	if len([]rune(post.Body)) > descriptionLength {
		description += "…"
	}

	article := struct {
		Context       string `json:"@context"`
		Type          string `json:"@type"`
		Headline      string `json:"headline"`
		DatePublished string `json:"datePublished"`
		DateModified  string `json:"dateModified,omitempty"`
	}{
		Context:       "https://schema.org",
		Type:          "NewsArticle",
		Headline:      title,
		DatePublished: isoTime(*post.PublishedAt),
	}
	if post.UpdatedAt != nil {
		article.DateModified = isoTime(*post.UpdatedAt)
	}
	// json.Marshal escapes <, >, and &, so the JSON is safe inside a script element
	jsonLD, err := json.Marshal(article)
	if err != nil {
		return views.Meta{}, err
	}

	return views.Meta{Title: title, Description: description, OGType: "article", JSONLD: string(jsonLD)}, nil
}

// truncate returns the first n characters of s.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// isoTime formats a time in UTC with milliseconds, like 2026-11-19T13:30:31.349Z.
func isoTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}
