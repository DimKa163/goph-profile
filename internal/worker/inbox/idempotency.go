package inbox

import (
	"context"
	"errors"
	"time"

	"github.com/DimKa163/goph-profile/internal/infra"
	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type (
	Transactor interface {
		WithTx(ctx context.Context, fn func(context.Context) error) error
	}
	MessageRepo interface {
		Insert(ctx context.Context, key, consumer string, content []byte) error
	}
)

var ErrAlreadyProcessed = errors.New("already processed")
var ErrEventHeaderIDMissing = errors.New("event header ID missing")
var ErrEventHeaderTypeMissing = errors.New("event header type missing")

type IdempotencyHandler func(ctx context.Context, clientID string, record *kgo.Record, f func(ctx context.Context, kind string) error) error

func Idempotency(tx Transactor, repo MessageRepo) IdempotencyHandler {
	return func(ctx context.Context, clientID string, record *kgo.Record, f func(ctx context.Context, kind string) error) error {
		var eventID string
		var eventType string
		log := logging.Logger(ctx)
		if err := validateRecord(record, &eventID, &eventType); err != nil {
			return err
		}
		log.Info("received message", zap.String("eventType", eventType))
		now := time.Now()
		defer func() {
			duration := time.Since(now)
			log.Info("processed message", zap.Duration("duration", duration), zap.String("eventType", eventType))
		}()
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
