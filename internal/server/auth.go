package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/matteocontrini/notedimatteo/internal/views"
)

const (
	// SvelteKit uses the same cookie name, so users stay logged in after the switch.
	sessionCookie   = "notedimatteo_session"
	sessionLifetime = 60 * 24 * time.Hour
	// The server updates a session's last activity at most once per activityInterval.
	activityInterval = time.Minute
	// parseForm rejects form bodies larger than formSizeLimit.
	formSizeLimit = 64 << 10
)

type contextKey int

const loggedInKey contextKey = 0

// isLoggedIn reports whether loadSession found a valid session for the request.
func isLoggedIn(r *http.Request) bool {
	loggedIn, _ := r.Context().Value(loggedInKey).(bool)
	return loggedIn
}

// loadSession looks up the session token in the request's cookie. If the
// session exists, it returns the request marked as logged in. If not, it
// deletes the cookie.
//
// TODO: after the migration, expire old sessions in the database too. Today
// only the browser drops the cookie, after 60 days.
func (s *Server) loadSession(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return r, nil
	}

	session, err := s.store.SessionByToken(r.Context(), cookie.Value)
	if err != nil {
		return r, err
	}
	if session == nil {
		http.SetCookie(w, newSessionCookie(r, "", -1))
		return r, nil
	}

	if time.Since(session.LastActivityAt) > activityInterval {
		// Nothing reads the activity time, so a failed update only logs an error
		if err := s.store.TouchSession(r.Context(), session.ID); err != nil {
			slog.Error("updating session activity failed", "error", err)
		}
	}

	return r.WithContext(context.WithValue(r.Context(), loggedInKey, true)), nil
}

// newSessionCookie returns a session cookie that holds token. A negative
// maxAge tells the browser to delete the cookie.
func newSessionCookie(r *http.Request, token string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Local development uses plain HTTP, where some browsers drop Secure cookies
		Secure: !isLocalhost(r.Host),
	}
}

func isLocalhost(host string) bool {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// requireLogin redirects logged-out requests to the login page.
func requireLogin(h handlerFunc) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if !isLoggedIn(r) {
			redirect(w, "/login", http.StatusSeeOther)
			return nil
		}
		return h(w, r)
	}
}

// parseForm parses the form body. It returns errBadRequest if the body is
// invalid or too large.
func parseForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, formSizeLimit)
	if err := r.ParseForm(); err != nil {
		return errBadRequest
	}
	return nil
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) error {
	return s.renderLogin(w, r, http.StatusOK, "", "")
}

// login checks the credentials and starts a session.
//
// TODO: after the migration, limit failed login attempts.
func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	if err := parseForm(w, r); err != nil {
		return err
	}
	username, password := r.PostForm.Get("username"), r.PostForm.Get("password")

	if username == "" || password == "" {
		return s.renderLogin(w, r, http.StatusBadRequest, username, "Compila tutti i campi.")
	}
	if !s.validCredentials(username, password) {
		return s.renderLogin(w, r, http.StatusBadRequest, username, "Credenziali non valide.")
	}

	token := rand.Text()
	if err := s.store.CreateSession(r.Context(), token); err != nil {
		return err
	}
	http.SetCookie(w, newSessionCookie(r, token, int(sessionLifetime.Seconds())))
	redirect(w, "/", http.StatusSeeOther)
	return nil
}

// validCredentials compares in constant time, so response times don't reveal
// how close a guess is.
func (s *Server) validCredentials(username, password string) bool {
	usernameOK := subtle.ConstantTimeCompare([]byte(username), []byte(s.adminUsername))
	passwordOK := subtle.ConstantTimeCompare([]byte(password), []byte(s.adminPassword))
	return usernameOK&passwordOK == 1
}

func (s *Server) renderLogin(w http.ResponseWriter, r *http.Request, status int, username, message string) error {
	page, err := s.newPage(r, views.Meta{})
	if err != nil {
		return err
	}
	return s.render(w, r, status, views.Login(page, username, message))
}

// logout deletes the session. SvelteKit used a GET link; this is a POST, so
// other sites can't log the user out.
func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		if err := s.store.DeleteSession(r.Context(), cookie.Value); err != nil {
			return err
		}
		http.SetCookie(w, newSessionCookie(r, "", -1))
	}
	redirect(w, "/", http.StatusSeeOther)
	return nil
}
