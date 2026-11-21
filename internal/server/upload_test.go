package server

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matteocontrini/notedimatteo/internal/images"
)

type fakeProcessor struct{}

func (fakeProcessor) Process(data []byte) (images.Processed, error) {
	if string(data) != "image" {
		return images.Processed{}, errors.New("not an image")
	}
	return images.Processed{Main: data, Preview: data, PreviewWidth: 1200, PreviewHeight: 800}, nil
}

type fakeStorage struct{ keys []string }

func (f *fakeStorage) Put(_ context.Context, key string, _ []byte, _ string, _ map[string]string) error {
	f.keys = append(f.keys, key)
	return nil
}

// uploadRequest posts a file in the field the editor uses, with the session
// cookie of loggedInStore.
func uploadRequest(t *testing.T, field string, data []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile(field, "photo.jpg")
	if err != nil {
		t.Fatal(err)
	}
	part.Write(data)
	mw.Close()

	r := newRequest(http.MethodPost, "/5/edit/upload", &body, "token")
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

// serveUpload sends r to a handler whose uploads use storage, or are off if
// storage is nil.
func serveUpload(t *testing.T, storage *fakeStorage, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()

	opts := testOptions(t)
	if storage != nil {
		opts.Uploader = images.NewUploader(fakeProcessor{}, storage)
	}
	handler, err := New(loggedInStore(), opts)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)
	return rec
}

func TestUpload(t *testing.T) {
	storage := &fakeStorage{}
	rec := serveUpload(t, storage, uploadRequest(t, "file", []byte("image")))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}

	got := decodeJSON[images.Uploaded](t, rec.Body.String())
	if got.Width != 1200 || got.Height != 800 || len(storage.keys) != 2 ||
		storage.keys[0] != got.Main || storage.keys[1] != got.Preview {
		t.Errorf("response %s, stored %v", rec.Body, storage.keys)
	}
}

func TestUploadErrors(t *testing.T) {
	loggedOut := uploadRequest(t, "file", []byte("image"))
	loggedOut.Header.Del("Cookie")

	tests := []struct {
		name    string
		storage *fakeStorage
		r       *http.Request
		status  int
	}{
		{"uploads off", nil, uploadRequest(t, "file", []byte("image")), http.StatusServiceUnavailable},
		{"logged out", &fakeStorage{}, loggedOut, http.StatusUnauthorized},
		{"no file", &fakeStorage{}, uploadRequest(t, "other", []byte("image")), http.StatusBadRequest},
		{"not an image", &fakeStorage{}, uploadRequest(t, "file", []byte("text")), http.StatusBadRequest},
		{"too large", &fakeStorage{}, uploadRequest(t, "file", make([]byte, uploadSizeLimit)), http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		rec := serveUpload(t, tt.storage, tt.r)
		if rec.Code != tt.status {
			t.Errorf("%s: status = %d: %s", tt.name, rec.Code, rec.Body)
		}
		if tt.storage != nil && len(tt.storage.keys) != 0 {
			t.Errorf("%s: stored %v", tt.name, tt.storage.keys)
		}
	}
}
