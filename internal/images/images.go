// Package images stores the images uploaded from the editor: a compressed
// JPEG and a smaller preview of each, which posts show with [img] markup.
//
// This package is plain Go. The libvips subpackage processes the images, so
// only it and the server binary need libvips to compile.
package images

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"time"
)

// Processed holds an image's two JPEG versions, and the preview's size.
type Processed struct {
	Main          []byte
	Preview       []byte
	PreviewWidth  int
	PreviewHeight int
}

// Processor turns an uploaded image into its two JPEG versions.
type Processor interface {
	Process(data []byte) (Processed, error)
}

// Storage saves files under a key, like an S3 bucket.
type Storage interface {
	Put(ctx context.Context, key string, body []byte, contentType string, metadata map[string]string) error
}

// Uploaded is where an upload's files are, with the preview's size. The editor
// writes it as [img width=… height=…]main[/img].
type Uploaded struct {
	Main    string `json:"main"`
	Preview string `json:"preview"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// ErrUnreadable means the processor couldn't read the file as an image. The
// processor wraps it; other processing errors are the server's fault.
var ErrUnreadable = errors.New("can't read the file as an image")

type Uploader struct {
	processor Processor
	storage   Storage
	now       func() time.Time
}

func NewUploader(processor Processor, storage Storage) *Uploader {
	return &Uploader{processor: processor, storage: storage, now: time.Now}
}

// Upload processes an image and stores both versions, with keys like
// 2026/11/6d2be8edb5a7.jpg and 2026/11/6d2be8edb5a7_preview.jpg. The main
// image keeps the original file name in its metadata.
func (u *Uploader) Upload(ctx context.Context, data []byte, filename string) (Uploaded, error) {
	processed, err := u.processor.Process(data)
	if err != nil {
		return Uploaded{}, fmt.Errorf("process image: %w", err)
	}

	base := newKeyBase(u.now())
	uploaded := Uploaded{
		Main:    base + ".jpg",
		Preview: base + "_preview.jpg",
		Width:   processed.PreviewWidth,
		Height:  processed.PreviewHeight,
	}

	// S3 metadata must be ASCII, so other names are encoded as RFC 2047 says
	metadata := map[string]string{"original": mime.QEncoding.Encode("utf-8", filename)}
	if err := u.storage.Put(ctx, uploaded.Main, processed.Main, "image/jpeg", metadata); err != nil {
		return Uploaded{}, fmt.Errorf("store image: %w", err)
	}
	if err := u.storage.Put(ctx, uploaded.Preview, processed.Preview, "image/jpeg", nil); err != nil {
		return Uploaded{}, fmt.Errorf("store preview: %w", err)
	}
	return uploaded, nil
}

// newKeyBase returns the year, month, and 12 random hex characters of a new
// upload's keys, like 2026/11/6d2be8edb5a7.
func newKeyBase(now time.Time) string {
	random := make([]byte, 6)
	rand.Read(random)
	return fmt.Sprintf("%d/%02d/%s", now.Year(), now.Month(), hex.EncodeToString(random))
}
