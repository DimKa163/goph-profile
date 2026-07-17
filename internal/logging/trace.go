package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	attr "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewTracerProvider(ctx context.Context, at ...attr.KeyValue) (func(), error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT not set")
	}
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			at...,
		),
		resource.WithOS(),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithContainerID(),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithBatchTimeout(2*time.Second),
		),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := tp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}
