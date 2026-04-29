package attachments

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"

	_ "image/gif"  // decoder
	_ "image/jpeg" // decoder
	_ "image/png"  // decoder

	"golang.org/x/image/draw"
)

// ErrNotImage is returned when Thumb is called on bytes that don't decode
// as one of the registered image formats.
var ErrNotImage = errors.New("not a decodable image")

// Thumb decodes src as an image, scales it so the longest edge is maxEdge,
// and JPEG-encodes the result at quality 80. Aspect ratio preserved.
func Thumb(src io.Reader, maxEdge int) (io.Reader, error) {
	img, _, err := image.Decode(src)
	if err != nil {
		return nil, ErrNotImage
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxEdge && h <= maxEdge {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}); err != nil {
			return nil, err
		}
		return &buf, nil
	}
	var nw, nh int
	if w >= h {
		nw = maxEdge
		nh = int(float64(h) * float64(maxEdge) / float64(w))
	} else {
		nh = maxEdge
		nw = int(float64(w) * float64(maxEdge) / float64(h))
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80}); err != nil {
		return nil, err
	}
	return &buf, nil
}
