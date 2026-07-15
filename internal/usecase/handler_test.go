package usecase

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/entity/events/v1"
	"github.com/DimKa163/goph-profile/internal/entity/mocks"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/beevik/guid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestUploadHandlerShouldBeWhenFirstImageSuccessful(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)

	id := entity.NewAvatarID()
	userID := entity.Email("user@user.com")
	ev, data := newAvatarUploadedEvent(t, id)
	metadata := newAvatarMetadata(id, userID)
	repo.EXPECT().Find(ctx, id).Return(metadata, nil)
	expectSuccessfulImageProcessing(ctx, s3, codec)
	repo.EXPECT().InsertImage(ctx, metadata).Return(nil)
	repo.EXPECT().ActivateOnlyThis(ctx, userID, metadata.ID).Return(nil)
	sut := NewUploadHandler(repo, s3, codec)

	err := sut(ctx, ev.AvatarID, data)

	require.NoError(t, err)
}

func TestUploadHandlerShouldBeFailedWhenMetadataNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)

	id := entity.NewAvatarID()
	ev, data := newAvatarUploadedEvent(t, id)
	repo.EXPECT().Find(ctx, id).Return(nil, infra.ErrNoRows)

	sut := NewUploadHandler(repo, s3, codec)
	err := sut(ctx, ev.AvatarID, data)

	require.Error(t, err)

	var pe *entity.ProfileError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, entity.NotFoundEntityErrorCode, pe.Code)
}

func TestUploadHandlerShouldBeFailedWhenImagesNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	id := entity.NewAvatarID()
	userID := entity.Email("user@user.com")
	ev, data := newAvatarUploadedEvent(t, id)
	metadata := &entity.Avatar{
		ID:     id,
		UserID: userID,
	}
	repo.EXPECT().Find(ctx, id).Return(metadata, nil)

	sut := NewUploadHandler(repo, s3, codec)
	err := sut(ctx, ev.AvatarID, data)

	require.Error(t, err)

	var pe *entity.ProfileError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, entity.NotFoundEntityErrorCode, pe.Code)
}

func TestUploadHandlerShouldBeFailedWhenDecodeFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	id := entity.NewAvatarID()
	userID := entity.Email("user@user.com")
	ev, data := newAvatarUploadedEvent(t, id)
	metadata := newAvatarMetadata(id, userID)
	repo.EXPECT().Find(ctx, id).Return(metadata, nil)
	imageData := []byte("image data")
	s3.EXPECT().Download(ctx, "key").Return(imageData, nil)
	decodeErr := errors.New("decode failed")
	codec.EXPECT().Decode(gomock.Any()).Return(nil, "", decodeErr)

	sut := NewUploadHandler(repo, s3, codec)
	err := sut(ctx, ev.AvatarID, data)

	require.ErrorIs(t, err, decodeErr)
}

func TestUploadHandlerShouldBeFailedWhenUploadFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	id := entity.NewAvatarID()
	userID := entity.Email("user@user.com")
	ev, data := newAvatarUploadedEvent(t, id)
	metadata := newAvatarMetadata(id, userID)
	repo.EXPECT().Find(ctx, id).Return(metadata, nil)
	imageData := []byte("image data")
	s3.EXPECT().Download(ctx, "key").Return(imageData, nil)
	orig := &img{}
	codec.EXPECT().Decode(gomock.Any()).Return(orig, "jpeg", nil)
	img300 := &img{}
	codec.EXPECT().Encode(orig, "png", 85).Return([]byte("original-png"), nil)
	codec.EXPECT().Encode(orig, "webp", 85).Return([]byte("original-webp"), nil)
	codec.EXPECT().Thumbnail(orig, 300, 300).Return(img300)
	codec.EXPECT().Encode(img300, "jpeg", 85).Return([]byte("thumb300"), nil)
	uploadErr := errors.New("upload failed")
	s3.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, uploadErr).AnyTimes()

	sut := NewUploadHandler(repo, s3, codec)
	err := sut(ctx, ev.AvatarID, data)

	require.ErrorIs(t, err, uploadErr)
}

