package server

import (
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

	uploaded, err := s.uploader.Upload(r.Context(), data, header.Filename)
	if errors.Is(err, images.ErrProcessing) {
		// Usually a file that isn't an image, so it's the request's fault
		slog.Warn("upload rejected", "file", header.Filename, "error", err)
		return errBadRequest
	}
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, uploaded)
}
