package usecase

import (
	"context"
	"image"
	"testing"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/entity/mocks"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestMetadataShouldBeSuccessful(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	id := entity.NewAvatarID()
	repo.EXPECT().Find(ctx, id).Return(&entity.Avatar{
		ID: id,
	}, nil)
	sut := NewAvatarService(newTransactor(), repo, taskRepo, s3, codec)

	m, err := sut.Metadata(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, m.ID)
}

func TestMetadataShouldReturnErrorWhenFindFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	id := entity.NewAvatarID()
	repo.EXPECT().Find(ctx, id).Return(nil, infra.ErrNoRows)
	sut := NewAvatarService(newTransactor(), repo, taskRepo, s3, codec)

	m, err := sut.Metadata(ctx, id)
	require.Nil(t, m)

	var pErr *entity.ProfileError
	require.ErrorAs(t, err, &pErr)
	require.Equal(t, entity.NotFoundEntityErrorCode, pErr.Code)
}

func TestGetShouldBeSuccessful(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	id := entity.NewAvatarID()
	eTag := "eTag"
	file := []byte("file mother fucker")
	repo.EXPECT().Find(ctx, id).Return(&entity.Avatar{
		Images: []*entity.Image{
			{
				Format:   "jpeg",
				S3Key:    "key",
				MimeType: "image/jpeg",
				ETag:     "tag",
				Size:     entity.S300x300Size,
			},
		},
	}, nil)
	s3.EXPECT().Download(ctx, gomock.Any(), gomock.Any()).Return(file, nil)
	sut := NewAvatarService(newTransactor(), repo, taskRepo, s3, codec)

	e, f, err := sut.Get(ctx, eTag, &Request{ID: id, Format: "jpeg", Size: entity.S300x300Size})
	require.NoError(t, err)
	require.NotNil(t, e)
	require.NotNil(t, f)
}

func TestGetShouldBeFailureWhenMetaNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	id := entity.NewAvatarID()
	eTag := "eTag"
	s3Key := "key"
	file := []byte("file mother fucker")
	repo.EXPECT().Find(ctx, id).Return(nil, infra.ErrNoRows)
	s3.EXPECT().Download(ctx, gomock.Any(), s3Key).Return(file, nil).Times(0)
	sut := NewAvatarService(newTransactor(), repo, taskRepo, s3, codec)

	e, f, err := sut.Get(ctx, eTag, &Request{ID: id})

	require.Nil(t, e)
	require.Nil(t, f)

	var pErr *entity.ProfileError
	require.ErrorAs(t, err, &pErr)
	require.Equal(t, entity.NotFoundEntityErrorCode, pErr.Code)
}

func TestGetShouldBeFailureWhenImageNotChanged(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	id := entity.NewAvatarID()
	eTag := "eTag"
	s3Key := "key"
	file := []byte("file mother fucker")
	repo.EXPECT().Find(ctx, id).Return(&entity.Avatar{
		Images: []*entity.Image{
			{
				ETag:   eTag,
				Format: "jpeg",
				Size:   entity.S300x300Size,
			},
		},
	}, nil)
	s3.EXPECT().Download(ctx, gomock.Any(), s3Key).Return(file, nil).Times(0)
	sut := NewAvatarService(newTransactor(), repo, taskRepo, s3, codec)

	e, f, err := sut.Get(ctx, eTag, &Request{ID: id, Format: "jpeg", Size: entity.S300x300Size})

	require.Nil(t, e)
	require.Nil(t, f)

	require.ErrorIs(t, err, ErrAvatarNotModified)
}

func TestUploadShouldHandleSpecificSize(t *testing.T) {
	cases := []struct {
		Name          string
		UploadCommand *UploadCommand
		ErrExpected   bool
		ExpectedError error
	}{
		{
			Name: "Valid size",
			UploadCommand: &UploadCommand{
				Size:     1024 * 1024 * 9,
				UserID:   entity.Email("user@user.com"),
				FileName: "test.png",
				MimeType: "image/png",
				Buf:      []byte("test"),
			},
			ErrExpected:   false,
			ExpectedError: nil,
		},
		{
			Name: "Invalid size",
			UploadCommand: &UploadCommand{
				Size:     maxAvatarSize + 1,
				UserID:   entity.Email("user@user.com"),
				FileName: "test.png",
				MimeType: "image/png",
				Buf:      []byte("test"),
			},
			ErrExpected:   true,
			ExpectedError: entity.ErrInvalidSize,
		},
		{
			Name: "Negative size",
			UploadCommand: &UploadCommand{
				Size:     -1024 * 1024 * 9,
				UserID:   entity.Email("user@user.com"),
				FileName: "test.png",
				MimeType: "image/png",
				Buf:      []byte("test"),
			},
			ErrExpected:   true,
			ExpectedError: entity.ErrInvalidSize,
		},
		{
			Name: "Zero size",
			UploadCommand: &UploadCommand{
				Size:     0,
				UserID:   entity.Email("user@user.com"),
				FileName: "test.png",
				MimeType: "image/png",
				Buf:      []byte("test"),
			},
			ErrExpected:   true,
			ExpectedError: entity.ErrInvalidSize,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()
			sut := createSut(ctx, ctrl, tc.UploadCommand, !tc.ErrExpected)
			e, err := sut.Upload(ctx, tc.UploadCommand)
			if tc.ErrExpected {
				require.Error(t, err)
				require.ErrorIs(t, err, entity.ErrInvalidSize)
				require.Nil(t, e)
			} else {
				require.NoError(t, err)
				require.NotNil(t, e)
			}
		})
	}
}

