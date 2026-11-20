package server

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/matteocontrini/notedimatteo/internal/db"
)

func TestHome(t *testing.T) {
	cached := "<p>Cached <del>body</del></p>"
	store := &fakeStore{total: 45, posts: []db.Post{{Number: 42, CachedHTML: &cached}}}

	rec := get(t, store, "/?page=2")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if store.lastOffset != 20 {
		t.Errorf("offset = %d", store.lastOffset)
	}
	assertContains(t, rec, cached)
}

func TestHomeDatabaseError(t *testing.T) {
	rec := get(t, &fakeStore{err: errors.New("secret connection details")}, "/")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Error("response leaks the error")
	}
}
