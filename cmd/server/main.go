package main

import (
	"context"
	"errors"
	"github.com/DimKa163/goph-profile/internal/api"
	"github.com/DimKa163/goph-profile/internal/config"
	"github.com/caarlos0/env"
	"github.com/labstack/echo/v4"
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
