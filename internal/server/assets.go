package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

const manifestPath = ".vite/manifest.json"

type manifestEntry struct {
	File string `json:"file"`
}

// assetResolver returns a function mapping a Vite source path, relative to the
// Vite root, to the URL of its hashed build output.
func assetResolver(dist fs.FS) (func(string) (string, error), error) {
	data, err := fs.ReadFile(dist, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read asset manifest: %w", err)
	}

	var manifest map[string]manifestEntry
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse asset manifest: %w", err)
	}

	return func(source string) (string, error) {
		entry, ok := manifest[source]
		if !ok {
			return "", fmt.Errorf("asset %q is not in the manifest", source)
		}
		return "/" + entry.File, nil
	}, nil
}

// hashedAssets serves files with hashed names, which can be cached forever.
func hashedAssets(dist fs.FS) http.Handler {
	files := http.FileServerFS(dist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// FileServer would otherwise list the directory
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		files.ServeHTTP(w, r)
	})
}

// favicon is served at a stable URL, so it's only cached for a day.
func favicon(dist fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFileFS(w, r, dist, "favicon.png")
	}
}
