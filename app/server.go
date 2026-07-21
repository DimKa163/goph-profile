package app

import (
	"context"
	"errors"
	"html/template"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/observability"
	"github.com/DimKa163/goph-profile/internal/rest"
	"github.com/DimKa163/goph-profile/internal/shared/img"
	"github.com/DimKa163/goph-profile/internal/usecase"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// RunServer starts the HTTP server application.
func RunServer(conf config.GophConfig, name, version, buildDate, commit string) error {
	return run(name, version, func(ctx context.Context) error {
		log := logging.Logger(ctx)
		pgpool, err := conf.CreatePg(ctx)
		if err != nil {
			log.Fatal("failed to create postgres pool", zap.Error(err))
			return err
		}
		if err = infra.Migrate(pgpool, "./migrations"); err != nil {
			log.Fatal("failed to migrate", zap.Error(err))
			return err
		}
		defer pgpool.Close()
		retryablePool := retryablepgxpool.New(pgpool)
		if err = retryablePool.Ping(ctx); err != nil {
			log.Fatal("failed to ping postgres", zap.Error(err))
			return err
		}

		s3Client, err := conf.CreateS3(ctx)
		if err != nil {
			log.Fatal("failed to create S3 client", zap.Error(err))
			return err
		}

		err = observability.UseStorageUsageObserver(name, retryablePool)
		if err != nil {
			log.Fatal("failed to create usage observer", zap.Error(err))
			return err
		}

		metricService, err := observability.NewMetricService(name)
		if err != nil {
			log.Fatal("failed to create metric service", zap.Error(err))
			return err
		}

		s3 := infra.NewS3(otel.Tracer("s3"), s3Client, conf.Bucket)

		uc := rest.NewUserController(newUserService(s3, retryablePool))
		ac := rest.NewAvatarController(metricService, newAvatarService(s3, retryablePool))
		web := rest.NewWebController(newUserService(s3, retryablePool))
		e := echo.New()
		e.Renderer = &TemplateRenderer{
			Templates: template.Must(template.ParseGlob("web/static/*.html")),
		}
		e.Use(middleware.Recover())
		e.Use(middleware.RequestID())
		e.Use(otelecho.Middleware(
			name,
			otelecho.WithTracerProvider(otel.GetTracerProvider()),
			otelecho.WithMeterProvider(otel.GetMeterProvider()),
			otelecho.WithSkipper(func(c echo.Context) bool {
				return c.Path() == "/health"
			}),
		))
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
			LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
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
		e.GET("/health", func(c echo.Context) error {
			var state struct {
				Server bool `json:"server"`
				Db     bool `json:"db"`
				S3     bool `json:"s3"`
			}
			state.Server = true
			state.Db = true
			state.S3 = true
			if err := retryablePool.Ping(c.Request().Context()); err != nil {
				log.Error("failed to ping postgres", zap.Error(err))
				state.Db = false
			}
			if err := s3.Check(ctx); err != nil {
				log.Error("failed to check S3 connection", zap.Error(err))
				state.S3 = false
			}
			return c.JSON(http.StatusOK, state)
		})
		e.File("/", "web/static/index.html")
		webApi := e.Group("/api")
		v1 := webApi.Group("/v1")

		uc.Register(v1)
		ac.Register(v1)
		web.Register(e)
		server := conf.Server(e)
		go func() {
			<-ctx.Done()
			log.Info("shutting down server...")
			timeoutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
			defer cancel()
			if err = server.Shutdown(timeoutCtx); err != nil {
				log.Warn("failed to shutdown server", zap.Error(err))
			}
			log.Info("server shutdown")
		}()
		listenConfig := net.ListenConfig{}

		listener, err := listenConfig.Listen(ctx, "tcp", server.Addr)
		if err != nil {
			log.Fatal("failed to listen server", zap.String("addr", server.Addr), zap.Error(err))
			return err
		}
		log.Info("server started",
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
			log.Fatal("server failed", zap.Error(err))
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

// TemplateRenderer renders HTML templates for Echo.
type TemplateRenderer struct {
	// Templates holds the parsed HTML templates.
	Templates *template.Template
}

// Render executes the named template.
func (r *TemplateRenderer) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return r.Templates.ExecuteTemplate(w, name, data)
}
