package app

import (
	"context"
	"errors"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/observability"
	"github.com/DimKa163/goph-profile/internal/openapi"
	"github.com/DimKa163/goph-profile/internal/rest"
	"github.com/DimKa163/goph-profile/internal/shared/img"
	"github.com/DimKa163/goph-profile/internal/usecase"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/jackc/pgx/v5/pgxpool"
	echootel "github.com/labstack/echo-opentelemetry"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// RunServer starts the HTTP server application.
func RunServer(ctx context.Context, conf config.GophConfig, name, version, buildDate, commit string) error {
	return run(ctx, name, version, func(ctx context.Context) error {
		logger := logging.Logger(ctx)
		pgpool, err := conf.CreatePg(ctx)
		if err != nil {
			logger.Fatal("failed to create postgres pool", zap.Error(err))
		}
		if err = infra.Migrate(pgpool, "./migrations"); err != nil {
			logger.Fatal("failed to migrate", zap.Error(err))
		}
		defer pgpool.Close()

		s3Client, err := conf.CreateS3(ctx)
		if err != nil {
			logger.Fatal("failed to create S3 client", zap.Error(err))
		}
		if err = infra.EnsureBucket(ctx, s3Client, conf.Bucket, conf.Region); err != nil {
			logger.Fatal("failed to ensure bucket", zap.Error(err))
		}
		s3 := infra.NewS3(otel.Tracer("s3"), s3Client, conf.Bucket)

		h, err := NewServer(ctx, name, s3, pgpool)
		if err != nil {
			logger.Fatal("failed to create server", zap.Error(err))
		}
		server := conf.Server(h)
		go func() {
			<-ctx.Done()
			logger.Info("shutting down server...")
			timeoutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
			defer cancel()
			if err = server.Shutdown(timeoutCtx); err != nil {
				logger.Warn("failed to shutdown server", zap.Error(err))
			}
			logger.Info("server shutdown")
		}()
		listenConfig := net.ListenConfig{}

		listener, err := listenConfig.Listen(ctx, "tcp", server.Addr)
		if err != nil {
			logger.Fatal("failed to listen server", zap.String("addr", server.Addr), zap.Error(err))
			return err
		}
		logger.Info("server started",
			zap.String("addr", listener.Addr().String()),
			zap.String("name", name),
			zap.String("version", version),
			zap.String("build_date", buildDate),
			zap.String("commit", commit),
			zap.String("bucket", conf.Bucket),
			zap.String("region", conf.Region),
			zap.String("s3_endpoint", conf.Endpoint),
			zap.Bool("s3_use_ssl", conf.UseSSL),
		)
		if err = server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server failed", zap.Error(err))
			return err
		}
		return nil
	})
}

func newAvatarService(s3client entity.S3, pool *retryablepgxpool.Pool) *usecase.AvatarService {
	return usecase.NewAvatarService(infra.NewTX(pool), infra.NewAvatarRepository(pool),
		infra.NewTaskRepository(pool),
		s3client,
		img.NewCodec())
}

func newUserService(s3client entity.S3, pool *retryablepgxpool.Pool) *usecase.UserService {
	return usecase.NewUserService(infra.NewTX(pool), infra.NewAvatarRepository(pool),
		infra.NewTaskRepository(pool), s3client)
}

