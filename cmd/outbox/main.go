package main

import (
	"context"
	"fmt"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/worker/outbox"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
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

	var conf config.GophOutboxConfig
	if err := env.Parse(&conf); err != nil {
		panic(err)
	}

	provider, sh, err := logging.NewLoggerProvider(
		ctx,
		semconv.ServiceNameKey.String("goph-outbox"),
		semconv.ServiceVersionKey.String("1.0.0"),
	)
	if err != nil {
		panic(err)
	}
	defer sh()
	logger := createLogger("goph-outbox", provider)
	ctx = logging.SetLogger(ctx, logger)

	cl, err := logging.InitMetricProvider(
		ctx,
		semconv.ServiceNameKey.String("goph-outbox"),
		semconv.ServiceVersionKey.String("1.0.0"),
	)
	if err != nil {
		logger.Fatal("Failed to initialize metric provider", zap.Error(err))
	}
	defer cl()
	d, err := logging.NewTracerProvider(
		ctx,
		semconv.ServiceNameKey.String("goph-outbox"),
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
	app := outbox.New(otel.Tracer("outbox"), infra.NewTX(retryablePool), infra.NewTaskRepository(retryablePool))
	if conf.Workers == 0 {
		conf.Workers = runtime.GOMAXPROCS(0) * 4
	}
	producerPool := kafka.NewKafkaProducerPool(conf.Workers, func() (*kgo.Client, error) {
		return newClient(conf)
	})
	defer producerPool.Close()
	app.Start(logging.SetLogger(ctx, logger), producerPool.Producers(), conf.BatchSize, conf.WaitTime, 1000*conf.WaitTime)
}

func createLogger(name string, provider *sdklog.LoggerProvider) *zap.Logger {
	return logging.NewZap(name, provider)
}

func createPg(ctx context.Context, conf config.GophOutboxConfig) (*pgxpool.Pool, error) {
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

func newClient(conf config.GophOutboxConfig) (*kgo.Client, error) {
	kotelTracer := kotel.NewTracer(
		kotel.TracerProvider(otel.GetTracerProvider()),
		kotel.TracerPropagator(otel.GetTextMapPropagator()),
		kotel.ClientID("goph-outbox"),
	)
	kotelService := kotel.NewKotel(
		kotel.WithTracer(kotelTracer),
	)
	return kgo.NewClient(
		kgo.SeedBrokers(conf.Brokers),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordDeliveryTimeout(conf.DeliveryTimeout),
		kgo.ProducerBatchMaxBytes(int32(conf.BatchMaxSize)*1024),
		kgo.ProducerLinger(100*time.Millisecond),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
		kgo.RecordRetries(7),
		kgo.WithHooks(kotelService.Hooks()...),
	)
}
