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
	// Avatar defines avatar.
	Avatar struct {
		// Name stores the name value.
		Name string
		// Size stores the size value.
		Size int64
		// MimeType stores the mime type value.
		MimeType string
		// Reader stores the reader value.
		Reader io.ReadSeeker
	}
)

// InboxHandler processes an inbox event payload.
type InboxHandler func(context.Context, []byte) error

// NewUploadHandler creates an inbox handler for avatar upload events.
func NewUploadHandler(
	repo entity.AvatarRepository,
	s3 entity.S3,
	codec entity.ImageCodec,
) InboxHandler {
	return func(ctx context.Context, content []byte) error {
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
				return entity.WrapError(entity.NotFoundEntityErrorCode, "data not found", nil)
			}
			return err
		}
		if len(meta.Images) == 0 {
			return entity.WrapError(entity.NotFoundEntityErrorCode, "no images found", nil)
		}
		img := meta.Images[0]
		buffer, err := s3.Download(ctx, meta.UserID, img.S3Key)
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
					original, err := convertToFormat(
						errCtx,
						codec,
						s3,
						meta,
						src,
						format,
						entity.OriginalSize,
					)
					if err != nil {
						return err
					}
					meta.Images[base] = original

				}
				x300, err := convertToFormat(
					errCtx,
					codec,
					s3,
					meta,
					src,
					format,
					entity.S300x300Size,
				)
				if err != nil {
					return err
				}
				meta.Images[base+1] = x300
				x100, err := convertToFormat(
					errCtx,
					codec,
					s3,
					meta,
					src,
					format,
					entity.S100x100Size,
				)
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

func convertToFormat(
	ctx context.Context,
	codec entity.ImageCodec,
	s3 entity.S3,
	a *entity.Avatar,
	src image.Image,
	format string,
	size entity.Size,
) (*entity.Image, error) {
	var buf []byte
	var err error
	var tag *string

	mimeType, err := shared.ContentType(format)
	if err != nil {
		return nil, err
	}

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
		S3Key:    fmt.Sprintf("%s/%s_%s.%s", a.ID.String(), size, a.Name, format),
	}
	tag, err = s3.Upload(ctx, a.UserID, e.S3Key, buf)
	if err != nil {
		return nil, err
	}
	e.ETag = *tag
	return e, nil
}

// NewDeleteHandler creates an inbox handler for avatar delete events.
func NewDeleteHandler(avatarRepo entity.AvatarRepository, s3 entity.S3) InboxHandler {
	return func(ctx context.Context, bytes []byte) error {
		var ev events.AvatarDeleted
		if err := ev.Read(bytes); err != nil {
			return err
		}
		id, err := entity.ParseAvatarID(ev.AvatarID)
		if err != nil {
			return err
		}
		userID, err := entity.ParseEmail(ev.UserID)
		if err != nil {
			return err
		}
		errGroup, errCtx := errgroup.WithContext(ctx)
		for _, k := range ev.S3Key {
			errGroup.Go(func() error {
				return s3.Delete(errCtx, userID, k)
			})
		}
		if err = errGroup.Wait(); err != nil {
			return err
		}
		if err = avatarRepo.DeleteImages(ctx, id); err != nil {
			if errors.Is(err, infra.ErrNoRows) {
				return entity.WrapError(entity.NotFoundEntityErrorCode, "data not found", nil)
			}
			return err
		}
		return nil
	}
}
