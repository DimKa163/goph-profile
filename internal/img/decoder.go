package img

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	_ "golang.org/x/image/webp"
)

type ImageDecoder struct{}

func NewDecoder() *ImageDecoder {
	return &ImageDecoder{}
}

func (decoder *ImageDecoder) DecodeConfig(r io.ReadSeeker) (image.Config, error) {
	var cfg image.Config
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return cfg, err
	}
	_, _ = r.Seek(0, io.SeekStart)
	return cfg, nil
}
