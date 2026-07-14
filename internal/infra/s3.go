package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Client struct {
	client     *s3.Client
	bucketName string
}

func NewS3(client *s3.Client, bucket string) *s3Client {
	return &s3Client{
		client:     client,
		bucketName: bucket,
	}
}
func (s *s3Client) Check(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: new(s.bucketName),
	})
	if err != nil {
		return fmt.Errorf("s3 healthcheck failed: %w", err)
	}

	return nil
}
func (s *s3Client) Upload(ctx context.Context, key string, reader io.ReadSeeker) (*string, error) {
	o, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: new(s.bucketName),
		Key:    new(key),
		Body:   reader,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}
	return o.ETag, nil
}

func (s *s3Client) Download(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: new(s.bucketName),
		Key:    new(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}

	defer func() {
		_ = result.Body.Close()
	}()

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, result.Body)

	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %v", err)
	}
	return buf.Bytes(), nil
}

func (s *s3Client) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: new(s.bucketName),
		Key:    new(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}
