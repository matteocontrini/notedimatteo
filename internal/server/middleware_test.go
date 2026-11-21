package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunHandler(t *testing.T) {
	failure := errors.New("failure")

	// A handler that fails before writing gets its error back, for the error page
	rec := httptest.NewRecorder()
	err := runHandler(func(http.ResponseWriter, *http.Request) error { return failure }, rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if !errors.Is(err, failure) {
		t.Errorf("error before the response = %v", err)
	}

	// After the status is sent, an error page can't follow
	for name, write := range map[string]func(http.ResponseWriter){
		"status": func(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) },
		"body":   func(w http.ResponseWriter) { w.Write([]byte("partial")) },
	} {
		rec := httptest.NewRecorder()
		err := runHandler(func(w http.ResponseWriter, _ *http.Request) error {
			write(w)
			return failure
		}, rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if err != nil {
			t.Errorf("error after writing the %s = %v", name, err)
		}
	}
}
