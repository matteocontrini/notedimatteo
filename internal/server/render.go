package server

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

// render renders a component into a buffer first, so that an error results in
// a clean error response rather than a partially written page.
func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) error {
	var buf bytes.Buffer
	if err := component.Render(r.Context(), &buf); err != nil {
		return fmt.Errorf("render: %w", err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}
