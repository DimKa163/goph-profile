package observability

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/DimKa163/goph-profile/internal/logging"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel"
	attr "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ObsConfiger applies observability configuration.
type ObsConfiger func(*observeConfiguration)

// WithZapEncoderConfig applies a Zap encoder configuration.
func WithZapEncoderConfig(config zapcore.EncoderConfig, layer ...func(encoderConfig *zapcore.EncoderConfig)) ObsConfiger {
	return func(o *observeConfiguration) {
		o.EncoderConfig = config
		o.EncoderConfig.TimeKey = "time"
		o.EncoderConfig.LevelKey = "level"
		o.EncoderConfig.MessageKey = "message"
		o.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		if len(layer) > 0 {
			for _, l := range layer {
				l(&o.EncoderConfig)
			}
		}
	}
}

// WithResources applies OpenTelemetry resource options.
func WithResources(res ...resource.Option) ObsConfiger {
	return func(o *observeConfiguration) {
		o.ResourceOptions = append(o.ResourceOptions, res...)
	}
}

// WithAttributes applies OpenTelemetry resource attributes.
func WithAttributes(attribute ...attr.KeyValue) ObsConfiger {
	return func(o *observeConfiguration) {
		o.ResourceOptions = append(o.ResourceOptions, resource.WithAttributes(attribute...))
	}
}

type observeConfiguration struct {
	// ApplicationName stores the application name value.
	ApplicationName string
	// ExporterEndpoint stores the exporter endpoint value.
	ExporterEndpoint string
	// EncoderConfig stores the encoder config value.
	EncoderConfig zapcore.EncoderConfig
	// ResourceOptions stores the resource options value.
	ResourceOptions []resource.Option
	// Resource stores the resource value.
	Resource *resource.Resource
}

// Init initializes logging, tracing, metrics, and profiling.
func Init(ctx context.Context, name string, opt ...ObsConfiger) (context.Context, func(), error) {
	var config observeConfiguration
	config.ApplicationName = name
	for _, o := range opt {
		o(&config)
	}
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return nil, nil, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT not set")
	}
	config.ExporterEndpoint = endpoint

	res, err := resource.New(
		ctx,
		config.ResourceOptions...,
	)
	if err != nil {
		return nil, nil, err
	}
	config.Resource = res

	logger, logProviderSh, err := createCoreLogger(ctx, config)
	if err != nil {
		return nil, nil, err
	}
	ctx = logging.SetLogger(ctx, logger)

	meterProviderSh, err := createMetricProvider(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	traceProviderSh, err := createTraceProvider(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	return ctx, func() {
		logProviderSh()
		meterProviderSh()
		traceProviderSh()
	}, nil

}

func createCoreLogger(ctx context.Context, o observeConfiguration) (*zap.Logger, func(), error) {
	exporter, err := otlploggrpc.New(
		ctx,
		otlploggrpc.WithEndpoint(o.ExporterEndpoint),
		otlploggrpc.WithInsecure(),
		otlploggrpc.WithCompressor("gzip"),
	)
	if err != nil {
		return nil, nil, err
	}
	processor := sdklog.NewBatchProcessor(
		exporter,
		sdklog.WithExportInterval(2*time.Second),
	)

	encoderCfg := o.EncoderConfig

	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)

	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(o.Resource),
		sdklog.WithProcessor(processor),
	)

	otelCore := otelzap.NewCore(o.ApplicationName, otelzap.WithLoggerProvider(provider))

	core := zapcore.NewTee(stdoutCore, otelCore)

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}

func createMetricProvider(ctx context.Context, o observeConfiguration) (func(), error) {
	metricExporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(o.ExporterEndpoint),
		otlpmetricgrpc.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}

	provider := metric.NewMeterProvider(
		metric.WithResource(o.Resource),
		metric.WithReader(
			metric.NewPeriodicReader(
				metricExporter,
				metric.WithInterval(2*time.Second),
			),
		),
	)
	otel.SetMeterProvider(provider)
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}

func createTraceProvider(ctx context.Context, o observeConfiguration) (func(), error) {
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(o.ExporterEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(o.Resource),
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := tp.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}
