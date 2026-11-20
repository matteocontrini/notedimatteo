package server

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// parseTemplates builds one template set per page, each with the shared layout
// and partials. Parsing at startup makes syntax errors fail immediately.
func parseTemplates(templates fs.FS, funcs template.FuncMap) (map[string]*template.Template, error) {
	base, err := template.New("").
		Funcs(funcs).
		ParseFS(templates, "layout.gohtml", "partials/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("parse shared templates: %w", err)
	}

	files, err := fs.Glob(templates, "pages/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("list page templates: %w", err)
	}

	pages := make(map[string]*template.Template, len(files))
	for _, file := range files {
		page, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone shared templates: %w", err)
		}
		if _, err := page.ParseFS(templates, file); err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		pages[strings.TrimSuffix(path.Base(file), ".gohtml")] = page
	}

	return pages, nil
}

// render executes a page into a buffer first, so that a template error results
// in a clean error response rather than a partially written page.
func (s *Server) render(w http.ResponseWriter, status int, name string, data any) error {
	page, ok := s.pages[name]
	if !ok {
		return fmt.Errorf("unknown page template %q", name)
	}

	var buf bytes.Buffer
	if err := page.ExecuteTemplate(&buf, "layout", data); err != nil {
		return fmt.Errorf("render %s: %w", name, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}
