package observability

import (
	"context"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type MetricStatus string

var (
	Success MetricStatus = "success"
	Failure MetricStatus = "failure"
)

//go:generate mockgen -source=metric.go -destination=mocks/mock_metric.go -package=mocks
type MetricService interface {
	AvatarUploaded(ctx context.Context, userID entity.Email, status MetricStatus)
	AvatarUploadDuration(ctx context.Context, status MetricStatus, duration time.Duration)
	AvatarProcessingDuration(ctx context.Context, status MetricStatus, kind string, duration time.Duration)
}

type metricService struct {
	avatarUploadsTotal   metric.Int64Counter
	uploadDurationServer metric.Float64Histogram
	processingDuration   metric.Float64Histogram
}

func (m *metricService) AvatarUploaded(ctx context.Context, userID entity.Email, status MetricStatus) {
	m.avatarUploadsTotal.Add(
		ctx,
		1,
		metric.WithAttributes(attribute.String(
			"user_id",
			userID.String(),
		), attribute.String(
			"status",
			string(status),
		)),
	)
}

func (m *metricService) AvatarUploadDuration(ctx context.Context, status MetricStatus, duration time.Duration) {
	m.uploadDurationServer.Record(
		ctx,
		duration.Seconds(),
		metric.WithAttributes(
			attribute.String("status", string(status)),
		),
	)
}

func (m *metricService) AvatarProcessingDuration(ctx context.Context, status MetricStatus, kind string, duration time.Duration) {
	m.processingDuration.Record(
		ctx,
		duration.Seconds(),
		metric.WithAttributes(
			attribute.String("kind", kind),
			attribute.String("status", string(status)),
		),
	)
}

func NewMetricService(name string) (MetricService, error) {
	meter := otel.Meter(name)

	counter, err := meter.Int64Counter(
		"avatars_uploads_total",
		metric.WithDescription("Total number of avatar uploads"),
	)
	if err != nil {
		return nil, err
	}

	uploadDurationServer, err := meter.Float64Histogram(
		"avatars_upload_duration_seconds",
		metric.WithDescription("Avatar upload duration"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.01,
			0.05,
			0.1,
			0.25,
			0.5,
			1,
			2.5,
			5,
			10,
			30,
		),
	)
	if err != nil {
		return nil, err
	}
	processingDuration, err := meter.Float64Histogram(
		"avatars_processing_duration_seconds",
		metric.WithDescription("Avatar process duration"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.01,
			0.05,
			0.1,
			0.25,
			0.5,
			1,
			2.5,
			5,
			10,
			30,
		),
	)
	if err != nil {
		return nil, err
	}
	return &metricService{
		avatarUploadsTotal:   counter,
		uploadDurationServer: uploadDurationServer,
		processingDuration:   processingDuration,
	}, nil
}
