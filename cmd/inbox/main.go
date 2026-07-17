package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/shared/img"
	"github.com/DimKa163/goph-profile/internal/usecase"
	"github.com/DimKa163/goph-profile/internal/worker/inbox"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/caarlos0/env/v11"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kotel"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var conf config.GophInboxConfig
	if err := env.Parse(&conf); err != nil {
		panic(err)
	}
	provider, sh, err := logging.NewLoggerProvider(
		ctx,
		semconv.ServiceNameKey.String("goph-inbox"),
		semconv.ServiceVersionKey.String("1.0.0"),
	)
	if err != nil {
		panic(err)
	}
	defer sh()
	logger := createLogger("goph-inbox", provider)
	ctx = logging.SetLogger(ctx, logger)

	cl, err := logging.InitMetricProvider(
		ctx,
		semconv.ServiceNameKey.String("goph-inbox"),
		semconv.ServiceVersionKey.String("1.0.0"),
	)
	if err != nil {
		logger.Fatal("Failed to initialize metric provider", zap.Error(err))
	}
	defer cl()
	d, err := logging.NewTracerProvider(
		ctx,
		semconv.ServiceNameKey.String("goph-inbox"),
		semconv.ServiceVersionKey.String("1.0.0"),
	)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer d()
	pgpool, err := createPg(ctx, conf)
	if err != nil {
		logger.Fatal("failed to create postgres pool", zap.Error(err))
	}
	defer pgpool.Close()

	retryablePool := retryablepgxpool.New(pgpool)
	if err = retryablePool.Ping(ctx); err != nil {
		logger.Fatal("failed to ping postgres", zap.Error(err))
	}

	s3Client, err := createS3Client(ctx, conf)
	if err != nil {
		logger.Fatal("failed to create S3 client", zap.Error(err))
	}
	n, err := os.Hostname()
	if err != nil {
		logger.Fatal("failed to get hostname", zap.Error(err))
	}

	kotelTracer := kotel.NewTracer(
		kotel.TracerProvider(otel.GetTracerProvider()),
		kotel.TracerPropagator(otel.GetTextMapPropagator()),
		kotel.ClientID(fmt.Sprintf("%s-%s", conf.Group, n)),
		kotel.ConsumerGroup(conf.Group),
	)
	kotelService := kotel.NewKotel(
		kotel.WithTracer(kotelTracer),
	)
	client, err := kgo.NewClient(
		kgo.SeedBrokers(conf.Brokers),
		kgo.ConsumerGroup(conf.Group),
		kgo.ConsumeTopics("avatar"),
		kgo.ClientID(fmt.Sprintf("%s-%s", conf.Group, n)),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		kgo.FetchMaxWait(100*time.Millisecond),
		kgo.FetchMinBytes(1),
		kgo.FetchMaxBytes(10*1024*1024),
		kgo.RecordDeliveryTimeout(conf.DeliveryTimeout),
		kgo.WithHooks(kotelService.Hooks()...),
	)
	if err != nil {
		logger.Fatal("failed to connect to kafka", zap.Error(err))
	}
	if err = client.Ping(ctx); err != nil {
		logger.Fatal("failed to ping kafka client", zap.Error(err))
	}
	clientS3 := infra.NewS3(otel.Tracer("s3"), s3Client, conf.Bucket)
	avatarRepo := infra.NewAvatarRepository(retryablePool)
	uploadHandler := usecase.NewUploadHandler(avatarRepo, clientS3, img.NewCodec())
	deleteHandler := usecase.NewDeleteHandler(avatarRepo, clientS3)

	w := inbox.AvatarUploadedEventWorker(ctx, kotelTracer, inbox.Idempotency(infra.NewTX(retryablePool), infra.NewInboxRepo(retryablePool)),
		func(ctx context.Context, eventType string) (usecase.InboxHandler, error) {
			switch eventType {
			case entity.AvatarUploaded.String():
				return uploadHandler, nil
			case entity.AvatarDeleted.String():
				return deleteHandler, nil
			}
			return nil, fmt.Errorf("not found handler")
		}, client)
	if err = w(); err != nil {
		logger.Fatal("failed to start inbox", zap.Error(err))
	}
}

func createLogger(name string, provider *sdklog.LoggerProvider) *zap.Logger {
	return logging.NewZap(name, provider)
}

func createPg(ctx context.Context, conf config.GophInboxConfig) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(conf.Database)
	if err != nil {
		return nil, err
	}
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	if err = otelpgx.RecordStats(pool); err != nil {
		return nil, fmt.Errorf("record pg stats: %w", err)
	}
	return pool, nil
}

func createS3Client(ctx context.Context, conf config.GophInboxConfig) (*s3.Client, error) {
	cfg, err := s3config.LoadDefaultConfig(
		ctx,
		s3config.WithRegion(conf.Region),
		s3config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(conf.AccessKey, conf.SecretKey, ""),
		),
		s3config.WithRetryMaxAttempts(3),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(conf.Endpoint)
		o.UsePathStyle = true // важно для MinIO
	})
	return client, nil
}
