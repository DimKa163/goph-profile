package services

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"

	_ "golang.org/x/image/webp"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/beevik/guid"
)

const maxAvatarSize = 10 * 1024 * 1024

//go:generate mockgen -source=avatar.go -destination=mocks/repo_mock.go -package=mocks
type AvatarRepository interface {
	Find(ctx context.Context, id guid.Guid) (*entity.Avatar, error)
	Insert(ctx context.Context, name, mimeType string, size int64, width, height int, userID guid.Guid) (*entity.Avatar, error)
	Update(ctx context.Context, e *entity.Avatar) (*entity.Avatar, error)
}

//go:generate mockgen -source=avatar.go -destination=mocks/s3_mock.go -package=mocks
type S3 interface {
	Upload(ctx context.Context, key string, reader io.ReadSeeker) error
	Download(ctx context.Context, key string) ([]byte, error)
}

//go:generate mockgen -source=avatar.go -destination=mocks/producer_mock.go -package=mocks
type Producer interface {
	Write(ctx context.Context, key []byte, value []byte, headers ...kafka.Header) error
}

//go:generate mockgen -source=avatar.go -destination=mocks/codec_mock.go -package=mocks
type ImageCodec interface {
	Decode(r io.Reader) (image.Image, string, error)
	DecodeConfig(r io.ReadSeeker) (image.Config, error)
	Encode(src image.Image, format string, quality int) ([]byte, error)
	Thumbnail(src image.Image, h, w int) image.Image
}
type (
	UploadCommand struct {
		FileName string
		UserID   guid.Guid
		Size     int64
		MimeType string
		Reader   io.ReadSeeker
	}
	ProcessCommand struct {
		Avatar guid.Guid
		Key    string
		UserID guid.Guid
	}
)
type AvatarService struct {
	repo  AvatarRepository
	s3    S3
	codec ImageCodec
	p     Producer
}

func NewAvatarService(repo AvatarRepository, s3 S3, codec ImageCodec, producer Producer) *AvatarService {
	return &AvatarService{
		repo:  repo,
		s3:    s3,
		codec: codec,
		p:     producer,
	}
}

func (s *AvatarService) Metadata(ctx context.Context, id guid.Guid) (*entity.Avatar, error) {
	e, err := s.repo.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *AvatarService) Upload(ctx context.Context, uc *UploadCommand) (*entity.Avatar, error) {
	if err := validateAvatarSize(uc.Size); err != nil {
		return nil, err
	}
	if err := validateContentType(uc.MimeType); err != nil {
		return nil, err
	}

	cfg, err := s.codec.DecodeConfig(uc.Reader)
	if err != nil {
		return nil, err
	}

	e, err := s.repo.Insert(ctx, uc.FileName, uc.MimeType, uc.Size, cfg.Width, cfg.Height, uc.UserID)
	if err != nil {
		return nil, err
	}

	s3Key := fmt.Sprintf("avatars/avatar_%s_%s", e.ID.String(), uc.FileName)
	if err = s.s3.Upload(ctx, s3Key, uc.Reader); err != nil {
		return nil, err
	}

	e.S3Key = s3Key
	e.UploadStatus = "uploaded"
	e.ProcessingStatus = "processing"

	e, err = s.repo.Update(ctx, e)
	if err != nil {
		return nil, err
	}

	ev := &entity.AvatarUploadedEvent{
		AvatarID: e.ID.String(),
		UserID:   uc.UserID.String(),
		S3Key:    s3Key,
	}
	value, err := ev.Bytes()
	if err != nil {
		return nil, err
	}
	if err = s.p.Write(ctx, []byte(e.ID.String()), value, kafka.Header{
		Key:   kafka.EventTypeHeaderKey,
		Value: ev.String(),
	}); err != nil {
		return nil, err
	}

	return e, nil
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
