package entity

import (
	"context"
	"io"
)

//go:generate mockgen -source=s3.go -destination=mocks/mock_s3.go -package=mocks
type S3 interface {
	Upload(ctx context.Context, key string, reader io.ReadSeeker) (*string, error)
	Download(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}
