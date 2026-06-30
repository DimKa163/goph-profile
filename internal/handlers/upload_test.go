package handlers

import (
	"context"
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/handlers/mocks"
	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/beevik/guid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestUploaderShould_BeSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()
	ctx := context.Background()
	s3 := mocks.NewMockS3Uploader(ctrl)
	repo := mocks.NewMockAvatarInsertUpdaterRepository(ctrl)
	decoder := mocks.NewMockDecoder(ctrl)
	producer := mocks.NewMockProducer(ctrl)
	uploader := NewUploader(s3, repo, decoder, producer)

	avatar := &Avatar{
		Name:     "avatar",
		Size:     100,
		MimeType: "image/jpeg",
	}
	userID := guid.New()
	var imgCfg image.Config
	imgCfg.ColorModel = color.RGBAModel
	imgCfg.Width = 100
	imgCfg.Height = 100
	avatarId := guid.New()
	e := &entity.Avatar{
		ID:               *avatarId,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Name:             avatar.Name,
		Size:             avatar.Size,
		MimeType:         avatar.MimeType,
		Width:            imgCfg.Width,
		Height:           imgCfg.Height,
		S3Key:            "",
		UploadStatus:     "uploading",
		ProcessingStatus: "pending",
	}
	decoder.EXPECT().DecodeConfig(gomock.Any()).Return(imgCfg, nil)
	repo.EXPECT().Insert(
		ctx,
		avatar.Name,
		avatar.MimeType,
		avatar.Size,
		imgCfg.Width,
		imgCfg.Height,
		*userID,
	).Return(e, nil)
	s3.EXPECT().Upload(
		ctx,
		gomock.Any(),
		gomock.Any(),
	).Return(nil)
	e1 := &entity.Avatar{
		ID:               *avatarId,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Name:             avatar.Name,
		Size:             avatar.Size,
		MimeType:         avatar.MimeType,
		Width:            imgCfg.Width,
		Height:           imgCfg.Height,
		S3Key:            "",
		UploadStatus:     "uploaded",
		ProcessingStatus: "processing",
	}
	repo.EXPECT().Update(ctx, gomock.Any()).Return(e1, nil)

	producer.EXPECT().Write(ctx, gomock.Any(), gomock.Any(), kafka.Header{
		Key:   "event-type",
		Value: []byte("avatar-uploaded"),
	}).Return(nil)

	st, err := uploader(ctx, avatar, *userID)

	require.NoError(t, err)
	require.Equal(t, "processing", st.Status)
}

func TestUploaderShouldReturnErrIfSizeTooBig(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()
	ctx := context.Background()
	s3 := mocks.NewMockS3Uploader(ctrl)
	repo := mocks.NewMockAvatarInsertUpdaterRepository(ctrl)
	decoder := mocks.NewMockDecoder(ctrl)
	producer := mocks.NewMockProducer(ctrl)

	uploader := NewUploader(s3, repo, decoder, producer)

	avatar := &Avatar{
		Name:     "avatar",
		Size:     maxAvatarSize + 1,
		MimeType: "image/jpeg",
	}

	userID := guid.New()

	st, err := uploader(ctx, avatar, *userID)

	require.Error(t, err)
	require.ErrorIs(t, entity.ErrTooBigSizeErrorMessage, err)
	require.Nil(t, st)
}

func TestUploaderShouldReturnErrIfTypeisWrong(t *testing.T) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()
	ctx := context.Background()
	s3 := mocks.NewMockS3Uploader(ctrl)
	repo := mocks.NewMockAvatarInsertUpdaterRepository(ctrl)
	decoder := mocks.NewMockDecoder(ctrl)
	producer := mocks.NewMockProducer(ctrl)

	uploader := NewUploader(s3, repo, decoder, producer)

	avatar := &Avatar{
		Name:     "avatar",
		Size:     100,
		MimeType: "wrong/type",
	}

	userID := guid.New()

	st, err := uploader(ctx, avatar, *userID)

	require.Error(t, err)
	require.Nil(t, st)
}
