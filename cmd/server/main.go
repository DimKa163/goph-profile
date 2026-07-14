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
	"github.com/caarlos0/env"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
		logger.Fatal("failed to create postgres pool", zap.Error(err))
	}
	defer pgpool.Close()
	retryablePool := retryablepgxpool.New(pgpool)
	if err = retryablePool.Ping(ctx); err != nil {
		logger.Fatal("failed to ping postgres", zap.Error(err))
	}
	uc := rest.NewUserController(newUserService(conf, retryablePool))
	ac := rest.NewAvatarController(newAvatarService(conf, retryablePool))
	web := rest.NewWebController(newUserService(conf, retryablePool))
	if err = infra.Migrate(pgpool, "./migrations"); err != nil {
		logger.Fatal("failed to migrate", zap.Error(err))
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

func newAvatarService(conf config.GophConfig, pool *retryablepgxpool.Pool) *usecase.AvatarService {
	return usecase.NewAvatarService(infra.NewTX(pool), infra.NewAvatarRepository(pool),
		infra.NewTaskRepository(pool),
		infra.NewS3(conf.Bucket,
			infra.Region(conf.Region),
			infra.Endpoint(conf.Endpoint),
			infra.ForcePathStyle(),
			infra.Credential(conf.AccessKey, conf.SecretKey, ""),
			infra.UseSSL(conf.UseSSL),
			infra.MaxRetries(3),
		),
		img.NewCodec())
}

func newUserService(conf config.GophConfig, pool *retryablepgxpool.Pool) *usecase.UserService {
	return usecase.NewUserService(infra.NewTX(pool), infra.NewAvatarRepository(pool),
		infra.NewTaskRepository(pool), infra.NewS3(conf.Bucket,
			infra.Region(conf.Region),
			infra.Endpoint(conf.Endpoint),
			infra.ForcePathStyle(),
			infra.Credential(conf.AccessKey, conf.SecretKey, ""),
			infra.UseSSL(conf.UseSSL),
			infra.MaxRetries(3),
		))
}

type TemplateRenderer struct {
	Templates *template.Template
}

func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return r.Templates.ExecuteTemplate(w, name, data)
}
