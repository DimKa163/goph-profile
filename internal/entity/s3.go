package entity

import (
	"context"
)

// S3 describes avatar object storage operations.
//
//go:generate mockgen -source=s3.go -destination=mocks/mock_s3.go -package=mocks
type S3 interface {
	// Check verifies that the storage backend is reachable.
	Check(ctx context.Context) error
	// Upload stores or processes an uploaded avatar.
	Upload(ctx context.Context, userID Email, key string, data []byte) (*string, error)
	// Download reads an object from the storage backend.
	Download(ctx context.Context, userID Email, key string) ([]byte, error)
	// Delete removes or marks a record as deleted.
	Delete(ctx context.Context, userID Email, key string) error
}
