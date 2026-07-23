// Package inbox contains inbox worker processing and idempotency helpers.
package inbox

import (
	"context"
	"errors"
	"time"

	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/observability"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type (
	// Transactor describes transactional execution.
	Transactor interface {
		// WithTx executes fn inside a transaction.
		WithTx(ctx context.Context, fn func(context.Context) error) error
	}
	// MessageRepo defines message repo.
	MessageRepo interface {
		// Insert stores a new record.
		Insert(ctx context.Context, key, consumer string, content []byte) error
	}
)

// ErrAlreadyProcessed is returned when an event was already handled.
var ErrAlreadyProcessed = errors.New("already processed")

// ErrEventHeaderIDMissing is returned when the event ID header is absent.
var ErrEventHeaderIDMissing = errors.New("event header ID missing")

// ErrEventHeaderTypeMissing is returned when the event type header is absent.
var ErrEventHeaderTypeMissing = errors.New("event header type missing")

// IdempotencyHandler wraps event handling with idempotency checks.
type IdempotencyHandler func(ctx context.Context, clientID string, record *kgo.Record, f func(ctx context.Context, kind string) error) error

// Idempotency wraps inbox processing with idempotency checks.
func Idempotency(tx Transactor, metricService observability.MetricService, repo MessageRepo) IdempotencyHandler {
	return func(ctx context.Context, clientID string, record *kgo.Record, f func(ctx context.Context, kind string) error) error {
		var eventID string
		var eventType string
		var err error
		log := logging.Logger(ctx)
		now := time.Now()
		defer func() {
			duration := time.Since(now)
			log.Info("processed message",
				zap.Duration("duration", duration),
				zap.String("eventType", eventType),
				zap.String("clientID", clientID),
				zap.String("eventID", eventID),
				zap.String("topic", record.Topic),
			)
			status := observability.Success
			if err != nil {
				status = observability.Failure
			}
			metricService.AvatarProcessingDuration(ctx, status, eventType, duration)
		}()
		if err = validateRecord(record, &eventID, &eventType); err != nil {
			return err
		}
		log.Info("received message", zap.String("eventType", eventType))
		span := trace.SpanFromContext(ctx)
		span.AddEvent("avatar received")
		span.SetAttributes(
			attribute.String("client_id", clientID),
			attribute.String("topic", record.Topic),
			attribute.Int("partition", int(record.Partition)),
			attribute.Int("offset", int(record.Offset)),
			attribute.String("event_id", eventID),
			attribute.String("event_type", eventType),
		)
		return tx.WithTx(ctx, func(ctx context.Context) error {
			if err := repo.Insert(ctx, eventID, clientID, record.Value); err != nil {
				if errors.Is(err, infra.ErrNoRows) {
					return ErrAlreadyProcessed
				}
				return err
			}
			if err := f(ctx, eventType); err != nil {
				return err
			}
			return nil
		})
	}
}

func validateRecord(record *kgo.Record, eventID, eventType *string) error {
	headers := kafka.NewHeaders(record.Headers...)
	var ok bool
	*eventID, ok = headers.Value(kafka.EventIDHeaderKey)
	if !ok {
		return ErrEventHeaderIDMissing
	}
	*eventType, ok = headers.Value(kafka.EventTypeHeaderKey)
	if !ok {
		return ErrEventHeaderTypeMissing
	}
	return nil
}
