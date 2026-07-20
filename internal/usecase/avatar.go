package usecase

import (
	"context"
	"errors"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
	"slices"

	"github.com/DimKa163/goph-profile/internal/entity/events/v1"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/logging"
	"go.uber.org/zap"
	_ "golang.org/x/image/webp"

	"github.com/DimKa163/goph-profile/internal/entity"
)

const maxAvatarSize = 10 * 1024 * 1024

var ErrAvatarNotModified = errors.New("avatar not modified")

type Transactor interface {
	WithTx(ctx context.Context, fn func(context.Context) error) error
}
type (
	UploadCommand struct {
		FileName string
		UserID   entity.Email
		Size     int64
		MimeType string
		Buf      []byte
	}
	Request struct {
		ID     entity.AvatarID
		Format string
		Size   entity.Size
	}
)
type AvatarService struct {
	tx       Transactor
	repo     entity.AvatarRepository
	taskRepo entity.TaskRepository
	s3       entity.S3
	codec    entity.ImageCodec
}

func NewAvatarService(tx Transactor, repo entity.AvatarRepository, taskRepo entity.TaskRepository, s3 entity.S3, codec entity.ImageCodec) *AvatarService {
	return &AvatarService{
		tx:       tx,
		repo:     repo,
		taskRepo: taskRepo,
		s3:       s3,
		codec:    codec,
	}
}

func (s *AvatarService) Metadata(ctx context.Context, id entity.AvatarID) (*entity.Avatar, error) {
	e, err := s.repo.Find(ctx, id)
	if err != nil {
		if errors.Is(err, infra.ErrNoRows) {
			return nil, entity.WrapError(entity.NotFoundEntityErrorCode, id, err)
		}
		return nil, entity.WrapError(entity.InternalErrorCode, id, err)
	}
	return e, nil
}

func (s *AvatarService) Get(ctx context.Context, eTag string, req *Request) (*entity.Image, []byte, error) {
	var err error
	var avatar *entity.Avatar
	var src []byte
	if req.Size == "" {
		req.Size = entity.S300x300Size
	}

	if req.Format == "" {
		req.Format = "webp"
	}
	avatar, err = s.repo.Find(ctx, req.ID)
	if err != nil {
		if errors.Is(err, infra.ErrNoRows) {
			return nil, nil, entity.WrapError(entity.NotFoundEntityErrorCode, req.ID, err)
		}
		return nil, nil, entity.WrapError(entity.InternalErrorCode, req.ID, err)
	}
	idx := slices.IndexFunc(avatar.Images, func(e *entity.Image) bool {
		if req.Size != "" && e.Size != req.Size {
			return false
		}

		if req.Format != "" && e.Format != req.Format {
			return false
		}

		return true
	})
	if idx == -1 {
		return nil, nil, entity.WrapError(entity.NotFoundEntityErrorCode, req.ID.String(), nil)
	}
	e := avatar.Images[idx]
	if eTag == e.ETag {
		return nil, nil, ErrAvatarNotModified
	}
	src, err = s.s3.Download(ctx, avatar.UserID, e.S3Key)
	if err != nil {
		return nil, nil, entity.WrapError(entity.InternalErrorCode, req.ID, err)
	}
	return e, src, nil
}

func (s *AvatarService) Upload(ctx context.Context, uc *UploadCommand) (*entity.Avatar, error) {
	if err := validateAvatarSize(uc.Size); err != nil {
		return nil, err
	}
	format, err := formatByMimeType(uc.MimeType)
	if err != nil {
		return nil, err
	}

	cfg, err := s.codec.DecodeConfig(uc.Buf)
	if err != nil {
		return nil, err
	}

	entityID := entity.NewAvatarID()
	fileName := baseName(uc.FileName)
	s3Key := fmt.Sprintf("%s/%s_%s", entityID.String(), entity.OriginalSize, uc.FileName)
	tag, err := s.s3.Upload(ctx, uc.UserID, s3Key, uc.Buf)
	if err != nil {
		return nil, entity.WrapError(entity.InternalErrorCode, "", err)
	}
	defer func() {
		if err != nil {
			logger := logging.Logger(ctx)
			logger.Debug(" error occurred during transaction. trying to clean up storage")
			if err = s.s3.Delete(ctx, uc.UserID, s3Key); err != nil {
				logger.Error("error occurred during cleaning storage", zap.Error(err))
			}
		}
	}()

	var e *entity.Avatar
	if err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		e, err = s.repo.Insert(
			ctx,
			entityID,
			fileName,
			uc.UserID,
			cfg.Width,
			cfg.Height,
			uc.Size,
			uc.MimeType,
			&entity.Image{
				Format:   format,
				FileSize: uc.Size,
				Size:     entity.OriginalSize,
				MimeType: uc.MimeType,
				S3Key:    s3Key,
				ETag:     *tag,
			})
		if err != nil {
			return err
		}
		ev := &events.AvatarUploadedEvent{
			AvatarID: e.ID.String(),
		}
		b, err := ev.Bytes()
		if err != nil {
			return err
		}
		return s.taskRepo.Insert(ctx, e.ID.String(), entity.AvatarUploaded, b)
	}); err != nil {
		return nil, err
	}

	return e, nil
}

func (s *AvatarService) Delete(ctx context.Context, id entity.AvatarID, userID entity.Email) error {
	meta, err := s.Metadata(ctx, id)
	if err != nil {
		return err
	}

	if meta.UserID != userID {
		return entity.WrapError(entity.PermissionDeniedErrorCode, meta.ID.String(), nil)
	}

	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err = s.repo.Delete(ctx, id); err != nil {
			return err
		}
		keys := make([]string, len(meta.Images))
		idx := 0
		for _, img := range meta.Images {
			keys[idx] = img.S3Key
			idx++
		}
		ev := &events.AvatarDeleted{
			AvatarID: id.String(),
			S3Key:    keys,
			UserID:   meta.UserID.String(),
		}
		b, err := ev.Bytes()
		if err != nil {
			return err
		}
		return s.taskRepo.Insert(ctx, id.String(), entity.AvatarDeleted, b)
	})
}

func validateAvatarSize(size int64) error {
	if size > maxAvatarSize || size <= 0 {
		return entity.WrapError(entity.InvalidSizeErrorCode, size, nil)
	}
	return nil
}

func formatByMimeType(mimeType string) (string, error) {
	switch mimeType {
	case "image/jpeg":
		return "jpeg", nil
	case "image/png":
		return "png", nil
	case "image/webp":
		return "webp", nil
	default:
		return "", entity.WrapError(entity.InvalidContentTypeErrorCode, mimeType, nil)
	}
}

func baseName(filename string) string {
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]
	return base
}
