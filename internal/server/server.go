// Package server contains the HTTP routes, handlers, and middleware.
package server

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

// Store is the database access used by the handlers.
type Store interface {
	Ping(ctx context.Context) error
	LatestPosts(ctx context.Context, limit int) ([]db.Post, error)
}

type Server struct {
	store Store
	pages map[string]*template.Template
}

// New returns the HTTP handler for the whole site.
func New(store Store, templates, dist fs.FS) (http.Handler, error) {
	asset, err := assetResolver(dist)
	if err != nil {
		return nil, err
	}

	pages, err := parseTemplates(templates, asset)
	if err != nil {
		return nil, err
	}

	s := &Server{store: store, pages: pages}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handle(s.home))
	mux.HandleFunc("GET /healthz", s.handle(s.healthz))
	mux.Handle("GET /assets/", hashedAssets(dist))
	mux.HandleFunc("GET /favicon.png", favicon(dist))

	return logRequests(recoverPanics(mux)), nil
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// handle turns a returned error into a logged 500 response.
func (s *Server) handle(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if v == http.ErrAbortHandler {
				panic(v)
			}
			slog.Error("request panicked",
				"method", r.Method,
				"path", r.URL.Path,
				"error", fmt.Sprint(v),
				"stack", string(debug.Stack()),
			)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}()

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("request",
			"ip", clientIP(r),
			"status", rec.status,
			"method", r.Method,
			"path", r.URL.RequestURI(),
		)
	})
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip, _, _ := strings.Cut(forwarded, ",")
		return strings.TrimSpace(ip)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
