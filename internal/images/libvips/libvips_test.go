package libvips

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

func TestMain(m *testing.M) {
	stop, err := Start()
	if err != nil {
		panic(err)
	}
	code := m.Run()
	stop()
	os.Exit(code)
}

// testPNG returns a PNG with a gradient, and transparency if alpha is set.
func testPNG(t *testing.T, width, height int, alpha bool) []byte {
	t.Helper()

	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			a := uint8(255)
			if alpha && x < width/2 {
				a = 0
			}
			img.Set(x, y, color.NRGBA{uint8(x), uint8(y), 128, a})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// rotatedJPEG returns a JPEG whose EXIF orientation says to rotate it 90°.
func rotatedJPEG(t *testing.T, width, height int) []byte {
	t.Helper()

	img, err := vips.NewImageFromBuffer(testPNG(t, width, height, false))
	if err != nil {
		t.Fatal(err)
	}
	defer img.Close()
	if err := img.SetOrientation(6); err != nil {
		t.Fatal(err)
	}
	data, _, err := img.ExportJpeg(&vips.JpegExportParams{Quality: 90})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// size decodes a JPEG and returns its size and orientation.
func size(t *testing.T, data []byte) (width, height, orientation int) {
	t.Helper()

	img, err := vips.NewImageFromBuffer(data)
	if err != nil {
		t.Fatal(err)
	}
	defer img.Close()
	if img.Format() != vips.ImageTypeJPEG {
		t.Errorf("format = %v", img.Format())
	}
	return img.Width(), img.Height(), img.Orientation()
}

func TestProcess(t *testing.T) {
	tests := []struct {
		name                        string
		data                        []byte
		mainWidth, mainHeight       int
		previewWidth, previewHeight int
	}{
		{"wide", testPNG(t, 3000, 1500, false), 3000, 1500, 1200, 600},
		{"narrow", testPNG(t, 800, 600, false), 800, 600, 800, 600},
		{"transparent", testPNG(t, 400, 300, true), 400, 300, 400, 300},
		// EXIF orientation 6: the stored 2000×1000 pixels show as 1000×2000
		{"rotated", rotatedJPEG(t, 2000, 1000), 1000, 2000, 1000, 2000},
	}

	for _, tt := range tests {
		processed, err := Processor{}.Process(tt.data)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}

		width, height, orientation := size(t, processed.Main)
		if width != tt.mainWidth || height != tt.mainHeight || orientation != 0 {
			t.Errorf("%s: main is %d×%d, orientation %d", tt.name, width, height, orientation)
		}
		width, height, _ = size(t, processed.Preview)
		if width != tt.previewWidth || height != tt.previewHeight {
			t.Errorf("%s: preview is %d×%d", tt.name, width, height)
		}
		if processed.PreviewWidth != width || processed.PreviewHeight != height {
			t.Errorf("%s: reported preview size %d×%d", tt.name, processed.PreviewWidth, processed.PreviewHeight)
		}
	}
}

func TestProcessInvalid(t *testing.T) {
	// BMP goes through the ImageMagick loader, which magick.c replaces
	bmp := append([]byte("BM"), make([]byte, 100)...)
	for name, data := range map[string][]byte{"text": []byte("not an image"), "BMP": bmp} {
		if _, err := (Processor{}).Process(data); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
