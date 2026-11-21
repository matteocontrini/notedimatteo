package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/matteocontrini/notedimatteo/internal/images"
)

// uploadSizeLimit matches BODY_SIZE_LIMIT in the SvelteKit app's production
// environment.
const uploadSizeLimit = 20 << 20

// upload stores the image in the form's file field, and answers its keys and
// the preview's size.
func (s *Server) upload(w http.ResponseWriter, r *http.Request) error {
	if s.uploader == nil {
		return errUnavailable
	}
	if err := parseForm(w, r, uploadSizeLimit); err != nil {
		return err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return errBadRequest
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// The upload continues if the client goes away, so the main image is never
	// stored without its preview. The server's write timeout still ends it.
	uploaded, err := s.uploader.Upload(context.WithoutCancel(r.Context()), data, header.Filename)
	if errors.Is(err, images.ErrUnreadable) {
		// The client sent a file that isn't an image, or one libvips can't read
		slog.Warn("upload rejected", "file", header.Filename, "error", err)
		return errBadRequest
	}
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, uploaded)
}
