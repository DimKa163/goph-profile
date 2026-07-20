package entity

import (
	"context"
)

//go:generate mockgen -source=s3.go -destination=mocks/mock_s3.go -package=mocks
type S3 interface {
	Check(ctx context.Context) error
	Upload(ctx context.Context, userID Email, key string, data []byte) (*string, error)
	Download(ctx context.Context, userID Email, key string) ([]byte, error)
	Delete(ctx context.Context, userID Email, key string) error
}
