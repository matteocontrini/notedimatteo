package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/views"
)

const postsPerPage = 20

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

// pageNumber reads the page query parameter. Invalid values mean the first page.
func pageNumber(r *http.Request) int {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) error {
	page, err := s.newPage(r, views.Meta{})
	if err != nil {
		return err
	}

	current := pageNumber(r)
	total, err := s.store.CountPublishedPosts(r.Context())
	if err != nil {
		return err
	}
	posts, err := s.store.PublishedPosts(r.Context(), postsPerPage, (current-1)*postsPerPage)
	if err != nil {
		return err
	}

	items, err := s.postItems(r.Context(), posts)
	if err != nil {
		return err
	}

	totalPages := (total + postsPerPage - 1) / postsPerPage
	component := views.Home(page, views.GroupByDay(items), newPagination(r.URL.Path, current, totalPages))

	return s.render(w, r, http.StatusOK, component)
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) error {
	if err := s.store.Ping(r.Context()); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte("ok\n"))
	return err
}
