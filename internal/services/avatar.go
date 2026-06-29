package services

import (
	"context"
	"fmt"
	"io"
)

type S3 interface {
	Upload(ctx context.Context, key string, reader io.ReadSeeker) error
	Download(ctx context.Context, key string) ([]byte, error)
}

type AvatarService struct {
	s3 S3
}

func NewAvatarService(s3 S3) *AvatarService {
	return &AvatarService{
		s3: s3,
	}
}

func (s *AvatarService) UploadAvatar(ctx context.Context, fileName, userID string, reader io.ReadSeeker) error {
	return s.s3.Upload(ctx, fmt.Sprintf("avatar_%s_%s", userID, fileName), reader)
}
