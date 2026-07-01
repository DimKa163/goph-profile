package handlers

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/beevik/guid"
	_ "golang.org/x/image/webp"
)

type (
	Avatar struct {
		Name     string
		Size     int64
		MimeType string
		Reader   io.ReadSeeker
	}
	UploaderHandler func(ctx context.Context, event *entity.AvatarUploadedEvent) error
)

type AvatarRepository interface {
	Find(ctx context.Context, id guid.Guid) (*entity.Avatar, error)
	Update(ctx context.Context, e *entity.Avatar) (*entity.Avatar, error)
}

type S3 interface {
	Upload(ctx context.Context, key string, reader io.ReadSeeker) error
	Download(ctx context.Context, key string) ([]byte, error)
}

type ImageCodec interface {
	Decode(r io.Reader) (image.Image, string, error)
	Encode(src image.Image, format string, quality int) ([]byte, error)
	Thumbnail(src image.Image, h, w int) image.Image
}

func NewUploadHandler(repo AvatarRepository, s3 S3, codec ImageCodec) UploaderHandler {
	return func(ctx context.Context, event *entity.AvatarUploadedEvent) error {
		avatarID, _ := guid.ParseString(event.AvatarID)
		userID, _ := guid.ParseString(event.UserID)
		meta, err := repo.Find(ctx, *avatarID)
		if err != nil {
			return err
		}

		buffer, err := s3.Download(ctx, event.S3Key)
		if err != nil {
			return err
		}

		src, format, _ := codec.Decode(bytes.NewBuffer(buffer))

		src100 := codec.Thumbnail(src, 100, 100)
		th100 := &entity.Thumbnail{
			Size: fmt.Sprintf("%dx%d", src100.Bounds().Dx(), src100.Bounds().Dy()),
			Url:  fmt.Sprintf("thumbnail/%s/av%s.%s", userID, meta.ID.String(), format),
		}
		buf100, err := codec.Encode(src100, format, 85)
		if err != nil {
			return err
		}
		if err = s3.Upload(ctx, th100.Url, bytes.NewReader(buf100)); err != nil {
			return err
		}
		meta.Thumbnails = append(meta.Thumbnails, th100)

		src300 := codec.Thumbnail(src, 300, 300)
		th300 := &entity.Thumbnail{
			Size: fmt.Sprintf("%dx%d", src300.Bounds().Dx(), src300.Bounds().Dy()),
			Url:  fmt.Sprintf("thumbnail/%s/av%s.%s", userID, meta.ID.String(), format),
		}
		buf300, err := codec.Encode(src300, format, 85)
		if err != nil {
			return err
		}
		if err = s3.Upload(ctx, th300.Url, bytes.NewReader(buf300)); err != nil {
			return err
		}
		meta.Thumbnails = append(meta.Thumbnails, th300)
		meta.ProcessingStatus = "completed"
		if _, err = repo.Update(ctx, meta); err != nil {
			return err
		}
		return nil
	}
}
