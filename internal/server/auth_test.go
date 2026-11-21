package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

// send makes a request. A non-nil form becomes a URL-encoded body, and a
// non-empty token becomes the session cookie.
func send(t *testing.T, store Store, method, path string, form url.Values, token string) *httptest.ResponseRecorder {
	t.Helper()

	if form == nil {
		return serve(t, store, newRequest(method, path, nil, token))
	}
	r := newRequest(method, path, strings.NewReader(form.Encode()), token)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return serve(t, store, r)
}

// newRequest returns a request with the session cookie if token isn't empty.
func newRequest(method, path string, body io.Reader, token string) *http.Request {
	r := httptest.NewRequest(method, path, body)
	if token != "" {
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: token})
	}
	return r
}

// serve sends r to a test handler using store.
func serve(t *testing.T, store Store, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	newTestHandler(t, store).ServeHTTP(rec, r)
	return rec
}

// loggedInStore returns a store with one session, "token", active a moment ago.
func loggedInStore() *fakeStore {
	return &fakeStore{sessions: map[string]*db.Session{
		"token": {ID: 1, LastActivityAt: time.Now()},
	}}
}

func responseCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	return nil
}

func TestLoggedOutAccess(t *testing.T) {
	tests := []struct {
		method, path string
		status       int
		location     string
	}{
		{http.MethodGet, "/login", http.StatusOK, ""},
		{http.MethodGet, "/unpublished", http.StatusSeeOther, "/login"},
		{http.MethodPost, "/unpublished/new", http.StatusSeeOther, "/login"},
		{http.MethodPost, "/5/delete", http.StatusSeeOther, "/login"},
		{http.MethodGet, "/5/edit", http.StatusSeeOther, "/login"},
		// SvelteKit's GET links no longer exist
		{http.MethodGet, "/unpublished/new", http.StatusNotFound, ""},
		{http.MethodGet, "/logout", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		store := &fakeStore{drafts: []db.Post{{Number: 5}}}
		rec := send(t, store, tt.method, tt.path, url.Values{}, "")

		if rec.Code != tt.status || rec.Header().Get("Location") != tt.location {
			t.Errorf("%s %s = %d %q", tt.method, tt.path, rec.Code, rec.Header().Get("Location"))
		}
		if store.createdDraft != 0 || len(store.deletedDrafts) != 0 {
			t.Errorf("%s %s changed drafts", tt.method, tt.path)
		}
	}
}

func TestLogin(t *testing.T) {
	tests := []struct {
		username, password string
		message            string
	}{
		{"", "", "Compila tutti i campi."},
		{"admin", "", "Compila tutti i campi."},
		{"admin", "wrong", "Credenziali non valide."},
		{"other", "secret", "Credenziali non valide."},
	}
	for _, tt := range tests {
		store := &fakeStore{}
		rec := send(t, store, http.MethodPost, "/login", url.Values{"username": {tt.username}, "password": {tt.password}}, "")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%q/%q: status = %d", tt.username, tt.password, rec.Code)
		}
		assertContains(t, rec, tt.message)
		if len(store.sessions) != 0 || responseCookie(rec) != nil {
			t.Errorf("%q/%q: started a session", tt.username, tt.password)
		}
	}

	store := &fakeStore{}
	rec := send(t, store, http.MethodPost, "/login", url.Values{"username": {"admin"}, "password": {"secret"}}, "")

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Errorf("status = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
	}
	cookie := responseCookie(rec)
	if cookie == nil || store.sessions[cookie.Value] == nil {
		t.Fatalf("cookie %v doesn't match a session", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode ||
		cookie.Path != "/" || cookie.MaxAge != 60*24*60*60 {
		t.Errorf("cookie = %+v", cookie)
	}
}

func TestSessionCookieOnLocalhost(t *testing.T) {
	for host, secure := range map[string]bool{
		"localhost:6200":  false,
		"127.0.0.1:6200":  false,
		"[::1]:6200":      false,
		"notedimatteo.it": true,
	} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = host
		if got := newSessionCookie(r, "token", 1).Secure; got != secure {
			t.Errorf("%s: Secure = %v", host, got)
		}
	}
}

func TestSessionLoading(t *testing.T) {
	store := loggedInStore()
	if rec := send(t, store, http.MethodGet, "/unpublished", nil, "token"); rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	if len(store.touched) != 0 {
		t.Error("updated a session active a moment ago")
	}

	store.sessions["token"].LastActivityAt = time.Now().Add(-2 * time.Minute)
	send(t, store, http.MethodGet, "/about", nil, "token")
	if len(store.touched) != 1 || store.touched[0] != 1 {
		t.Errorf("touched = %v", store.touched)
	}

	// An unknown token logs the request out and deletes the cookie
	rec := send(t, store, http.MethodGet, "/unpublished", nil, "unknown")
	if rec.Code != http.StatusSeeOther {
		t.Errorf("unknown token: status = %d", rec.Code)
	}
	if cookie := responseCookie(rec); cookie == nil || cookie.MaxAge >= 0 {
		t.Errorf("unknown token: cookie = %v", cookie)
	}
}

func TestLogout(t *testing.T) {
	store := loggedInStore()
	rec := send(t, store, http.MethodPost, "/logout", url.Values{}, "token")

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Errorf("status = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
	}
	if store.sessions["token"] != nil {
		t.Error("session wasn't deleted")
	}
	if cookie := responseCookie(rec); cookie == nil || cookie.MaxAge >= 0 {
		t.Errorf("cookie = %v", cookie)
	}
}

func TestCrossOriginPostsAreRejected(t *testing.T) {
	for site, status := range map[string]int{
		"cross-site":  http.StatusForbidden,
		"same-site":   http.StatusForbidden,
		"same-origin": http.StatusSeeOther,
	} {
		store := loggedInStore()
		r := httptest.NewRequest(http.MethodPost, "/unpublished/new", nil)
		r.Header.Set("Sec-Fetch-Site", site)
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "token"})

		rec := httptest.NewRecorder()
		newTestHandler(t, store).ServeHTTP(rec, r)

		if rec.Code != status {
			t.Errorf("%s: status = %d", site, rec.Code)
		}
		if created := store.createdDraft != 0; created != (status == http.StatusSeeOther) {
			t.Errorf("%s: created a draft = %v", site, created)
		}
	}
}

func TestDrafts(t *testing.T) {
	store := loggedInStore()
	store.drafts = []db.Post{{ID: 9, Number: 5, CreatedAt: time.Now()}}

	if rec := send(t, store, http.MethodGet, "/unpublished", nil, "token"); rec.Code != http.StatusOK {
		t.Errorf("list: status = %d", rec.Code)
	}
	if _, ok := store.saved[9]; !ok {
		t.Error("list: draft HTML wasn't cached")
	}

	rec := send(t, store, http.MethodPost, "/unpublished/new", url.Values{}, "token")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/671/edit" {
		t.Errorf("new: status = %d, Location = %q", rec.Code, rec.Header().Get("Location"))
	}

	for path, status := range map[string]int{
		"/6/delete":   http.StatusNotFound, // not a draft
		"/abc/delete": http.StatusNotFound,
		"/5/delete":   http.StatusSeeOther,
	} {
		if rec := send(t, store, http.MethodPost, path, url.Values{}, "token"); rec.Code != status {
			t.Errorf("%s: status = %d", path, rec.Code)
		}
	}
	if len(store.deletedDrafts) != 1 || store.deletedDrafts[0] != 5 {
		t.Errorf("deleted = %v", store.deletedDrafts)
	}
}
