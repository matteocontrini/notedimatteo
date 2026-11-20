//go:build vipscheck

// Command vipscheck proves that govips builds, links, and runs against the
// libvips in the Docker image. It's only built by the Dockerfile's vipscheck
// stage, and should be removed once the upload endpoint uses govips.
package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/rand/v2"

	"github.com/davidbyttow/govips/v2/vips"
)

func main() {
	vips.LoggingSettings(nil, vips.LogLevelWarning)
	if err := vips.Startup(nil); err != nil {
		log.Fatal(err)
	}
	defer vips.Shutdown()

	source, err := noisePNG(1600, 1200)
	if err != nil {
		log.Fatal(err)
	}

	plain := vips.NewJpegExportParams()
	plain.Quality = 85
	plainJPEG, err := resizeToJPEG(source, plain)
	if err != nil {
		log.Fatal(err)
	}

	// The options that sharp's mozjpeg preset enables. libvips ignores them
	// if its JPEG library doesn't support them.
	mozjpeg := vips.NewJpegExportParams()
	mozjpeg.Quality = 85
	mozjpeg.OptimizeCoding = true
	mozjpeg.Interlace = true
	mozjpeg.TrellisQuant = true
	mozjpeg.OvershootDeringing = true
	mozjpeg.OptimizeScans = true
	mozjpeg.QuantTable = 3
	mozjpegJPEG, err := resizeToJPEG(source, mozjpeg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("libvips %s\n", vips.Version)
	fmt.Printf("plain JPEG: %d bytes, mozjpeg options: %d bytes\n", len(plainJPEG), len(mozjpegJPEG))
}

func resizeToJPEG(source []byte, params *vips.JpegExportParams) ([]byte, error) {
	img, err := vips.NewImageFromBuffer(source)
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	defer img.Close()

	if err := img.AutoRotate(); err != nil {
		return nil, fmt.Errorf("rotate: %w", err)
	}
	if err := img.Resize(1200.0/float64(img.Width()), vips.KernelLanczos3); err != nil {
		return nil, fmt.Errorf("resize: %w", err)
	}
	if img.Width() != 1200 {
		return nil, fmt.Errorf("resized width = %d", img.Width())
	}

	out, _, err := img.ExportJpeg(params)
	if err != nil {
		return nil, fmt.Errorf("export: %w", err)
	}
	return out, nil
}

// noisePNG returns a blurry-free test image with enough detail for JPEG
// encoder settings to change the output size.
func noisePNG(width, height int) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			base := uint8(x * 255 / width)
			img.Set(x, y, color.RGBA{R: base + uint8(rand.IntN(32)), G: uint8(y * 255 / height), B: uint8(rand.IntN(64)), A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
