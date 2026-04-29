package attachments_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	_ "image/jpeg" // register decoder

	"github.com/and1truong/liveboard/internal/attachments"
)

func TestThumb(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1024, 512))
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	out, err := attachments.Thumb(&buf, 256)
	if err != nil {
		t.Fatalf("thumb: %v", err)
	}
	img, _, err := image.Decode(out)
	if err != nil {
		t.Fatalf("decode thumb: %v", err)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w > 256 || h > 256 {
		t.Errorf("thumb too large: %dx%d", w, h)
	}
	if w != 256 {
		t.Errorf("expected width 256 (max edge), got %d", w)
	}
}

func TestThumbNonImage(t *testing.T) {
	_, err := attachments.Thumb(bytes.NewReader([]byte("not an image")), 128)
	if err == nil {
		t.Errorf("expected error on non-image")
	}
}
