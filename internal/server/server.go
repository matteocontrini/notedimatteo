// Package server contains the HTTP routes, handlers, and middleware.
package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"github.com/matteocontrini/notedimatteo/internal/db"
	"github.com/matteocontrini/notedimatteo/internal/images"
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

	SessionByToken(ctx context.Context, token string) (*db.Session, error)
	CreateSession(ctx context.Context, token string) error
	TouchSession(ctx context.Context, id int) error
	DeleteSession(ctx context.Context, token string) error

	Drafts(ctx context.Context) ([]db.Post, error)
	CountDrafts(ctx context.Context) (int, error)
	CreateDraft(ctx context.Context) (int, error)
	DeleteDraft(ctx context.Context, number int) (bool, error)

	PostForEdit(ctx context.Context, number int) (*db.EditablePost, error)
	SavePost(ctx context.Context, number int, edit db.PostEdit) (*db.EditablePost, error)
}

type Options struct {
	// SiteURL is the public address, like https://notedimatteo.it.
	SiteURL string
	// Dist holds the assets built by Vite, with their manifest.
	Dist fs.FS
	// Markdown renders post bodies that have no up-to-date cached HTML.
	Markdown *markdown.Renderer
	// AdminUsername and AdminPassword are the only login credentials.
	AdminUsername string
	AdminPassword string
	// Uploader stores images uploaded from the editor. If it's nil, uploads
	// answer 503.
	Uploader *images.Uploader
}

type Server struct {
	store      Store
	markdown   *markdown.Renderer
	siteURL    string
	siteHost   string
	stylesheet string
	script     string
	// editorScript is the URL of the Svelte editor, which only the edit page loads.
	editorScript string

	adminUsername string
	adminPassword string
	uploader      *images.Uploader
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
	if opts.AdminUsername == "" || opts.AdminPassword == "" {
		return nil, errors.New("admin credentials are required")
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
	editorScript, err := asset("editor/main.ts")
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

		editorScript: editorScript,

		adminUsername: opts.AdminUsername,
		adminPassword: opts.AdminPassword,
		uploader:      opts.Uploader,
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
	mux.HandleFunc("GET /login", s.handle(s.loginPage))
	mux.HandleFunc("POST /login", s.handle(s.login))
	mux.HandleFunc("POST /logout", s.handle(s.logout))
	mux.HandleFunc("GET /unpublished", s.handle(requireLogin(s.unpublished)))
	mux.HandleFunc("POST /unpublished/new", s.handle(requireLogin(s.newDraft)))
	mux.HandleFunc("POST /{number}/delete", s.handle(requireLogin(s.deleteDraft)))
	mux.HandleFunc("POST /{number}/edit", s.handleAPI(s.savePost))
	mux.HandleFunc("POST /{number}/edit/preview", s.handleAPI(s.preview))
	mux.HandleFunc("POST /{number}/edit/upload", s.handleAPI(s.upload))
	mux.HandleFunc("GET /healthz", s.handle(s.healthz))
	mux.Handle("GET /assets/{file}", hashedAssets(opts.Dist))
	mux.HandleFunc("GET /favicon.png", favicon(opts.Dist))
	mux.HandleFunc("/", s.handle(s.notFound))

	// CrossOriginProtection rejects POSTs from other sites, using the
	// Sec-Fetch-Site and Origin headers
	csrf := http.NewCrossOriginProtection()
	csrf.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.renderError(w, r, http.StatusForbidden)
	}))

	return logRequests(recoverPanics(trimTrailingSlash(csrf.Handler(mux)))), nil
}

// healthz reports whether the server can reach the database, for health checks.
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) error {
	if err := s.store.Ping(r.Context()); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte("ok\n"))
	return err
}
