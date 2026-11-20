// Package server contains the HTTP routes, handlers, and middleware.
package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/markdown"
)

// Store is the database access used by the handlers.
type Store interface {
	Ping(ctx context.Context) error
	Posts(ctx context.Context, filter db.PostFilter, limit, offset int) ([]db.Post, error)
	CountPosts(ctx context.Context, filter db.PostFilter) (int, error)
	PostByNumber(ctx context.Context, number int) (*db.Post, error)
	Categories(ctx context.Context) ([]db.CategoryCount, error)
	Archive(ctx context.Context) ([]db.MonthCount, error)
	Tags(ctx context.Context) ([]db.TagCount, error)
	SavePostHTML(ctx context.Context, postID, bodyRevision int, html string) error
}

type Options struct {
	// SiteURL is the public address, like https://notedimatteo.it.
	SiteURL string
	// Dist holds the assets built by Vite, with their manifest.
	Dist fs.FS
	// Markdown renders post bodies that have no up-to-date cached HTML.
	Markdown *markdown.Renderer
}

type Server struct {
	store      Store
	markdown   *markdown.Renderer
	siteURL    string
	siteHost   string
	stylesheet string
	script     string
}

// New returns the HTTP handler for the whole site.
func New(store Store, opts Options) (http.Handler, error) {
	site, err := url.Parse(opts.SiteURL)
	if err != nil || site.Scheme == "" || site.Host == "" {
		return nil, fmt.Errorf("invalid site URL %q", opts.SiteURL)
	}
	if opts.Markdown == nil {
		return nil, errors.New("a Markdown renderer is required")
	}

	asset, err := assetResolver(opts.Dist)
	if err != nil {
		return nil, err
	}
	stylesheet, err := asset("assets/main.css")
	if err != nil {
		return nil, err
	}
	script, err := asset("assets/site.ts")
	if err != nil {
		return nil, err
	}

	s := &Server{
		store:      store,
		markdown:   opts.Markdown,
		siteURL:    strings.TrimSuffix(opts.SiteURL, "/"),
		siteHost:   site.Host,
		stylesheet: stylesheet,
		script:     script,
	}

	// The URLs match the SvelteKit app's routes. The router redirects paths with
	// empty or dot segments to their clean version.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handle(s.home))
	mux.HandleFunc("GET /about", s.handle(s.about))
	mux.HandleFunc("GET /cerca", s.handle(s.search))
	mux.HandleFunc("GET /feed.xml", s.handle(s.feed))
	mux.HandleFunc("GET /archivio/{tag}", s.handle(s.tag))
	mux.HandleFunc("GET /archivio/{year}/{month}", s.handle(s.month))
	mux.HandleFunc("GET /{segment}", s.handle(s.postOrCategory))
	mux.HandleFunc("GET /{number}/{slug}", s.handle(s.postWithSlug))
	mux.HandleFunc("GET /healthz", s.handle(s.healthz))
	mux.Handle("GET /assets/{file}", hashedAssets(opts.Dist))
	mux.HandleFunc("GET /favicon.png", favicon(opts.Dist))
	mux.HandleFunc("/", s.handle(s.notFound))

	return logRequests(recoverPanics(trimTrailingSlash(mux))), nil
}

// trimTrailingSlash redirects /path/ to /path, like the SvelteKit app, so old
// links with a trailing slash keep working.
func trimTrailingSlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		if path == "/" || !strings.HasSuffix(path, "/") {
			next.ServeHTTP(w, r)
			return
		}

		// A single leading slash, so //example.com/ can't redirect to another host
		target := "/" + strings.Trim(path, "/")
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		redirect(w, target, http.StatusPermanentRedirect)
	})
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
