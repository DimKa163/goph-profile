package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel"
	attr "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLoggerProvider(ctx context.Context, attr ...attr.KeyValue) (*sdklog.LoggerProvider, func(), error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return nil, nil, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT not set")
	}

	exporter, err := otlploggrpc.New(
		ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, err
	}

	processor := sdklog.NewBatchProcessor(
		exporter,
		sdklog.WithExportInterval(2*time.Second),
	)
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			attr...,
		),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithContainerID(),
	)
	if err != nil {
		return nil, nil, err
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(processor),
	)

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}

	}
	return provider, shutdown, nil
}

func NewZap(name string, provider *sdklog.LoggerProvider) *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "time"
	encoderCfg.LevelKey = "level"
	encoderCfg.MessageKey = "message"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)

	otelCore := otelzap.NewCore(name, otelzap.WithLoggerProvider(provider))

	core := zapcore.NewTee(stdoutCore, otelCore)

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}
