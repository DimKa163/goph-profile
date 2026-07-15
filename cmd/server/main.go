package main

import (
	"context"
	"errors"
	"html/template"
	"io"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/rest"
	"github.com/DimKa163/goph-profile/internal/shared/img"
	"github.com/DimKa163/goph-profile/internal/usecase"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/caarlos0/env"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var conf config.GophConfig
	if err := env.Parse(&conf); err != nil {
		panic(err)
	}

	provider, sh, err := logging.NewLoggerProvider(
		ctx,
		semconv.ServiceNameKey.String("goph-server"),
		semconv.ServiceVersionKey.String("1.0.0"),
	)
	if err != nil {
		panic(err)
	}
	defer sh()
	log := createLogger("goph-server", provider)
	ctx = logging.SetLogger(ctx, log)
	pgpool, err := createPg(ctx, conf)
	if err != nil {
		log.Fatal("failed to create postgres pool", zap.Error(err))
	}
	defer pgpool.Close()
	retryablePool := retryablepgxpool.New(pgpool)
	if err = retryablePool.Ping(ctx); err != nil {
		log.Fatal("failed to ping postgres", zap.Error(err))
	}
	s3Client, err := createS3Client(ctx, conf)
	if err != nil {
		log.Fatal("failed to create S3 client", zap.Error(err))
	}
	uc := rest.NewUserController(newUserService(conf, s3Client, retryablePool))
	ac := rest.NewAvatarController(newAvatarService(conf, s3Client, retryablePool))
	web := rest.NewWebController(newUserService(conf, s3Client, retryablePool))
	if err = infra.Migrate(pgpool, "./migrations"); err != nil {
		log.Fatal("failed to migrate", zap.Error(err))
	}
	e := echo.New()
	e.Renderer = &TemplateRenderer{
		Templates: template.Must(template.ParseGlob("web/static/*.html")),
	}
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
			c.SetRequest(req.WithContext(logging.SetLogger(req.Context(), log)))
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
		if err := retryablePool.Ping(c.Request().Context()); err != nil {
			state.Db = false
		}
		return c.JSON(http.StatusOK, state)
	})
	e.File("/", "web/static/index.html")
	webApi := e.Group("/api")
	v1 := webApi.Group("/v1")

	uc.Register(v1)
	ac.Register(v1)
	web.Register(e)
	server := &http.Server{
		Addr:    conf.Addr,
		Handler: e,
	}
	go func() {
		<-ctx.Done()
		log.Info("shutting down server...")
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err = server.Shutdown(timeoutCtx); err != nil {
			log.Warn("failed to shutdown server", zap.Error(err))
		}
		log.Info("server shutdown")
	}()
	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		e.Logger.Fatal(err)
	}
}

func createLogger(name string, provider *sdklog.LoggerProvider) *zap.Logger {
	return logging.NewZap(name, provider)
}

func createPg(ctx context.Context, conf config.GophConfig) (*pgxpool.Pool, error) {
	p, err := pgxpool.New(ctx, conf.Database)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func newAvatarService(conf config.GophConfig, s3client *s3.Client, pool *retryablepgxpool.Pool) *usecase.AvatarService {
	return usecase.NewAvatarService(infra.NewTX(pool), infra.NewAvatarRepository(pool),
		infra.NewTaskRepository(pool),
		infra.NewS3(s3client, conf.Bucket),
		img.NewCodec())
}

func newUserService(conf config.GophConfig, s3client *s3.Client, pool *retryablepgxpool.Pool) *usecase.UserService {
	return usecase.NewUserService(infra.NewTX(pool), infra.NewAvatarRepository(pool),
		infra.NewTaskRepository(pool), infra.NewS3(s3client, conf.Bucket))
}

func createS3Client(ctx context.Context, conf config.GophConfig) (*s3.Client, error) {
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

type TemplateRenderer struct {
	Templates *template.Template
}

func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return r.Templates.ExecuteTemplate(w, name, data)
}
