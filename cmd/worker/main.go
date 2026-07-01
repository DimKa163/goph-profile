package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/DimKa163/goph-profile/internal/handlers"
	"github.com/DimKa163/goph-profile/internal/img"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/infra/repository"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/worker"
	"github.com/caarlos0/env"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var conf config.GophConfig
	if err := env.Parse(&conf); err != nil {
		panic(err)
	}
	logger, err := createLogger()
	if err != nil {
		panic(err)
	}
	ctx = logging.SetLogger(ctx, logger)
	pgpool, err := createPg(ctx, conf)
	if err != nil {
		logger.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer pgpool.Close()
	n, err := os.Hostname()
	if err != nil {
		logger.Fatal("failed to get hostname", zap.Error(err))
	}
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
	)
	if err != nil {
		logger.Fatal("failed to connect to kafka", zap.Error(err))
	}
	w := worker.AvatarUploadedEventWorker(ctx, handlers.NewUploadHandler(repository.NewAvatarRepository(pgpool), infra.NewS3(conf.Bucket,
		infra.Region(conf.Region),
		infra.Endpoint(conf.Endpoint),
		infra.ForcePathStyle(),
		infra.Credential(conf.AccessKey, conf.SecretKey, ""),
		infra.UseSSL(conf.UseSSL),
	), img.NewCodec()), client)

	if err = w(); err != nil {
		logger.Fatal("failed to start worker", zap.Error(err))
	}
}

func createLogger() (*zap.Logger, error) {
	return zap.NewDevelopment()
}

func createPg(ctx context.Context, conf config.GophConfig) (*pgxpool.Pool, error) {
	p, err := pgxpool.New(ctx, conf.Database)
	if err != nil {
		return nil, err
	}
	return p, nil
}
