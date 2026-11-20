package server

import (
	"fmt"
	"html"
	"net/http"
	"strconv"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/views"
)

const postsPerPage = 20

func newPostItem(post db.Post) views.PostItem {
	if post.CachedHTML != nil {
		return views.PostItem{Post: post, HTML: *post.CachedHTML}
	}
	// Temporary until the Markdown renderer is ported
	return views.PostItem{Post: post, HTML: "<pre>" + html.EscapeString(post.Body) + "</pre>"}
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

	items := make([]views.PostItem, 0, len(posts))
	for _, post := range posts {
		items = append(items, newPostItem(post))
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
