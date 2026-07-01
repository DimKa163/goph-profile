package worker

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/handlers"
	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/DimKa163/goph-profile/internal/logging"
	"github.com/DimKa163/goph-profile/internal/tracing"
	"github.com/beevik/guid"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

func AvatarUploadedEventWorker(ctx context.Context, h handlers.UploaderHandler, cl *kgo.Client) func() error {
	logger := logging.Logger(ctx)
	return func() error {
		for {
			fetches := cl.PollFetches(ctx)

			timeoutCtx, cancel := context.WithTimeout(ctx, 30000*time.Second)
			if fetches.IsClientClosed() {
				cancel()
				return io.EOF
			}

			if errs := fetches.Errors(); errs != nil {
				errSlice := make([]error, len(errs))
				for i, err := range errSlice {
					errSlice[i] = err
				}
				cancel()
				return errors.Join(errSlice...)
			}
			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()
				headers := kafka.NewHeaders(record.Headers...)
				correlationID, ok := headers.Value(tracing.CorrelationIDHeader)
				if !ok {
					correlationID = guid.NewString()
				}
				log := logger.With(
					zap.String("topic", record.Topic),
					zap.Int32("partition", record.Partition),
					zap.Int64("offset", record.Offset),
					zap.String("correlation_id", correlationID),
				)
				log.Debug("received message")
				timeoutCtx = logging.SetLogger(timeoutCtx, log)
				var aue entity.AvatarUploadedEvent
				if err := aue.Read(record.Value); err != nil {
					panic(err)
				}
				if err := h(timeoutCtx, &aue); err != nil {
					panic(err)
				}
				if err := cl.CommitRecords(timeoutCtx, record); err != nil {
					panic(err)
				}
			}
			cancel()
		}
	}
}
