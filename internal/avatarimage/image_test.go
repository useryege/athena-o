package avatarimage

import (
	"bytes"
	"compress/bzip2"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"golang.org/x/image/webp"
)

func TestValidateAcceptsSupportedImages(t *testing.T) {
	pngData := encodeTestImage(t, "png", 3, 2)
	jpegData := encodeTestImage(t, "jpeg", 3, 2)
	tests := []struct {
		name, contentType string
		data              []byte
		width, height     int
	}{
		{"PNG", "image/png", pngData, 3, 2},
		{"JPEG", "image/jpeg", jpegData, 3, 2},
		{"lossy WebP", "image/webp", fixture(t, "blue-purple-pink.lossy.webp"), 150, 100},
		{"extended WebP", "image/webp", fixture(t, "yellow_rose.lossy-with-alpha.webp"), 400, 301},
		{"lossless WebP with skipped Huffman groups", "image/webp", fixture(t, "gopher-doc.skip-hgroup.lossless.webp"), 75, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Validate(bytes.NewReader(tt.data), int64(len(tt.data)))
			if err != nil {
				t.Fatalf("Validate: %v", err)
			}
			if got.ContentType != tt.contentType || got.Width != tt.width || got.Height != tt.height {
				t.Fatalf("metadata = (%q, %d, %d), want (%q, %d, %d)", got.ContentType, got.Width, got.Height, tt.contentType, tt.width, tt.height)
			}
			if !bytes.Equal(got.Data, tt.data) {
				t.Fatal("validated original bytes changed")
			}
		})
	}
}

func TestValidateEnforcesByteLimit(t *testing.T) {
	data := encodeTestImage(t, "png", 3, 2)
	if _, err := Validate(bytes.NewReader(data), int64(len(data)-1)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("one byte over the configured limit: got %v, want ErrTooLarge", err)
	}
	if _, err := Validate(bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("exactly at the configured limit: %v", err)
	}
	oversized := bytes.Repeat([]byte{'x'}, int(DefaultMaxBytes)+1)
	if _, err := Validate(bytes.NewReader(oversized), 0); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("one byte over the default limit: got %v, want ErrTooLarge", err)
	}
}

func TestValidateEnforcesDimensionsBeforeDecodingPixels(t *testing.T) {
	base := encodeTestImage(t, "png", 1, 1)
	for _, tt := range []struct {
		name          string
		width, height int
	}{
		{"edge over 4096", 4097, 1},
		{"more than 16 million pixels", 4001, 4000},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := pngWithDeclaredDimensions(base, tt.width, tt.height)
			if _, err := Validate(bytes.NewReader(data), DefaultMaxBytes); !errors.Is(err, ErrDimensions) {
				t.Fatalf("%d x %d: got %v, want ErrDimensions", tt.width, tt.height, err)
			}
		})
	}
	data := encodeTestImage(t, "png", MaxDimension, 1)
	if _, err := Validate(bytes.NewReader(data), DefaultMaxBytes); err != nil {
		t.Fatalf("image exactly at the edge limit: %v", err)
	}
}

func TestValidateRejectsUnsupportedAndIncompleteImages(t *testing.T) {
	var gifData bytes.Buffer
	if err := gif.Encode(&gifData, image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.Black}), nil); err != nil {
		t.Fatal(err)
	}
	pngData := encodeTestImage(t, "png", 3, 2)
	jpegData := encodeTestImage(t, "jpeg", 3, 2)
	webpData := fixture(t, "blue-purple-pink.lossy.webp")
	for _, tt := range []struct {
		name string
		data []byte
		want error
	}{
		{"empty", nil, ErrInvalidContent},
		{"SVG", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), ErrInvalidContent},
		{"GIF", gifData.Bytes(), ErrUnsupported},
		{"truncated PNG", pngData[:len(pngData)-8], ErrInvalidContent},
		{"truncated JPEG", jpegData[:len(jpegData)-2], ErrInvalidContent},
		{"truncated WebP", webpData[:len(webpData)-8], ErrInvalidContent},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Validate(bytes.NewReader(tt.data), DefaultMaxBytes); !errors.Is(err, tt.want) {
				t.Fatalf("Validate: got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestValidateRejectsAnimatedAndMismatchedWebP(t *testing.T) {
	base := fixture(t, "yellow_rose.lossy-with-alpha.webp")
	animated := bytes.Clone(base)
	animated[20] |= 0x02 // VP8X animation flag.
	mismatchedCanvas := bytes.Clone(base)
	mismatchedCanvas[24]++ // VP8X canvas width differs from the VP8 frame.
	invalidRIFFLength := bytes.Clone(base)
	binary.LittleEndian.PutUint32(invalidRIFFLength[4:8], uint32(len(base)))
	for _, tt := range []struct {
		name string
		data []byte
		want error
	}{
		{"animation flag", animated, ErrUnsupported},
		{"canvas/frame mismatch", mismatchedCanvas, ErrInvalidContent},
		{"RIFF claims missing bytes", invalidRIFFLength, ErrInvalidContent},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Validate(bytes.NewReader(tt.data), DefaultMaxBytes); !errors.Is(err, tt.want) {
				t.Fatalf("Validate: got %v, want %v", err, tt.want)
			}
		})
	}
}

// TestWebPDecoderBoundsExcessiveHuffmanGroups must run under an external memory
// and time limit. The old x/image decoder could allocate about 170 MiB from
// this 164 KiB upload, so the normal test suite deliberately skips it.
func TestWebPDecoderBoundsExcessiveHuffmanGroups(t *testing.T) {
	if os.Getenv("ATHENA_AVATAR_BOUNDED_WEBP_TEST") != "1" {
		t.Skip("run under a memory- and time-limited container")
	}
	compressed := fixture(t, "large-huffman-index.lossless.webp.bz2")
	data, err := io.ReadAll(io.LimitReader(bzip2.NewReader(bytes.NewReader(compressed)), DefaultMaxBytes+1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Validate(bytes.NewReader(data), DefaultMaxBytes); !errors.Is(err, ErrInvalidContent) {
		t.Fatalf("resource-intensive VP8L upload: got %v, want ErrInvalidContent", err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err = webp.Decode(bytes.NewReader(data))
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("WebP decoder allocated %d bytes", allocated)
	if allocated > 32<<20 {
		t.Fatalf("resource-intensive VP8L upload allocated %d bytes, want at most 32 MiB", allocated)
	}
	if err == nil {
		t.Fatal("WebP decoder accepted an excessive Huffman-tree index")
	}
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func encodeTestImage(t *testing.T, format string, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 20, G: 80, B: 140, A: 255})
		}
	}
	var out bytes.Buffer
	var err error
	switch format {
	case "png":
		err = png.Encode(&out, img)
	case "jpeg":
		err = jpeg.Encode(&out, img, nil)
	default:
		t.Fatalf("unsupported test format %q", format)
	}
	if err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func pngWithDeclaredDimensions(data []byte, width, height int) []byte {
	patched := bytes.Clone(data)
	binary.BigEndian.PutUint32(patched[16:20], uint32(width))
	binary.BigEndian.PutUint32(patched[20:24], uint32(height))
	binary.BigEndian.PutUint32(patched[29:33], crc32.ChecksumIEEE(patched[12:29]))
	return patched
}
