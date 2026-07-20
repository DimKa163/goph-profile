package usecase

import (
	"context"
	"slices"
	"testing"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/entity/mocks"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestGetUserAvatarShouldReturnX300WebInDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)

	userID := entity.Email("user@user.com")
	tag := "sta"
	e := createMetadata(userID)
	repo.EXPECT().FindByUserID(ctx, userID).Return(e, nil)
	image := findImage(e.Images, func(i *entity.Image) bool {
		return i.Size == entity.S300x300Size && i.Format == "webp"
	})
	if image == nil {
		t.Error("image should not be nil")
		return
	}
	buffer := []byte("webp in S300x300")
	s3.EXPECT().Download(ctx, userID, image.S3Key).Return(buffer, nil)

	sut := NewUserService(newTransactor(), repo, taskRepo, s3)

	m, buf, err := sut.Get(ctx, tag, userID, &Request{})

	require.NoError(t, err)
	require.Equal(t, buffer, buf)
	require.Equal(t, m.Size, entity.S300x300Size)
	require.Equal(t, m.Format, "webp")
}

func TestGetUserAvatarShouldReturnCorrectImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)

	reqs := []*Request{
		{
			Format: "webp",
			Size:   "original",
		},
		{
			Format: "webp",
			Size:   "100x100",
		},
		{
			Format: "png",
			Size:   "original",
		},
		{
			Format: "png",
			Size:   "100x100",
		},
	}

	for _, req := range reqs {
		userID := entity.Email("user@user.com")
		tag := "sta"
		e := createMetadata(userID)
		repo.EXPECT().FindByUserID(ctx, userID).Return(e, nil)
		image := findImage(e.Images, func(i *entity.Image) bool {
			return i.Size == req.Size && i.Format == req.Format
		})
		if image == nil {
			t.Error("image should not be nil")
			return
		}
		buffer := []byte("some image")
		s3.EXPECT().Download(ctx, userID, image.S3Key).Return(buffer, nil)

		sut := NewUserService(newTransactor(), repo, taskRepo, s3)

		m, buf, err := sut.Get(ctx, tag, userID, req)

		require.NoError(t, err)
		require.Equal(t, buffer, buf)
		require.Equal(t, m.Size, req.Size)
		require.Equal(t, m.Format, req.Format)
	}
}

func TestGetUserAvatarShouldReturnErrorWhenTagTheSame(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)

	userID := entity.Email("user@user.com")
	tag := "sta"
	e := createMetadata(userID)
	repo.EXPECT().FindByUserID(ctx, userID).Return(e, nil)
	image := findImage(e.Images, func(i *entity.Image) bool {
		return i.Size == entity.S300x300Size && i.Format == "webp"
	})
	if image == nil {
		t.Error("image should not be nil")
		return
	}
	image.ETag = tag
	buffer := []byte("webp in S300x300")
	s3.EXPECT().Download(ctx, userID, image.S3Key).Return(buffer, nil).Times(0)

	sut := NewUserService(newTransactor(), repo, taskRepo, s3)

	m, buf, err := sut.Get(ctx, tag, userID, &Request{})

	require.Error(t, err)
	require.ErrorIs(t, err, ErrAvatarNotModified)
	require.Nil(t, buf)
	require.Nil(t, m)
}

func TestGetUserAvatarShouldReturnErrorWhenMetadataNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)

	userID := entity.Email("user@user.com")
	tag := "sta"
	repo.EXPECT().FindByUserID(ctx, userID).Return(nil, infra.ErrNoRows)

	sut := NewUserService(newTransactor(), repo, taskRepo, s3)

	m, buf, err := sut.Get(ctx, tag, userID, &Request{})

	require.Error(t, err)
	require.ErrorIs(t, err, entity.ErrNotFoundEntity)
	var pe *entity.ProfileError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, pe.Code, entity.NotFoundEntityErrorCode)
	require.Nil(t, buf)
	require.Nil(t, m)
}

func TestGetUserAvatarShouldReturnErrorWhenNoImages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)

	userID := entity.Email("user@user.com")
	tag := "sta"
	e := createMetadata(userID)
	e.Images = nil
	repo.EXPECT().FindByUserID(ctx, userID).Return(e, nil)

	sut := NewUserService(newTransactor(), repo, taskRepo, s3)

	m, buf, err := sut.Get(ctx, tag, userID, &Request{})

	require.Error(t, err)
	require.ErrorIs(t, err, entity.ErrNotFoundEntity)
	var pe *entity.ProfileError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, pe.Code, entity.NotFoundEntityErrorCode)
	require.Nil(t, buf)
	require.Nil(t, m)
}

func TestDeleteUserAvatarShouldBeSuccessful(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)

	userID := entity.Email("user@user.com")
	e := createMetadata(userID)
	repo.EXPECT().FindByUserID(ctx, userID).Return(e, nil)
	repo.EXPECT().Delete(ctx, e.ID).Return(nil)
	taskRepo.EXPECT().Insert(ctx, e.ID.String(), entity.AvatarDeleted, gomock.Any()).Return(nil)

	sut := NewUserService(newTransactor(), repo, taskRepo, s3)

	err := sut.Delete(ctx, userID)

	require.NoError(t, err)
}

func createMetadata(userID entity.Email) *entity.Avatar {
	return &entity.Avatar{
		ID:     entity.NewAvatarID(),
		UserID: userID,
		Images: []*entity.Image{
			{
				Format:   "jpeg",
				Size:     entity.OriginalSize,
				S3Key:    "original-jpg-key",
				FileSize: 1234,
				ETag:     "original-jpg-key-tag",
			},
			{
				Format:   "jpeg",
				Size:     entity.S300x300Size,
				S3Key:    "s300-jpg-key",
				FileSize: 123,
				ETag:     "s300-jpg-key-tag",
			},
			{
				Format:   "jpeg",
				Size:     entity.S100x100Size,
				S3Key:    "s100-jpg-key",
				FileSize: 12,
				ETag:     "s100-jpg-key-tag",
			},
			{
				Format:   "png",
				Size:     entity.OriginalSize,
				S3Key:    "original-png-key",
				FileSize: 1234,
				ETag:     "original-png-key-tag",
			},
			{
				Format:   "png",
				Size:     entity.S300x300Size,
				S3Key:    "s300-png-key",
				FileSize: 123,
				ETag:     "s300-png-key-tag",
			},
			{
				Format:   "png",
				Size:     entity.S100x100Size,
				S3Key:    "original-jpg-key",
				FileSize: 12,
				ETag:     "s100-jpg-key-tag",
			},
			{
				Format:   "webp",
				Size:     entity.OriginalSize,
				S3Key:    "original-webp-key",
				FileSize: 1234,
				ETag:     "original-webp-key-tag",
			},
			{
				Format:   "webp",
				Size:     entity.S300x300Size,
				S3Key:    "s300-webp-key",
				FileSize: 123,
				ETag:     "s300-webp-key-tag",
			},
			{
				Format:   "webp",
				Size:     entity.S100x100Size,
				S3Key:    "s100-webp-key",
				FileSize: 12,
				ETag:     "s100-webp-key-tag",
			},
		},
	}
}

func findImage(images []*entity.Image, fn func(image *entity.Image) bool) *entity.Image {
	idx := slices.IndexFunc(images, fn)
	if idx == -1 {
		return nil
	}
	return images[idx]
}