func TestUploadHandlerShouldBeFailedWhenInsertImageFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)
	codec := mocks.NewMockImageCodec(ctrl)
	id := entity.NewAvatarID()
	userID := entity.Email("user@user.com")
	ev, data := newAvatarUploadedEvent(t, id)
	metadata := newAvatarMetadata(id, userID)
	repo.EXPECT().Find(ctx, id).Return(metadata, nil)
	expectSuccessfulImageProcessing(ctx, s3, codec)
	insertErr := errors.New("insert image failed")
	repo.EXPECT().InsertImage(ctx, metadata).Return(insertErr)

	sut := NewUploadHandler(repo, s3, codec)
	err := sut(ctx, ev.AvatarID, data)

	require.ErrorIs(t, err, insertErr)
}

func newAvatarUploadedEvent(t *testing.T, id entity.AvatarID) (events.AvatarUploadedEvent, []byte) {
	t.Helper()

	ev := events.AvatarUploadedEvent{AvatarID: id.String()}
	data, err := ev.Bytes()
	require.NoError(t, err)

	return ev, data
}

func newAvatarMetadata(id entity.AvatarID, userID entity.Email) *entity.Avatar {
	return &entity.Avatar{
		ID:     id,
		UserID: userID,
		Images: []*entity.Image{
			{
				Format:   "jpeg",
				FileSize: 10000,
				S3Key:    "key",
				MimeType: "image/jpeg",
				Size:     entity.OriginalSize,
			},
		},
	}
}

func expectSuccessfulImageProcessing(
	ctx context.Context,
	s3 *mocks.MockS3,
	codec *mocks.MockImageCodec,
) {
	imageData := []byte("image data")
	s3.EXPECT().Download(ctx, "key").Return(imageData, nil)
	orig := &img{}
	codec.EXPECT().Decode(gomock.Any()).Return(orig, "jpeg", nil)
	img100 := &img{}
	img300 := &img{}
	codec.EXPECT().Thumbnail(orig, 100, 100).Return(img100).Times(3)
	codec.EXPECT().Thumbnail(orig, 300, 300).Return(img300).Times(3)
	codec.EXPECT().Encode(orig, "png", 85).Return([]byte("original-png"), nil)
	codec.EXPECT().Encode(orig, "webp", 85).Return([]byte("original-webp"), nil)
	for _, format := range []string{"png", "jpeg", "webp"} {
		codec.EXPECT().Encode(img100, format, 85).Return([]byte("thumb100"), nil)
		codec.EXPECT().Encode(img300, format, 85).Return([]byte("thumb300"), nil)
	}
	tag := "etag"
	s3.EXPECT().Upload(gomock.Any(), gomock.Any(), gomock.Any()).Return(&tag, nil).Times(8)
}
func TestDeleteHandlerShouldBeSuccessful(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)

	e := events.AvatarDeleted{
		AvatarID: guid.NewString(),
		S3Key:    []string{"key1", "key2"},
	}
	repo.EXPECT().DeleteImages(ctx, gomock.Any()).Return(nil)
	s3.EXPECT().Delete(gomock.Any(), "key1").Return(nil)
	s3.EXPECT().Delete(gomock.Any(), "key2").Return(nil)
	sut := NewDeleteHandler(repo, s3)
	data, _ := e.Bytes()
	err := sut(ctx, e.AvatarID, data)

	require.NoError(t, err)
}

func TestDeleteHandlerShouldBeFailedWhenNoImages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ctx := context.Background()

	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockS3(ctrl)

	e := events.AvatarDeleted{
		AvatarID: guid.NewString(),
		S3Key:    []string{"key1", "key2"},
	}
	repo.EXPECT().DeleteImages(ctx, gomock.Any()).Return(infra.ErrNoRows)
	for _, k := range e.S3Key {
		s3.EXPECT().Delete(gomock.Any(), k).Return(nil)
	}
	sut := NewDeleteHandler(repo, s3)
	data, _ := e.Bytes()
	err := sut(ctx, e.AvatarID, data)

	require.Error(t, err)
	var pe *entity.ProfileError
	require.ErrorAs(t, err, &pe)
	require.Equal(t, entity.NotFoundEntityErrorCode, pe.Code)
}

var _ image.Image = (*img)(nil)

type img struct {
}

func (i *img) ColorModel() color.Model {
	return nil
}

func (i *img) Bounds() image.Rectangle {
	return image.Rectangle{}

}

func (i *img) At(x, y int) color.Color {
	return nil
}
