// Package app wires and starts the profile service applications.
package app

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/DimKa163/goph-profile/internal/observability"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

func run(name, version string, configurator func(ctx context.Context) error) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	ctx, cleanup, err := observability.Init(
		ctx,
		name,
		observability.WithResources(
			resource.WithFromEnv(),
			resource.WithTelemetrySDK(),
			resource.WithOS(),
			resource.WithHost(),
			resource.WithProcess(),
			resource.WithContainer(),
			resource.WithContainerID(),
		),
		observability.WithAttributes(
			semconv.ServiceNameKey.String(name),
			semconv.ServiceVersionKey.String(version),
		),
		observability.WithZapEncoderConfig(
			zap.NewDevelopmentEncoderConfig(),
		),
	)
	if err != nil {
		return err
	}
	defer cleanup()
	return configurator(ctx)
}
