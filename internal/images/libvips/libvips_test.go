package libvips

import (
	"bytes"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"math"
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

// p3PNG returns a PNG of one sRGB color converted to Display P3, with the P3
// profile embedded, like a screenshot on a recent Mac. It also returns the P3
// pixel values.
func p3PNG(t *testing.T, srgb color.NRGBA) ([]byte, [3]float64) {
	t.Helper()

	src := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			src.Set(x, y, srgb)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}

	img, err := vips.NewImageFromBuffer(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer img.Close()
	// libvips knows "p3" by name, and embeds the output profile
	if err := img.TransformICCProfileWithFallback("p3", "srgb"); err != nil {
		t.Fatal(err)
	}
	data, _, err := img.ExportPng(&vips.PngExportParams{})
	if err != nil {
		t.Fatal(err)
	}
	return data, averageColor(t, data)
}

// averageColor returns the mean of each channel, which evens out JPEG noise.
func averageColor(t *testing.T, data []byte) [3]float64 {
	t.Helper()

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	var sum [3]float64
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			sum[0] += float64(r >> 8)
			sum[1] += float64(g >> 8)
			sum[2] += float64(bl >> 8)
		}
	}
	n := float64(b.Dx() * b.Dy())
	return [3]float64{sum[0] / n, sum[1] / n, sum[2] / n}
}

// TestProcessConvertsColorProfiles checks that a Display P3 image keeps its
// colors. Dropping the profile without converting the pixels, as Hugo did to
// the blog's images, would leave the P3 values, and the image would look washed out.
func TestProcessConvertsColorProfiles(t *testing.T) {
	srgb := color.NRGBA{200, 60, 40, 255}
	data, p3 := p3PNG(t, srgb)

	input, err := vips.NewImageFromBuffer(data)
	if err != nil {
		t.Fatal(err)
	}
	hasProfile := input.HasICCProfile()
	input.Close()
	if !hasProfile {
		t.Fatal("the test image has no ICC profile")
	}
	// Check the test is meaningful: P3 stores this color with other numbers
	if math.Abs(p3[0]-float64(srgb.R)) < 10 {
		t.Fatalf("P3 values %v are too close to sRGB %v to tell them apart", p3, srgb)
	}

	processed, err := Processor{}.Process(data)
	if err != nil {
		t.Fatal(err)
	}
	for name, out := range map[string][]byte{"main": processed.Main, "preview": processed.Preview} {
		img, err := vips.NewImageFromBuffer(out)
		if err != nil {
			t.Fatal(err)
		}
		if img.HasICCProfile() {
			t.Errorf("%s keeps an ICC profile", name)
		}
		img.Close()

		got := averageColor(t, out)
		want := [3]float64{float64(srgb.R), float64(srgb.G), float64(srgb.B)}
		for i := range got {
			if math.Abs(got[i]-want[i]) > 3 {
				t.Errorf("%s is %.0f, want the original sRGB %v, not the P3 values %.0f", name, got, srgb, p3)
				break
			}
		}
	}
}
