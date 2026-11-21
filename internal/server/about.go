package server

import (
	"net/http"

	"github.com/matteocontrini/notedimatteo/internal/views"
)

// about serves the about page.
func (s *Server) about(w http.ResponseWriter, r *http.Request) error {
	page, err := s.newPage(r, views.Meta{})
	if err != nil {
		return err
	}
	return s.render(w, r, http.StatusOK, views.About(page))
}
