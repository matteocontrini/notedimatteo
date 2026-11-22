// Package libvips processes uploaded images with libvips, through govips.
//
// It's the only package that needs libvips to compile: run Go commands with
// PKG_CONFIG_LIBDIR=.libvips (see scripts/libvips.mjs).
package libvips

/*
#include <stdlib.h>
#ifdef __GLIBC__
#include <malloc.h>
#else
// Do nothing on macOS, which has no malloc_trim and doesn't need it
static int malloc_trim(size_t pad) { return 0; }
#endif
*/
import "C" // Also compiles magick.c, which govips needs to link

import (
	"fmt"
	"log/slog"

	"github.com/davidbyttow/govips/v2/vips"

	"github.com/matteocontrini/notedimatteo/internal/images"
)

const (
	mainQuality    = 85
	previewQuality = 80
	// previewWidth is the preview's largest width. Narrower images keep their size.
	previewWidth = 1200
)

// Start starts libvips with one worker thread and its operation cache turned
// off, so memory goes back down after each upload. Call the returned function
// to stop it.
func Start() (stop func(), err error) {
	vips.LoggingSettings(func(domain string, level vips.LogLevel, message string) {
		slog.Warn("libvips", "domain", domain, "message", message)
	}, vips.LogLevelWarning)

	config := &vips.Config{
		// Set concurrency to 1 thread to keep memory low, because each thread holds its own buffers.
		// Without this, libvips starts a thread for each CPU of the host.
		ConcurrencyLevel: 1,
		// Turn off libvips cache
		MaxCacheMem: 0, MaxCacheSize: 0, MaxCacheFiles: 0,
	}
	if err := vips.Startup(config); err != nil {
		return nil, fmt.Errorf("start libvips: %w", err)
	}
	return vips.Shutdown, nil
}

// Processor create the two JPEGs from an uploaded image with libvips.
type Processor struct{}

// Process rotates the image as its EXIF orientation says, converts it to sRGB,
// and encodes the main JPEG and a preview at most 1200 pixels wide. Both have
// no metadata.
func (Processor) Process(data []byte) (images.Processed, error) {
	// Release memory to the system after processing, otherwise glibc keeps
	// the memory even after libvips frees it
	defer C.malloc_trim(0)

	img, err := vips.NewImageFromBuffer(data)
	if err != nil {
		return images.Processed{}, fmt.Errorf("%w: %w", images.ErrUnreadable, err)
	}
	defer img.Close()

	if err := img.AutoRotate(); err != nil {
		return images.Processed{}, fmt.Errorf("rotate: %w", err)
	}
	// Convert the pixels with the embedded color profile, the export then drops it
	if img.HasICCProfile() {
		if err := img.TransformICCProfile(vips.SRGBIEC6196621ICCProfilePath); err != nil {
			return images.Processed{}, fmt.Errorf("convert to sRGB: %w", err)
		}
	} else if err := img.ToColorSpace(vips.InterpretationSRGB); err != nil {
		return images.Processed{}, fmt.Errorf("convert to sRGB: %w", err)
	}

	main, _, err := img.ExportJpeg(jpegParams(mainQuality))
	if err != nil {
		return images.Processed{}, fmt.Errorf("encode: %w", err)
	}

	if img.Width() > previewWidth {
		if err := img.Resize(float64(previewWidth)/float64(img.Width()), vips.KernelLanczos3); err != nil {
			return images.Processed{}, fmt.Errorf("resize: %w", err)
		}
	}
	preview, _, err := img.ExportJpeg(jpegParams(previewQuality))
	if err != nil {
		return images.Processed{}, fmt.Errorf("encode preview: %w", err)
	}

	return images.Processed{
		Main:          main,
		Preview:       preview,
		PreviewWidth:  img.Width(),
		PreviewHeight: img.Height(),
	}, nil
}

// jpegParams returns the settings of sharp's mozjpeg preset.
func jpegParams(quality int) *vips.JpegExportParams {
	return &vips.JpegExportParams{
		StripMetadata:      true,
		Quality:            quality,
		Interlace:          true,
		OptimizeCoding:     true,
		TrellisQuant:       true,
		OvershootDeringing: true,
		OptimizeScans:      true,
		QuantTable:         3,
	}
}
