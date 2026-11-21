package server

import (
	"bytes"
	"encoding/json"
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

// writeJSON writes v as JSON with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(data)
	return err
}
