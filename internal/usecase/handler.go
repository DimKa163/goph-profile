package usecase

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/entity/events/v1"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/shared"
	"golang.org/x/sync/errgroup"
)

type (
	Avatar struct {
		Name     string
		Size     int64
		MimeType string
		Reader   io.ReadSeeker
	}
)

type InboxHandler func(context.Context, string, []byte) error

func NewUploadHandler(repo entity.AvatarRepository, s3 entity.S3, codec entity.ImageCodec) InboxHandler {
	return func(ctx context.Context, key string, content []byte) error {
		var ev events.AvatarUploadedEvent
		if err := ev.Read(content); err != nil {
			return err
		}
		avatarID, err := entity.ParseAvatarID(ev.AvatarID)
		if err != nil {
			return err
		}
		meta, err := repo.Find(ctx, avatarID)
		if err != nil {
			if errors.Is(err, infra.ErrNoRows) {
				return entity.Error(entity.NotFoundEntityErrorCode, "data not found")
			}
			return err
		}
		if len(meta.Images) == 0 {
			return entity.Error(entity.NotFoundEntityErrorCode, "no images found")
		}
		img := meta.Images[0]
		buffer, err := s3.Download(ctx, img.S3Key)
		if err != nil {
			return err
		}

		meta.Images = make([]*entity.Image, 9)
		src, _, err := codec.Decode(bytes.NewBuffer(buffer))
		if err != nil {
			return err
		}
		errGroup, errCtx := errgroup.WithContext(ctx)
		for idx, format := range shared.Formats {
			errGroup.Go(func() error {
				base := idx * 3
				if img.Format == format {
					meta.Images[base] = img
				} else {
					original, err := convertToFormat(errCtx, codec, s3, meta, src, format, entity.OriginalSize)
					if err != nil {
						return err
					}
					meta.Images[base] = original
				}
				x300, err := convertToFormat(errCtx, codec, s3, meta, src, format, entity.S300x300Size)
				if err != nil {
					return err
				}
				meta.Images[base+1] = x300
				x100, err := convertToFormat(errCtx, codec, s3, meta, src, format, entity.S100x100Size)
				if err != nil {
					return err
				}
				meta.Images[base+2] = x100
				return nil
			})
		}
		if err = errGroup.Wait(); err != nil {
			return err
		}
		if err = repo.InsertImage(ctx, meta); err != nil {
			return err
		}
		return repo.ActivateOnlyThis(ctx, meta.UserID, meta.ID)
	}
}

func convertToFormat(ctx context.Context, codec entity.ImageCodec, s3 entity.S3, a *entity.Avatar, src image.Image, format string, size entity.Size) (*entity.Image, error) {
	var buf []byte
	var err error
	var tag *string
	mimeType, _ := shared.ContentType(format)

	switch size {
	case entity.S300x300Size:
		src = codec.Thumbnail(src, 300, 300)
	case entity.S100x100Size:
		src = codec.Thumbnail(src, 100, 100)
	}
	buf, err = codec.Encode(src, format, 85)
	if err != nil {
		return nil, err
	}
	e := &entity.Image{
		Format:   format,
		Size:     size,
		FileSize: int64(len(buf)),
		MimeType: mimeType,
		S3Key:    fmt.Sprintf("%s/%s/%s_%s.%s", a.UserID.String(), a.ID.String(), size, a.Name, format),
	}
	tag, err = s3.Upload(ctx, e.S3Key, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	e.ETag = *tag
	return e, nil
}

func NewDeleteHandler(avatarRepo entity.AvatarRepository, s3 entity.S3) InboxHandler {
	return func(ctx context.Context, key string, bytes []byte) error {
		var ev events.AvatarDeleted
		if err := ev.Read(bytes); err != nil {
			return err
		}
		id, err := entity.ParseAvatarID(ev.AvatarID)
		if err != nil {
			return err
		}
		errGroup, errCtx := errgroup.WithContext(ctx)
		for _, k := range ev.S3Key {
			errGroup.Go(func() error {
				return s3.Delete(errCtx, k)
			})
		}
		if err = errGroup.Wait(); err != nil {
			return err
		}
		if err = avatarRepo.DeleteImages(ctx, id); err != nil {
			if errors.Is(err, infra.ErrNoRows) {
				return entity.Error(entity.NotFoundEntityErrorCode, "data not found")
			}
			return err
		}
		return nil
	}
}
