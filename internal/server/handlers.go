package server

import (
	"fmt"
	"net/http"

	"github.com/matteocontrini/notedimatteo/internal/views"
)

func (s *Server) about(w http.ResponseWriter, r *http.Request) error {
	page, err := s.newPage(r, views.Meta{})
	if err != nil {
		return err
	}
	return s.render(w, r, http.StatusOK, views.About(page))
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) error {
	if err := s.store.Ping(r.Context()); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte("ok\n"))
	return err
}
