package entity

import (
	"image"
	"io"
)

//go:generate mockgen -source=codec.go -destination=mocks/mock_codec.go -package=mocks
type ImageCodec interface {
	Decode(r io.Reader) (image.Image, string, error)
	DecodeConfig(data []byte) (image.Config, error)
	Encode(src image.Image, format string, quality int) ([]byte, error)
	Thumbnail(src image.Image, h, w int) image.Image
}
