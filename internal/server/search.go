package server

import (
	"net/http"
	"strings"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/views"
)

const searchResults = 20

func (s *Server) search(w http.ResponseWriter, r *http.Request) error {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	var items []views.PostItem
	meta := views.Meta{Title: "Cerca in Note di Matteo"}
	if query != "" {
		posts, err := s.store.Posts(r.Context(), db.PostFilter{Search: query}, searchResults, 0)
		if err != nil {
			return err
		}
		if items, err = s.postItems(r.Context(), posts); err != nil {
			return err
		}
		meta.Title = `Risultati per "` + query + `"`
	}

	page, err := s.newSidebarPage(r, meta)
	if err != nil {
		return err
	}
	return s.render(w, r, http.StatusOK, views.Search(page, query, items))
}
