package inbox

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/usecase"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kotel"
	"go.uber.org/zap"
)

type RootHandler func(ctx context.Context, eventType string) (usecase.InboxHandler, error)

type failedResult struct {
	Err    error
	Record *kgo.Record
}

func AvatarUploadedEventWorker(ctx context.Context, tracer *kotel.Tracer, h IdempotencyHandler, root RootHandler, cl *kgo.Client) func() error {
	logger := logging.Logger(ctx)
	clientID := cl.OptValue("ClientID").(string)
	return func() error {
		for {
			fetches := cl.PollFetches(ctx)

			if fetches.IsClientClosed() {
				return io.EOF
			}

			if errs := fetches.Errors(); errs != nil {
				errSlice := make([]error, 0, len(errs))
				for _, fetchErr := range errs {
					if errors.Is(fetchErr.Err, context.Canceled) ||
						errors.Is(fetchErr.Err, kgo.ErrClientClosed) {
						return fetchErr.Err
					}

					logger.Error(
						"kafka fetch error",
						zap.String("topic", fetchErr.Topic),
						zap.Int32("partition", fetchErr.Partition),
						zap.Error(fetchErr.Err),
					)

					errSlice = append(errSlice, fetchErr.Err)
				}
				return errors.Join(errSlice...)
			}
			iter := fetches.RecordIter()
			suc := make([]*kgo.Record, 0)
			failed := make([]failedResult, 0)
			for !iter.Done() {
				record := iter.Next()
				ctx, span := tracer.WithProcessSpan(record)
				ctx = logging.SetLogger(
					ctx,
					logger.With(zap.String(
						"topic",
						record.Topic,
					), zap.Int32(
						"partition",
						record.Partition,
					), zap.ByteString(
						"key",
						record.Key,
					),
					),
				)
				if err := h(ctx, clientID, record, func(ctx context.Context, kind string) error {
					timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					defer cancel()
					handler, err := root(timeoutCtx, kind)
					if err != nil {
						return err
					}
					if err = handler(timeoutCtx, string(record.Key), record.Value); err != nil {
						return err
					}
					return nil
				}); err != nil && !errors.Is(err, ErrAlreadyProcessed) {
					span.RecordError(err)
					failed = append(failed, failedResult{
						Err:    err,
						Record: record,
					})
					continue
				}
				suc = append(suc, record)
				span.End()
			}
			if len(suc) > 0 {
				if err := cl.CommitRecords(ctx, suc...); err != nil {
					return err
				}
			}
			for _, r := range failed {
				logger.Sugar().Error("failed to handle event", zap.Error(r.Err), zap.String("key", string(r.Record.Key)))
				if err := cl.CommitRecords(ctx, r.Record); err != nil {
					return err
				}
			}
		}
	}
}