func NewServer(ctx context.Context, name string, s3 entity.S3, pgpool *pgxpool.Pool) (http.Handler, error) {
	logger := logging.Logger(ctx)
	retryablePool := retryablepgxpool.New(pgpool)
	if err := retryablePool.Ping(ctx); err != nil {
		return nil, err
	}
	err := observability.UseStorageUsageObserver(name, retryablePool)
	if err != nil {
		return nil, err
	}
	metricService, err := observability.NewMetricService(name)
	if err != nil {
		return nil, err
	}
	uc := rest.NewUserController(newUserService(s3, retryablePool))
	ac := rest.NewAvatarController(metricService, newAvatarService(s3, retryablePool))
	staticDir := webStaticDir()
	web := rest.NewWebController(newUserService(s3, retryablePool), staticDir)
	e := echo.New()
	templates, err := template.ParseGlob(filepath.Join(staticDir, "*.html"))
	if err != nil {
		return nil, err
	}
	e.Renderer = &TemplateRenderer{
		Templates: templates,
	}
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(echootel.NewMiddlewareWithConfig(echootel.Config{
		ServerName:     name,
		TracerProvider: otel.GetTracerProvider(),
		Skipper: func(c *echo.Context) bool {
			return c.Path() == "/health"
		},
		MeterProvider: otel.GetMeterProvider(),
	}))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:       true,
		LogStatus:    true,
		LogMethod:    true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogHost:      true,
		LogUserAgent: true,
		HandleError:  true,
		BeforeNextFunc: func(c *echo.Context) {
			logger := logging.Logger(ctx)
			req := c.Request()
			traceID := trace.SpanFromContext(req.Context()).SpanContext().TraceID()
			fields := []zap.Field{
				zap.String("method", c.Request().Method),
				zap.String("uri", c.Request().RequestURI),
				zap.String("remote_ip", c.RealIP()),
				zap.String("host", c.Request().Host),
				zap.String("user_agent", c.Request().UserAgent()),
				zap.String("trace_id", traceID.String()),
			}
			logger = logger.With(fields...)

			c.SetRequest(req.WithContext(logging.SetLogger(req.Context(), logger)))
		},
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			fields := []zap.Field{
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
			}
			logger := logging.Logger(c.Request().Context())
			if v.Error != nil {
				fields = append(fields, zap.Error(v.Error))
			}

			if v.Status >= 500 {
				logger.Error("request failed", fields...)
			} else if v.Status >= 400 {
				logger.Warn("request client error", fields...)
			} else {
				logger.Info("request processed", fields...)
			}
			return nil
		},
	}))
	e.GET("/health", func(c *echo.Context) error {
		var state struct {
			Server bool `json:"server"`
			Db     bool `json:"db"`
			S3     bool `json:"s3"`
		}
		state.Server = true
		state.Db = true
		state.S3 = true
		if err := retryablePool.Ping(c.Request().Context()); err != nil {
			logger.Error("failed to ping postgres", zap.Error(err))
			state.Db = false
		}
		if err := s3.Check(c.Request().Context()); err != nil {
			logger.Error("failed to check S3 connection", zap.Error(err))
			state.S3 = false
		}
		return c.JSON(http.StatusOK, state)
	})
	e.File("/", filepath.Join(staticDir, "index.html"))
	e.File("/openapi.yaml", openAPIFile())
	webApi := e.Group("/api")
	v1 := webApi.Group("/v1")

	//uc.Register(v1)
	//ac.Register(v1)
	web.Register(e)

	adapter := rest.NewOpenAPIHandler(ac, uc)

	openapi.RegisterHandlers(v1, adapter)
	e.GET("/docs", func(c *echo.Context) error {
		return c.Render(http.StatusOK, "swaggerui.html", map[string]interface{}{})
	})
	return e, nil
}

func webStaticDir() string {
	staticDir := filepath.Join("web", "static")
	if _, err := os.Stat(staticDir); err == nil {
		return staticDir
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return staticDir
	}
	return filepath.Join(filepath.Dir(filepath.Dir(file)), "web", "static")
}

func openAPIFile() string {
	const openAPIFileName = "openapi.yaml"

	candidates := []string{
		filepath.Join("docs", openAPIFileName),
	}

	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "docs", openAPIFileName))
	}

	_, file, _, ok := runtime.Caller(0)
	if ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(filepath.Dir(file)), "docs", openAPIFileName))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return candidates[0]
}

// TemplateRenderer renders HTML templates for Echo.
type TemplateRenderer struct {
	// Templates holds the parsed HTML templates.
	Templates *template.Template
}

// Render executes the named template.
func (r *TemplateRenderer) Render(c *echo.Context, w io.Writer, templateName string, data any) error {
	return r.Templates.ExecuteTemplate(w, templateName, data)
}
