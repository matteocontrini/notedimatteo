package images

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"
)

type fakeProcessor struct{ err error }

func (f fakeProcessor) Process([]byte) (Processed, error) {
	return Processed{Main: []byte("main"), Preview: []byte("preview"), PreviewWidth: 1200, PreviewHeight: 800}, f.err
}

type stored struct {
	body        string
	contentType string
	metadata    map[string]string
}

type fakeStorage struct {
	files map[string]stored
	err   error
}

func (f *fakeStorage) Put(_ context.Context, key string, body []byte, contentType string, metadata map[string]string) error {
	if f.err != nil {
		return f.err
	}
	if f.files == nil {
		f.files = map[string]stored{}
	}
	f.files[key] = stored{string(body), contentType, metadata}
	return nil
}

func TestUpload(t *testing.T) {
	storage := &fakeStorage{}
	uploader := NewUploader(fakeProcessor{}, storage)
	uploader.now = func() time.Time { return time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC) }

	uploaded, err := uploader.Upload(context.Background(), []byte("data"), "foto è.png")
	if err != nil {
		t.Fatal(err)
	}

	if !regexp.MustCompile(`^2026/03/[0-9a-f]{12}\.jpg$`).MatchString(uploaded.Main) {
		t.Errorf("main key = %q", uploaded.Main)
	}
	if uploaded.Preview != uploaded.Main[:len(uploaded.Main)-4]+"_preview.jpg" {
		t.Errorf("preview key = %q", uploaded.Preview)
	}
	if uploaded.Width != 1200 || uploaded.Height != 800 {
		t.Errorf("size = %d×%d", uploaded.Width, uploaded.Height)
	}

	main, preview := storage.files[uploaded.Main], storage.files[uploaded.Preview]
	if main.body != "main" || main.contentType != "image/jpeg" || main.metadata["original"] != "=?utf-8?q?foto_=C3=A8.png?=" {
		t.Errorf("main = %+v", main)
	}
	if preview.body != "preview" || preview.metadata != nil {
		t.Errorf("preview = %+v", preview)
	}

	// ASCII names stay as they are
	uploaded, _ = uploader.Upload(context.Background(), []byte("data"), "image.png")
	if got := storage.files[uploaded.Main].metadata["original"]; got != "image.png" {
		t.Errorf("ASCII name = %q", got)
	}
}

func TestUploadErrors(t *testing.T) {
	failure := errors.New("failure")
	for name, uploader := range map[string]*Uploader{
		"processing": NewUploader(fakeProcessor{err: failure}, &fakeStorage{}),
		"storage":    NewUploader(fakeProcessor{}, &fakeStorage{err: failure}),
	} {
		_, err := uploader.Upload(context.Background(), nil, "x.png")
		if !errors.Is(err, failure) || errors.Is(err, ErrProcessing) != (name == "processing") {
			t.Errorf("%s: %v", name, err)
		}
	}
}
