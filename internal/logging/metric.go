package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	attr "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func InitMetricProvider(ctx context.Context, at ...attr.KeyValue) (func(), error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT not set")
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			at...,
		),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithProcess(),
		//resource.WithContainer(),
		//resource.WithContainerID(),
	)
	if err != nil {
		return nil, err
	}
	metricExporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(
			metric.NewPeriodicReader(
				metricExporter,
				metric.WithInterval(2*time.Second),
			),
		),
	)
	otel.SetMeterProvider(meterProvider)
	return func() {
		if err := meterProvider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}
