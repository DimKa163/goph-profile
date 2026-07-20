package usecase

import (
	"context"
	"errors"
	"slices"

	"github.com/DimKa163/goph-profile/internal/assets"
	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/entity/events/v1"
	"github.com/DimKa163/goph-profile/internal/infra"
)

type UserService struct {
	transactor Transactor
	repo       entity.AvatarRepository
	taskRepo   entity.TaskRepository
	s3         entity.S3
}
type (
	GetAllResponse struct {
		Name    string
		Format  string
		Size    entity.Size
		Content []byte
	}
)

func NewUserService(transactor Transactor, repo entity.AvatarRepository, taskRepo entity.TaskRepository, s3 entity.S3) *UserService {
	return &UserService{transactor: transactor,
		repo: repo, taskRepo: taskRepo, s3: s3}
}

func (s *UserService) Get(ctx context.Context, tag string, userID entity.Email, request *Request) (*entity.Image, []byte, error) {
	if request.Size == "" {
		request.Size = entity.S300x300Size
	}

	if request.Format == "" {
		request.Format = "webp"
	}

	e, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, infra.ErrNoRows) {
			return nil, nil, entity.WrapError(entity.NotFoundEntityErrorCode, "by user_id", err)
		}
		return nil, nil, err
	}
	idx := slices.IndexFunc(e.Images, func(image *entity.Image) bool {
		return image.Format == request.Format && image.Size == request.Size
	})
	if idx == -1 {
		return nil, nil, entity.WrapError(entity.NotFoundEntityErrorCode, "not found source", err)
	}
	image := e.Images[idx]

	if image.ETag == tag {
		return nil, nil, ErrAvatarNotModified
	}

	buf, err := s.s3.Download(ctx, userID, image.S3Key)
	if err != nil {
		return nil, nil, err
	}

	return image, buf, nil
}

func (s *UserService) GetDefault(eTag string, request *Request) (*entity.Image, []byte, error) {
	if request.Size == "" {
		request.Size = entity.S300x300Size
	}

	if request.Format == "" {
		request.Format = "webp"
	}
	idx := slices.IndexFunc(assets.DefaultAvatars, func(avatar *assets.DefaultAvatar) bool {
		return avatar.Format == request.Format && avatar.Size == request.Size.String()
	})
	if idx == -1 {
		return nil, nil, entity.WrapError(entity.NotFoundEntityErrorCode, "not found source", nil)
	}
	av := assets.DefaultAvatars[idx]
	if eTag == av.ETag {
		return nil, nil, ErrAvatarNotModified
	}
	return &entity.Image{
		Format:   av.Format,
		Size:     entity.Size(av.Size),
		MimeType: av.MimeType,
		ETag:     av.ETag,
	}, av.Data, nil
}

func (s *UserService) ListByUserID(ctx context.Context, userID entity.Email) ([]*entity.Avatar, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *UserService) Delete(ctx context.Context, userID entity.Email) error {
	e, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, infra.ErrNoRows) {
			return entity.WrapError(entity.NotFoundEntityErrorCode, "by user_id", err)
		}
		return err
	}
	return s.transactor.WithTx(ctx, func(ctx context.Context) error {
		err = s.repo.Delete(ctx, e.ID)
		if err != nil {
			if errors.Is(err, infra.ErrNoRows) {
				return entity.WrapError(entity.NotFoundEntityErrorCode, "by user_id", err)
			}
			return err
		}
		keys := make([]string, len(e.Images))
		for i, k := range e.Images {
			keys[i] = k.S3Key
		}
		ev := events.AvatarDeleted{
			AvatarID: e.ID.String(),
			S3Key:    keys,
		}
		j, _ := ev.Bytes()
		return s.taskRepo.Insert(ctx, e.ID.String(), entity.AvatarDeleted, j)
	})
}
