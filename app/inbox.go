package app

import (
	"context"
	"fmt"
	"os"

	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/observability"
	"github.com/DimKa163/goph-profile/internal/shared/img"
	"github.com/DimKa163/goph-profile/internal/usecase"
	"github.com/DimKa163/goph-profile/internal/worker/inbox"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/twmb/franz-go/plugin/kotel"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

// RunInbox starts the inbox worker application.
func RunInbox(conf config.GophConfig, name, version, buildDate, commit string) error {
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
		s3Client, err := conf.CreateS3(ctx)
		if err != nil {
			logger.Fatal("failed to get hostname", zap.Error(err))
			return err
		}

		metricService, err := observability.NewMetricService(name)
		if err != nil {
			logger.Fatal("failed to create metric service", zap.Error(err))
			return err
		}

		s3 := infra.NewS3(otel.Tracer("s3"), s3Client, conf.Bucket)

		avatarRepo := infra.NewAvatarRepository(retryablePool)
		uploadHandler := usecase.NewUploadHandler(avatarRepo, s3, img.NewCodec())
		deleteHandler := usecase.NewDeleteHandler(avatarRepo, s3)
		n, err := os.Hostname()
		if err != nil {
			logger.Fatal("failed to get hostname", zap.Error(err))
			return err
		}
		clientID := fmt.Sprintf("%s-%s", conf.Group, n)
		kotelTracer := kotel.NewTracer(
			kotel.TracerProvider(otel.GetTracerProvider()),
			kotel.TracerPropagator(otel.GetTextMapPropagator()),
			kotel.ClientID(clientID),
			kotel.ConsumerGroup(conf.Group),
		)
		kotelService := kotel.NewKotel(
			kotel.WithTracer(kotelTracer),
		)

		consumer, err := conf.Consumer(ctx, kotelService, clientID, "avatar")
		if err != nil {
			logger.Fatal("failed to create consumer", zap.Error(err))
			return err
		}
		logger.Info("inbox started",
			zap.String("name", name),
			zap.String("version", version),
			zap.String("build_date", buildDate),
			zap.String("commit", commit),
			zap.String("brokers", conf.Brokers),
			zap.String("group", conf.Group),
			zap.String("client_id", clientID),
			zap.Int32("batch_max_size", conf.BatchMaxSize),
			zap.Duration("delivery_timeout", conf.DeliveryTimeout),
			zap.Bool("auto_commit", conf.AutoCommit),
			zap.String("bucket", conf.Bucket),
			zap.String("region", conf.Region),
			zap.String("s3_endpoint", conf.Endpoint),
			zap.Bool("s3_use_ssl", conf.UseSSL),
			zap.Bool("database_configured", conf.Database != ""),
		)
		consumerHandler := inbox.AvatarUploadedEventWorker(ctx, kotelTracer, inbox.Idempotency(
			infra.NewTX(retryablePool),
			metricService,
			infra.NewInboxRepo(retryablePool),
		),
			func(ctx context.Context, eventType string) (usecase.InboxHandler, error) {
				switch eventType {
				case entity.AvatarUploaded.String():
					return uploadHandler, nil
				case entity.AvatarDeleted.String():
					return deleteHandler, nil
				}
				return nil, fmt.Errorf("not found handler")
			}, consumer)
		return consumerHandler()
	})
}
