package infra

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Option func(*aws.Config)

func Region(region string) S3Option {
	return func(c *aws.Config) {
		c.Region = new(region)
	}
}

func Endpoint(region string) S3Option {
	return func(c *aws.Config) {
		c.Endpoint = new(region)
	}
}

func Credential(accessKey, secretKey, token string) S3Option {
	return func(c *aws.Config) {
		c.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, token)
	}
}

func ForcePathStyle() S3Option {
	return func(c *aws.Config) {
		c.S3ForcePathStyle = new(true)
	}
}

func UseSSL(val bool) S3Option {
	return func(c *aws.Config) {
		c.DisableSSL = new(!val)
	}
}

func MaxRetries(n int) S3Option {
	return func(c *aws.Config) {
		c.MaxRetries = new(n)
	}
}

type s3Client struct {
	client     *s3.S3
	bucketName string
}

func NewS3(bucketName string, opt ...S3Option) *s3Client {
	if len(opt) == 0 {
		panic("no options provided")
	}
	var config aws.Config
	for _, o := range opt {
		o(&config)
	}
	sess := session.Must(session.NewSession(&config))
	return &s3Client{
		client:     s3.New(sess),
		bucketName: bucketName,
	}
}
func (s *s3Client) Check(ctx context.Context) error {
	_, err := s.client.HeadBucketWithContext(ctx, &s3.HeadBucketInput{
		Bucket: new(s.bucketName),
	})
	if err != nil {
		return fmt.Errorf("s3 healthcheck failed: %w", err)
	}

	return nil
}
func (s *s3Client) Upload(ctx context.Context, key string, reader io.ReadSeeker) (*string, error) {
	o, err := s.client.PutObjectWithContext(ctx, &s3.PutObjectInput{
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
	result, err := s.client.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: new(s.bucketName),
		Key:    new(key),
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

func (s *s3Client) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: new(s.bucketName),
		Key:    new(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}
