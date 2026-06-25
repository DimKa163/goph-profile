package main

import (
	"context"
	"errors"
	"github.com/DimKa163/goph-profile/internal/api"
	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/caarlos0/env"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"net/http"
	"os/signal"
	"syscall"
	"time"
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

			if v.Error != nil {
				fields = append(fields, zap.Error(v.Error))
			}

			if v.Status >= 500 {
				logger.Error("request failed", fields...)
			} else if v.Status >= 400 {
				logger.Warn("request client error", fields...)
			} else {
				logger.Info("request", fields...)
			}
			c.Request().WithContext(context.WithValue(c.Request().Context(), "logger", logger))
			return nil
		},
	}))
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, struct {
			Server string `json:"server"`
		}{
			Server: "Ok",
		})
	})
	webApi := e.Group("/api")
	v1 := webApi.Group("/v1")
	uc := api.NewUserController()
	uc.Register(v1)
	ac := api.NewAvatarController()
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
