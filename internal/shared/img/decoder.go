// Package img provides image decoding, encoding, and thumbnail helpers.
package img

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"

	"github.com/chai2010/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// ErrUnsupportedImageFormat is returned for unsupported image formats.
var ErrUnsupportedImageFormat = errors.New("unsupported image format")

type imageCodec struct{}

// NewCodec creates an image codec.
func NewCodec() *imageCodec {
	return &imageCodec{}
}

// DecodeConfig reads image metadata without decoding the whole image.
func (codec *imageCodec) DecodeConfig(data []byte) (image.Config, error) {
	var cfg image.Config
	var buf bytes.Buffer
	buf.Write(data)
	cfg, _, err := image.DecodeConfig(&buf)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Decode decodes image bytes.
func (codec *imageCodec) Decode(r io.Reader) (image.Image, string, error) {
	img, format, err := image.Decode(r)
	if err != nil {
		return nil, "", err
	}
	return img, format, nil
}

// Encode writes an image in the requested format.
func (codec *imageCodec) Encode(src image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer
	var err error
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(&buf, src, &jpeg.Options{
			Quality: quality,
		})
	case "png":
		err = png.Encode(&buf, src)
	case "webp":
		err = webp.Encode(&buf, src, &webp.Options{
			Quality: float32(quality),
		})
	default:
		err = ErrUnsupportedImageFormat
	}
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Thumbnail describes a generated avatar thumbnail.
func (codec *imageCodec) Thumbnail(src image.Image, h, w int) image.Image {
	bounds := src.Bounds()

	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	ratioX := float64(w) / float64(srcWidth)
	ratioY := float64(h) / float64(srcHeight)
	ratio := ratioX
	if ratioY < ratioX {
		ratio = ratioY
	}

	newWidth := int(float64(srcWidth) * ratio)
	newHeight := int(float64(srcHeight) * ratio)

	if newWidth <= 0 {
		newWidth = 1
	}

	if newHeight <= 0 {
		newHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	draw.CatmullRom.Scale(
		dst,
		dst.Bounds(),
		src,
		src.Bounds(),
		draw.Over,
		nil,
	)

	return dst
}
