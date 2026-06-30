package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/DimKa163/goph-profile/internal/api"
	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/DimKa163/goph-profile/internal/handlers"
	"github.com/DimKa163/goph-profile/internal/img"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/DimKa163/goph-profile/internal/infra/repository"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/caarlos0/env"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	if err = infra.Migrate(pgpool, "./migrations"); err != nil {
		logger.Fatal("failed to migrate", zap.Error(err))
	}

	cl, err := newClient(conf)
	if err != nil {
		logger.Fatal("failed to create client", zap.Error(err))
	}
	defer cl.Close()
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogMethod:    true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogHost:      true,
		LogUserAgent: true,
		LogError:     true,
		BeforeNextFunc: func(c echo.Context) {
			req := c.Request()
			c.SetRequest(req.WithContext(logging.SetLogger(req.Context(), logger)))
		},
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			fields := []zap.Field{
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
				zap.String("remote_ip", v.RemoteIP),
				zap.String("host", v.Host),
				zap.String("user_agent", v.UserAgent),
			}
			log := logging.Logger(c.Request().Context())
			if v.Error != nil {
				fields = append(fields, zap.Error(v.Error))
			}

			if v.Status >= 500 {
				log.Error("request failed", fields...)
			} else if v.Status >= 400 {
				log.Warn("request client error", fields...)
			} else {
				log.Info("request", fields...)
			}
			return nil
		},
	}))
	e.GET("/health", func(c echo.Context) error {
		var state struct {
			Server bool `json:"server"`
			Db     bool `json:"db"`
		}
		state.Server = true
		state.Db = true
		if err := pgpool.Ping(c.Request().Context()); err != nil {
			state.Server = false
		}
		return c.JSON(http.StatusOK, state)
	})
	webApi := e.Group("/api")
	v1 := webApi.Group("/v1")
	uc := api.NewUserController()
	uc.Register(v1)
	ac := api.NewAvatarController(newAvatarService(conf, pgpool, cl))
	ac.Register(v1)
	server := &http.Server{
		Addr:    conf.Addr,
		Handler: e,
	}
	go func() {
		<-ctx.Done()
		logger.Info("shutting down server...")
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err = server.Shutdown(timeoutCtx); err != nil {
			logger.Warn("failed to shutdown server", zap.Error(err))
		}
		logger.Info("server shutdown")
	}()
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		e.Logger.Fatal(err)
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

func newClient(conf config.GophConfig) (*kgo.Client, error) {
	return kgo.NewClient(
		kgo.SeedBrokers(conf.Brokers),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordDeliveryTimeout(time.Duration(conf.DeliveryTimeout)*time.Second),
		kgo.ProducerBatchMaxBytes(int32(conf.BatchMaxSize)*1024),
		kgo.ProducerLinger(100*time.Millisecond),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
	)
}

func newAvatarService(conf config.GophConfig, pool *pgxpool.Pool, client *kgo.Client) handlers.Uploader {
	return handlers.NewUploader(infra.NewS3(&aws.Config{
		Region:           new(conf.Region),
		Endpoint:         new(conf.Endpoint),
		S3ForcePathStyle: new(true),
		Credentials:      credentials.NewStaticCredentials(conf.AccessKey, conf.SecretKey, ""),
		DisableSSL:       new(!conf.UseSSL),
	}, conf.Bucket), repository.NewAvatarRepository(pool), img.NewDecoder(),
		kafka.NewProducer(client, kafka.Topic("avatar")))
}
