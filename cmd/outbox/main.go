package main

import (
	"context"
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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var conf config.GophOutboxConfig
	if err := env.Parse(&conf); err != nil {
		panic(err)
	}
	logger, err := createLogger()
	if err != nil {
		panic(err)
	}
	pgpool, err := createPg(ctx, conf)
	if err != nil {
		logger.Fatal("failed to create postgres pool", zap.Error(err))
	}
	defer pgpool.Close()
	retryablePool := retryablepgxpool.New(pgpool)
	if err = retryablePool.Ping(ctx); err != nil {
		logger.Fatal("failed to ping postgres", zap.Error(err))
	}
	app := outbox.New(infra.NewTX(retryablePool), infra.NewTaskRepository(retryablePool))
	if conf.Workers == 0 {
		conf.Workers = runtime.GOMAXPROCS(0) * 4
	}
	producerPool := kafka.NewKafkaProducerPool(conf.Workers, func() (*kgo.Client, error) {
		return newClient(conf)
	})
	defer producerPool.Close()
	app.Start(logging.SetLogger(ctx, logger), producerPool.Producers(), conf.BatchSize, conf.WaitTime, 1000*conf.WaitTime)
}

func createLogger() (*zap.Logger, error) {
	return zap.NewDevelopment()
}

func createPg(ctx context.Context, conf config.GophOutboxConfig) (*pgxpool.Pool, error) {
	p, err := pgxpool.New(ctx, conf.Database)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func newClient(conf config.GophOutboxConfig) (*kgo.Client, error) {
	return kgo.NewClient(
		kgo.SeedBrokers(conf.Brokers),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordDeliveryTimeout(time.Duration(conf.DeliveryTimeout)*time.Second),
		kgo.ProducerBatchMaxBytes(int32(conf.BatchMaxSize)*1024),
		kgo.ProducerLinger(100*time.Millisecond),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
		kgo.RecordRetries(7),
	)
}
