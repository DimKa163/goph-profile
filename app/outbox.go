package app

import (
	"context"

	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/worker/outbox"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

func RunOutbox(conf config.GophConfig, name, version, buildDate string) error {
	return run(name, version, func(ctx context.Context) error {
		logger := logging.Logger(ctx)
		pgpool, err := conf.CreatePg(ctx)
		if err != nil {
			logger.Fatal("failed to create postgres pool", zap.Error(err))
			return err
		}
		defer pgpool.Close()
		retryablePool := retryablepgxpool.New(pgpool)
		if err = retryablePool.Ping(ctx); err != nil {
			logger.Fatal("failed to ping postgres", zap.Error(err))
			return err
		}
		app := outbox.New(otel.Tracer("outbox"), infra.NewTX(retryablePool), infra.NewTaskRepository(retryablePool))
		producerPool := conf.ProducerPool(ctx)
		defer producerPool.Close()
		logger.Info("outbox started",
			zap.String("name", name),
			zap.String("version", version),
			zap.String("build_date", buildDate),
			zap.String("brokers", conf.Brokers),
			zap.Int("batch_max_size", conf.BatchMaxSize),
			zap.Duration("delivery_timeout", conf.DeliveryTimeout),
			zap.Int("batch_size", conf.BatchSize),
			zap.Duration("wait_time", conf.WaitTime),
			zap.Int("workers", conf.Workers),
			zap.Bool("database_configured", conf.Database != ""),
		)
		app.Start(logging.SetLogger(ctx, logger), producerPool.Producers(), conf.BatchSize, conf.WaitTime, 1000*conf.WaitTime)
		return nil
	})
}
