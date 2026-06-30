package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type s3Client struct {
	client     *s3.S3
	bucketName string
}

func NewS3(config *aws.Config, bucketName string) *s3Client {
	sess := session.Must(session.NewSession(config))
	return &s3Client{
		client:     s3.New(sess),
		bucketName: bucketName,
	}
}

func (s *s3Client) Upload(ctx context.Context, key string, reader io.ReadSeeker) error {
	_, err := s.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
	return nil
}

func (s *s3Client) Download(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}
	defer result.Body.Close()

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, result.Body)

	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %v", err)
	}
	return buf.Bytes(), nil
}
