package entity

import (
	"image"
	"io"
)

// ImageCodec read/transform images
//
//go:generate mockgen -source=codec.go -destination=mocks/mock_codec.go -package=mocks
type ImageCodec interface {
	// Decode decodes image bytes.
	Decode(r io.Reader) (image.Image, string, error)
	// DecodeConfig reads image metadata without decoding the whole image.
	DecodeConfig(data []byte) (image.Config, error)
	// Encode writes an image in the requested format.
	Encode(src image.Image, format string, quality int) ([]byte, error)
	// Thumbnail describes a generated avatar thumbnail.
	Thumbnail(src image.Image, h, w int) image.Image
}
