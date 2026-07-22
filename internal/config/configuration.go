// Package config app settings
package config

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kotel"
	"go.opentelemetry.io/otel"
)

// GophConfig app configuration
type GophConfig struct {
	// Addr is the HTTP listen address.
	Addr string `env:"GOPH_ADDR" envDefault:":8080"`
	// Database is the PostgreSQL connection string.
	Database string `env:"GOPH_DATABASE" envDefault:"goph"`
	// Brokers is the Kafka broker list.
	Brokers string `env:"GOPH_BROKERS"`
	// BatchMaxSize is the Kafka producer batch size limit.
	BatchMaxSize int32 `env:"GOPH_BATCH_MAX_SIZE" envDefault:"1024"`
	// Bucket is the S3 bucket name.
	Bucket string `env:"GOPH_BUCKET" envDefault:"goph"`
	// Region is the S3 region.
	Region string `env:"GOPH_REGION" envDefault:"us-east-1"`
	// Endpoint is the S3-compatible endpoint.
	Endpoint string `env:"GOPH_ENDPOINT"`
	// AccessKey is the S3 access key.
	AccessKey string `env:"GOPH_ACCESS_KEY"`
	// SecretKey is the S3 secret key.
	SecretKey string `env:"GOPH_SECRET_KEY"`
	// UseSSL enables SSL for S3 connections.
	UseSSL bool `env:"GOPH_USE_SSL" envDefault:"false"`
	// DeliveryTimeout is the Kafka delivery timeout.
	DeliveryTimeout time.Duration `env:"GOPH_DELIVERY_TIMEOUT" envDefault:"10s"`
	// Group is the Kafka consumer group.
	Group string `env:"GOPH_GROUP" envDefault:"profile"`
	// AutoCommit enables Kafka offset autocommit.
	AutoCommit bool `env:"GOPH_AUTOCOMMIT" envDefault:"false"`
	// FetchMaxWait wait time between fetches
	FetchMaxWait time.Duration `env:"GOPH_FETCH_MAX_WAIT" envDefault:"100ms"`
	// FetchMinBytes min bytes to fetch
	FetchMinBytes int32 `env:"GOPH_FETCH_MIN_BYTES" envDefault:"1"`
	// FetchMaxBytes min bytes to fetch
	FetchMaxBytes int32 `env:"GOPH_FETCH_MAX_BYTES" envDefault:"10485760"`
	// KafkaRetryCount retry count
	KafkaRetryCount int `env:"GOPH_KAFKA_RETRY_COUNT" envDefault:"7"`
	// BatchSize is the worker processing batch size.
	BatchSize int `env:"GOPH_BATCH_SIZE" envDefault:"100"`
	// WaitTime is the worker polling interval.
	WaitTime time.Duration `env:"GOPH_WAIT_TIME" envDefault:"10s"`
	// Workers is the number of worker routines or producer clients.
	Workers int `env:"GOPH_WORKERS" envDefault:"0"`
}

// CreatePg configure postgresql-client
func (c GophConfig) CreatePg(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(c.Database)
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

// CreateS3 configure s3-client
func (c GophConfig) CreateS3(ctx context.Context) (*s3.Client, error) {
	cfg, err := s3config.LoadDefaultConfig(
		ctx,
		s3config.WithRegion(c.Region),
		s3config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(c.AccessKey, c.SecretKey, ""),
		),
		s3config.WithRetryMaxAttempts(3),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(c.Endpoint)
		o.UsePathStyle = true
	})
	return client, nil
}

// ProducerPool configure producer pool
func (c GophConfig) ProducerPool(ctx context.Context, clientID string) *kafka.KafkaProducerPool {
	if c.Workers == 0 {
		c.Workers = runtime.GOMAXPROCS(0) * 4
	}
	return kafka.NewKafkaProducerPool(c.Workers, func() (*kgo.Client, error) {
		return c.Producer(ctx, clientID)
	})
}

// Producer configure producer
func (c GophConfig) Producer(ctx context.Context, clientID string) (*kgo.Client, error) {
	kotelTracer := kotel.NewTracer(
		kotel.TracerProvider(otel.GetTracerProvider()),
		kotel.TracerPropagator(otel.GetTextMapPropagator()),
		kotel.ClientID(clientID),
	)
	kotelService := kotel.NewKotel(
		kotel.WithTracer(kotelTracer),
	)
	client, err := kgo.NewClient(
		kgo.SeedBrokers(c.Brokers),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordDeliveryTimeout(c.DeliveryTimeout),
		kgo.ProducerBatchMaxBytes(c.BatchMaxSize*1024),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
		kgo.RecordRetries(c.KafkaRetryCount),
		kgo.WithHooks(kotelService.Hooks()...),
	)
	if err != nil {
		return nil, err
	}
	if err = client.Ping(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

// Consumer configure consumer
func (c GophConfig) Consumer(ctx context.Context, kotel *kotel.Kotel, clientID string, topics ...string) (*kgo.Client, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(c.Brokers),
		kgo.ConsumerGroup(c.Group),
		kgo.ConsumeTopics(topics...),
		kgo.ClientID(clientID),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		kgo.FetchMaxWait(c.FetchMaxWait),
		kgo.FetchMinBytes(c.FetchMinBytes),
		kgo.FetchMaxBytes(c.FetchMaxBytes),
		kgo.RecordDeliveryTimeout(c.DeliveryTimeout),
		kgo.RecordRetries(c.KafkaRetryCount),
		kgo.WithHooks(kotel.Hooks()...),
	)
	if err != nil {
		return nil, err
	}
	if err = client.Ping(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

// Server configure web-server
func (c GophConfig) Server(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              c.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
	}
}
