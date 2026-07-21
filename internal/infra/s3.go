package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type s3Client struct {
	tracer     trace.Tracer
	client     *s3.Client
	bucketName string
}

func NewS3(tracer trace.Tracer, client *s3.Client, bucket string) *s3Client {
	return &s3Client{
		tracer:     tracer,
		client:     client,
		bucketName: bucket,
	}
}
func (s *s3Client) Check(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "s3Client.Check", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: new(s.bucketName),
	})
	if err != nil {
		return fmt.Errorf("s3 healthcheck failed: %w", err)
	}

	return nil
}
func (s *s3Client) Upload(ctx context.Context, userID entity.Email, key string, data []byte) (*string, error) {
	ctx, span := s.tracer.Start(ctx, "s3Client.Upload", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(
		attribute.String("key", key),
	)
	fullKey := fmt.Sprintf("%s/%s", userID, key)
	o, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        new(s.bucketName),
		Key:           new(fullKey),
		Body:          bytes.NewReader(data),
		ContentLength: new(int64(len(data))),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}
	return o.ETag, nil
}

func (s *s3Client) Download(ctx context.Context, userID entity.Email, key string) ([]byte, error) {
	ctx, span := s.tracer.Start(ctx, "s3Client.Download", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(
		attribute.String("key", key),
	)
	fullKey := fmt.Sprintf("%s/%s", userID, key)
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: new(s.bucketName),
		Key:    new(fullKey),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to download object: %w", err)
	}

	defer func() {
		_ = result.Body.Close()
	}()

	buf := &bytes.Buffer{}
	_, err = io.Copy(buf, result.Body)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to read object data: %v", err)
	}
	return buf.Bytes(), nil
}

func (s *s3Client) Delete(ctx context.Context, userID entity.Email, key string) error {
	ctx, span := s.tracer.Start(ctx, "s3Client.Delete", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(
		attribute.String("key", key),
	)
	fullKey := fmt.Sprintf("%s/%s", userID, key)
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: new(s.bucketName),
		Key:    new(fullKey),
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}
