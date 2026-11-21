package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"time"

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
	if isLoggedIn(r) {
		// Shared caches must not keep pages with admin controls
		w.Header().Set("Cache-Control", "private, no-cache")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	return writeBody(w, r, status, buf.Bytes())
}

// writeBody writes a complete response body with its length: Railway's edge
// compresses only responses that declare it. A successful GET also gets an
// ETag, so a browser that already has the same body gets a 304 instead.
func writeBody(w http.ResponseWriter, r *http.Request, status int, body []byte) error {
	if status == http.StatusOK && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		hash := fnv.New64a()
		hash.Write(body)
		w.Header().Set("ETag", fmt.Sprintf(`"%016x"`, hash.Sum64()))
		// ServeContent sets the length and answers If-None-Match
		http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(body))
		return nil
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, err := w.Write(body)
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
