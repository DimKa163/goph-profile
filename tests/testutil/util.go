package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DimKa163/goph-profile/app"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel"
)

func NewPostgresPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)

	ctr, err := tcpostgres.Run(
		ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("goph_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	testcontainers.CleanupContainer(t, ctr)

	connString, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connString)
	require.NoError(t, err)

	t.Cleanup(pool.Close)

	err = infra.Migrate(pool, "../migrations")
	require.NoError(t, err)

	return pool
}

func NewS3(t *testing.T) *s3.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	volumeName := "goph-minio-" + strings.ReplaceAll(t.Name(), "/", "-")
	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "admin",
			"MINIO_ROOT_PASSWORD": "admin12345",
		},
		Cmd: []string{"server", "/data", "--console-address", ":9001"},
		Mounts: testcontainers.Mounts(
			testcontainers.VolumeMount(volumeName, "/data"),
		),
		WaitingFor: wait.ForHTTP("/minio/health/ready").WithPort("9000/tcp").WithStartupTimeout(160 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatal(err)
		}
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	port, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := s3config.LoadDefaultConfig(
		ctx,
		s3config.WithRegion("eu-central-1"),
		s3config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("admin", "admin12345", ""),
		),
		s3config.WithRetryMaxAttempts(3),
	)
	if err != nil {
		t.Fatal(err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("http://%s:%s", host, port.Port()))
		o.UsePathStyle = true
	})

	recreateBucket(t, ctx, client, "goph")

	return client
}

func recreateBucket(t *testing.T, ctx context.Context, client *s3.Client, bucket string) {
	t.Helper()

	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		emptyBucket(t, ctx, client, bucket)

		_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
		require.NoError(t, err)
	}

	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
		return err == nil
	}, 10*time.Second, 100*time.Millisecond)
}

func emptyBucket(t *testing.T, ctx context.Context, client *s3.Client, bucket string) {
	t.Helper()

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		require.NoError(t, err)

		if len(page.Contents) == 0 {
			continue
		}

		objects := make([]s3types.ObjectIdentifier, 0, len(page.Contents))
		for _, object := range page.Contents {
			objects = append(objects, s3types.ObjectIdentifier{
				Key: object.Key,
			})
		}

		_, err = client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(bucket),
			Delete: &s3types.Delete{
				Objects: objects,
				Quiet:   aws.Bool(true),
			},
		})
		require.NoError(t, err)
	}
}

func NewServer(t *testing.T) *httptest.Server {
	handler, err := app.NewServer(t.Context(), "test", infra.NewS3(otel.Tracer("s3"), NewS3(t), "goph"), NewPostgresPool(t))
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(handler)
}

func UploadAvatar(t *testing.T, server *httptest.Server, userID string, path string) (*http.Response, error) {
	t.Helper()

	filePath := testdataFilePath(t, path)
	f, err := os.ReadFile(filePath) // #nosec G304 -- path is constrained to tests/testdata by testdataFilePath.
	require.NoError(t, err)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	filePart, err := writer.CreateFormFile("image", filepath.Base(filePath))
	require.NoError(t, err)
	_, err = io.Copy(filePart, bytes.NewReader(f))
	require.NoError(t, err)
	err = writer.WriteField("description", "avatar")
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, fmt.Sprintf("%s/api/v1/avatars", server.URL), &body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-User-Id", userID)
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	return client.Do(req)
}

func testdataFilePath(t *testing.T, name string) string {
	t.Helper()

	cleanName := filepath.Clean(name)
	require.False(t, filepath.IsAbs(cleanName), "testdata file path must be relative")
	require.Equal(t, filepath.Base(cleanName), cleanName, "testdata file path must be a file name")

	fullPath := filepath.Join("testdata", cleanName)
	testdataDir, err := filepath.Abs("testdata")
	require.NoError(t, err)

	absPath, err := filepath.Abs(fullPath)
	require.NoError(t, err)

	relPath, err := filepath.Rel(testdataDir, absPath)
	require.NoError(t, err)
	require.NotEqual(t, "..", relPath)
	require.False(t, strings.HasPrefix(relPath, ".."+string(os.PathSeparator)))

	return fullPath
}
