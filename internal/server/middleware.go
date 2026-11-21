package server

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
)

// handlerFunc is a page handler. Instead of writing an error response itself,
// it returns the error, and handle shows the error page.
type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// handle loads the session and runs h. If h returns an error, handle shows an
// error page, and logs the error if it's unexpected.
func (s *Server) handle(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r, err := s.loadSession(w, r)
		if err == nil {
			err = runHandler(h, w, r)
		}
		if err != nil {
			s.renderError(w, r, errorStatus(r, err))
		}
	}
}

// handleAPI is handle for the editor's JSON endpoints. It returns 401 for
// logged-out requests, and writes errors as JSON, like {"error": "Not Found"}.
func (s *Server) handleAPI(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r, err := s.loadSession(w, r)
		if err == nil && !isLoggedIn(r) {
			err = errUnauthorized
		}
		if err == nil {
			err = runHandler(h, w, r)
		}
		if err != nil {
			status := errorStatus(r, err)
			if err := writeJSON(w, status, map[string]string{"error": http.StatusText(status)}); err != nil {
				slog.Error("writing JSON error failed", "path", r.URL.Path, "error", err)
			}
		}
	}
}

// runHandler runs h and returns its error, unless h had already started the
// response. Then the client already has a status, so no error response can
// follow. That error usually means the client went away during the response.
func runHandler(h handlerFunc, w http.ResponseWriter, r *http.Request) error {
	sw := &startWriter{ResponseWriter: w}
	err := h(sw, r)
	if err != nil && sw.started {
		slog.Debug("response failed after it started", "path", r.URL.Path, "error", err)
		return nil
	}
	return err
}

// startWriter records whether a handler wrote a status or any of the body.
type startWriter struct {
	http.ResponseWriter
	started bool
}

func (w *startWriter) WriteHeader(status int) {
	w.started = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *startWriter) Write(b []byte) (int, error) {
	w.started = true
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the connection.
func (w *startWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
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
