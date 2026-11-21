package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/matteocontrini/notedimatteo/internal/views"
)

// statusError is returned by handlers to show an error page with that status.
type statusError int

func (e statusError) Error() string {
	return http.StatusText(int(e))
}

const (
	errBadRequest   = statusError(http.StatusBadRequest)
	errUnauthorized = statusError(http.StatusUnauthorized)
	errNotFound     = statusError(http.StatusNotFound)
	errTooLarge     = statusError(http.StatusRequestEntityTooLarge)
	errUnavailable  = statusError(http.StatusServiceUnavailable)
)

// errorStatus returns the status for an error returned by a handler. It logs
// errors that aren't a statusError, and returns 500 for them.
func errorStatus(r *http.Request, err error) int {
	if status, ok := errors.AsType[statusError](err); ok {
		return int(status)
	}
	slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	return http.StatusInternalServerError
}

// renderError shows the error page in the site layout, or plain text if the
// layout's data can't be loaded either.
func (s *Server) renderError(w http.ResponseWriter, r *http.Request, status int) {
	page, err := s.newPage(r, views.Meta{})
	if err == nil {
		err = s.render(w, r, status, views.ErrorPage(page, status, http.StatusText(status)))
	}
	if err != nil {
		slog.Error("rendering error page failed", "path", r.URL.Path, "error", err)
		http.Error(w, http.StatusText(status), status)
	}
}

func (s *Server) notFound(http.ResponseWriter, *http.Request) error {
	return errNotFound
}

// redirect responds with a redirect and no body. http.Redirect would also add
// a small HTML page.
func redirect(w http.ResponseWriter, location string, status int) {
	w.Header().Set("Location", location)
	w.WriteHeader(status)
}
