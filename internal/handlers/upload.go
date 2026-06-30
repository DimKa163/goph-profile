package handlers

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/beevik/guid"
)

const maxAvatarSize = 10 * 1024 * 1024

type (
	Avatar struct {
		Name     string
		Size     int64
		MimeType string
		Reader   io.ReadSeeker
	}
	Uploader      func(ctx context.Context, avatar *Avatar, userId guid.Guid) (*UploaderState, error)
	UploaderState struct {
		ID        string    `json:"id"`
		UserID    string    `json:"userId"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"createdAt"`
	}
)

//go:generate mockgen -source=upload.go -destination=mocks/upload_mock.go -package=mocks
type S3Uploader interface {
	Upload(ctx context.Context, key string, reader io.ReadSeeker) error
}

//go:generate mockgen -source=upload.go -destination=mocks/upload_mock.go -package=mocks
type AvatarInsertUpdaterRepository interface {
	Insert(ctx context.Context, name, mimeType string, size int64, width, height int, userID guid.Guid) (*entity.Avatar, error)
	Update(ctx context.Context, e *entity.Avatar) (*entity.Avatar, error)
}

//go:generate mockgen -source=upload.go -destination=mocks/upload_mock.go -package=mocks
type Decoder interface {
	DecodeConfig(r io.ReadSeeker) (image.Config, error)
}

//go:generate mockgen -source=upload.go -destination=mocks/upload_mock.go -package=mocks
type Producer interface {
	Write(ctx context.Context, key []byte, value []byte, headers ...kafka.Header) error
}

func NewUploader(uploader S3Uploader, repository AvatarInsertUpdaterRepository, imgDecoder Decoder, producer Producer) Uploader {
	return func(ctx context.Context, avatar *Avatar, userID guid.Guid) (*UploaderState, error) {
		if err := validateAvatarSize(avatar.Size); err != nil {
			return nil, err
		}
		if err := validateContentType(avatar.MimeType); err != nil {
			return nil, err
		}
		cfg, err := imgDecoder.DecodeConfig(avatar.Reader)
		if err != nil {
			return nil, err
		}

		e, err := repository.Insert(ctx, avatar.Name, avatar.MimeType, avatar.Size, cfg.Width, cfg.Height, userID)
		if err != nil {
			return nil, err
		}
		s3Key := fmt.Sprintf("avatar_%s_%s", e.ID.String(), avatar.Name)
		if err = uploader.Upload(ctx, s3Key, avatar.Reader); err != nil {
			return nil, err
		}

		e.S3Key = s3Key
		e.UploadStatus = "uploaded"
		e.ProcessingStatus = "processing"

		e, err = repository.Update(ctx, e)
		if err != nil {
			return nil, err
		}
		ev := &entity.AvatarUploadEvent{
			AvatarID: e.ID.String(),
			UserID:   userID.String(),
			S3Key:    s3Key,
		}
		value, err := ev.Bytes()
		if err != nil {
			return nil, err
		}
		if err = producer.Write(ctx, []byte(e.ID.String()), value, kafka.Header{
			Key:   "event-type",
			Value: []byte("avatar-uploaded"),
		}); err != nil {
			return nil, err
		}
		return &UploaderState{
			ID:        e.ID.String(),
			UserID:    userID.String(),
			Status:    e.ProcessingStatus,
			CreatedAt: e.CreatedAt,
		}, nil
	}
}
func validateAvatarSize(size int64) error {
	if size > maxAvatarSize {
		return entity.ErrTooBigSizeErrorMessage
	}
	return nil
}
func validateContentType(contentType string) error {
	switch contentType {
	case "image/jpeg",
		"image/png",
		"image/gif",
		"image/webp":
		return nil
	default:
		return fmt.Errorf("%w: %s", entity.ErrInvalidContentErrorMessage, contentType)
	}
}
