// Package libvips processes uploaded images with libvips, through govips.
//
// It's the only package that needs libvips to compile: run Go commands with
// PKG_CONFIG_LIBDIR=.libvips (see scripts/libvips.mjs).
package libvips

import "C" // Compiles magick.c, which govips needs to link

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

// Start starts libvips with its operation cache turned off, so memory goes
// back down after each upload. Call the returned function to stop it.
func Start() (stop func(), err error) {
	vips.LoggingSettings(func(domain string, level vips.LogLevel, message string) {
		slog.Warn("libvips", "domain", domain, "message", message)
	}, vips.LogLevelWarning)

	// Zero values turn the cache off; Startup(nil) would use govips's defaults
	if err := vips.Startup(&vips.Config{}); err != nil {
		return nil, fmt.Errorf("start libvips: %w", err)
	}
	return vips.Shutdown, nil
}

// Processor makes the JPEGs the SvelteKit app made with sharp.
type Processor struct{}

// Process rotates the image as its EXIF orientation says, converts it to sRGB,
// and encodes the main JPEG and a preview at most 1200 pixels wide. Both have
// no metadata.
func (Processor) Process(data []byte) (images.Processed, error) {
	img, err := vips.NewImageFromBuffer(data)
	if err != nil {
		return images.Processed{}, fmt.Errorf("%w: %w", images.ErrUnreadable, err)
	}
	defer img.Close()

	if err := img.AutoRotate(); err != nil {
		return images.Processed{}, fmt.Errorf("rotate: %w", err)
	}
	// Like sharp: convert with the embedded color profile, then drop it
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

// jpegParams returns sharp's settings with its mozjpeg option.
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