func TestUploadShouldHandleSpecificContentType(t *testing.T) {
	cases := []struct {
		Name          string
		UploadCommand *UploadCommand
		ErrExpected   bool
		ExpectedError error
	}{
		{
			Name: "image/jpeg",
			UploadCommand: &UploadCommand{
				Size:     maxAvatarSize,
				UserID:   entity.Email("user@user.com"),
				FileName: "test.jpeg",
				MimeType: "image/jpeg",
				Buf:      []byte("test"),
			},
			ErrExpected:   false,
			ExpectedError: nil,
		},
		{
			Name: "image/png",
			UploadCommand: &UploadCommand{
				Size:     maxAvatarSize,
				UserID:   entity.Email("user@user.com"),
				FileName: "test.png",
				MimeType: "image/png",
				Buf:      []byte("test"),
			},
			ErrExpected:   false,
			ExpectedError: nil,
		},
		{
			Name: "image/webp",
			UploadCommand: &UploadCommand{
				Size:     maxAvatarSize,
				UserID:   entity.Email("user@user.com"),
				FileName: "test.webp",
				MimeType: "image/webp",
				Buf:      []byte("test"),
			},
			ErrExpected:   false,
			ExpectedError: nil,
		},
		{
			Name: "image/abc",
			UploadCommand: &UploadCommand{
				Size:     maxAvatarSize,
				UserID:   entity.Email("user@user.com"),
				FileName: "test.abc",
				MimeType: "image/abc",
				Buf:      []byte("test"),
			},
			ErrExpected:   true,
			ExpectedError: entity.ErrInvalidContentErrorMessage,
		},
		{
			Name: "empty",
			UploadCommand: &UploadCommand{
				Size:     maxAvatarSize,
				UserID:   entity.Email("user@user.com"),
				FileName: "test",
				MimeType: "",
				Buf:      []byte("test"),
			},
			ErrExpected:   true,
			ExpectedError: entity.ErrInvalidContentErrorMessage,
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx := context.Background()
			sut := createSut(ctx, ctrl, tc.UploadCommand, !tc.ErrExpected)
			e, err := sut.Upload(ctx, tc.UploadCommand)
			if tc.ErrExpected {
				require.Error(t, err)
				require.ErrorIs(t, err, entity.ErrInvalidContentErrorMessage)
				require.Nil(t, e)
			} else {
				require.NoError(t, err)
				require.NotNil(t, e)
			}
		})
	}
}

func TestDeleteShouldBeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	id := entity.NewAvatarID()
	userID := entity.Email("user@user.com")
	a := &entity.Avatar{
		ID:     id,
		UserID: userID,
	}
	repo.EXPECT().Find(ctx, id).Return(a, nil)
	repo.EXPECT().Delete(ctx, id).Return(nil)
	taskRepo.EXPECT().Insert(ctx, gomock.Any(), entity.AvatarDeleted, gomock.Any()).Return(nil)

	sut := NewAvatarService(newTransactor(), repo, taskRepo, s3, codec)

	err := sut.Delete(ctx, id, userID)

	require.NoError(t, err)
}

func TestDeleteShouldBeFailureForAnotherUserID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	id := entity.NewAvatarID()
	userID := entity.Email("user@user.com")
	a := &entity.Avatar{
		ID:     id,
		UserID: userID,
	}
	userID2 := entity.Email("user2@user.com")
	repo.EXPECT().Find(ctx, id).Return(a, nil)

	sut := NewAvatarService(newTransactor(), repo, taskRepo, s3, codec)

	err := sut.Delete(ctx, id, userID2)

	require.Error(t, err)
	var pErr *entity.ProfileError
	require.ErrorAs(t, err, &pErr)
	require.NotNil(t, pErr)
	require.Equal(t, entity.PermissionDeniedErrorCode, pErr.Code)
}

func TestDeleteShouldBeFailureIfAvatarNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()
	repo := mocks.NewMockAvatarRepository(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	id := entity.NewAvatarID()
	userID := entity.Email("user@user.com")

	repo.EXPECT().Find(ctx, id).Return(nil, infra.ErrNoRows)
	taskRepo.EXPECT().Insert(ctx, gomock.Any(), entity.AvatarDeleted, gomock.Any()).Return(nil).Times(0)

	sut := NewAvatarService(newTransactor(), repo, taskRepo, s3, codec)

	err := sut.Delete(ctx, id, userID)

	require.Error(t, err)
	var pErr *entity.ProfileError
	require.ErrorAs(t, err, &pErr)
	require.NotNil(t, pErr)
	require.Equal(t, entity.NotFoundEntityErrorCode, pErr.Code)
}

func createSut(ctx context.Context, ctrl *gomock.Controller, command *UploadCommand, expectUploadTask bool) *AvatarService {
	repo := mocks.NewMockAvatarRepository(ctrl)
	taskRepo := mocks.NewMockTaskRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)

	codec.EXPECT().DecodeConfig(command.Buf).Return(image.Config{
		ColorModel: nil,
		Width:      10,
		Height:     10,
	}, nil).AnyTimes()
	id := entity.NewAvatarID()
	av := &entity.Avatar{
		ID: id,
	}

	if expectUploadTask {
		s3.EXPECT().Upload(ctx, gomock.Any(), gomock.Any(), command.Buf).Return(new("etag"), nil)
		repo.EXPECT().Insert(
			ctx,
			gomock.Any(),
			baseName(command.FileName),
			command.UserID,
			10,
			10,
			command.Size,
			command.MimeType,
			gomock.Any(),
		).Return(av, nil)
		taskRepo.EXPECT().Insert(ctx, gomock.Any(),
			entity.AvatarUploaded, gomock.Any()).Return(nil)
	}
	return NewAvatarService(newTransactor(), repo, taskRepo, s3, codec)
}

type transactor struct {
}

func newTransactor() *transactor {
	return &transactor{}
}

func (t *transactor) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
