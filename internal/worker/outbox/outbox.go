package outbox

import (
	"context"
	"sync"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/internal/infra/kafka"
	"github.com/DimKa163/goph-profile/internal/logging"
	"go.uber.org/zap"
)

type (
	Transactor interface {
		WithTx(ctx context.Context, fn func(context.Context) error) error
	}
	Outbox interface {
		Start(ctx context.Context, workers []kafka.Producer, batchSize int, waitTime, ttl time.Duration)
	}
)

var _ Outbox = (*outboxImpl)(nil)

type TypeHandler func(ctx context.Context, key []byte, buffer []byte, headers ...kafka.Header) error

type RootHandler func(t entity.TaskType) (TypeHandler, error)

type outboxImpl struct {
	transactor Transactor
	taskRepo   entity.TaskRepository
}

func New(tx Transactor, taskRepo entity.TaskRepository) *outboxImpl {
	return &outboxImpl{
		transactor: tx,
		taskRepo:   taskRepo,
	}
}

func (o *outboxImpl) Start(ctx context.Context, workers []kafka.Producer, batchSize int, waitTime, ttl time.Duration) {
	var wg sync.WaitGroup
	for w := 0; len(workers) > w; w++ {
		wg.Add(1)
		go o.worker(ctx, &wg, workers[w], batchSize, waitTime, ttl)
	}
	logger := logging.Logger(ctx)
	logger.Info("starting outbox", zap.Int("workers", len(workers)))
	wg.Wait()
}

func (o *outboxImpl) worker(ctx context.Context, wg *sync.WaitGroup, producer kafka.Producer, batchSize int, waitTime, ttl time.Duration) {
	defer wg.Done()
	ticker := time.NewTicker(waitTime)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger := logging.Logger(ctx)
			if err := o.transactor.WithTx(ctx, func(ctx context.Context) error {
				m, err := o.taskRepo.GetAll(ctx, ttl, batchSize)
				if err != nil {
					return err
				}
				if len(m) == 0 {
					return nil
				}
				logger.Sugar().Infof("Processing %d tasks", len(m))

				succeed := make([]string, 0, len(m))
				failed := make([]taskErrorDescription, 0)
				for _, task := range m {
					if err = producer.Produce(ctx, &kafka.Message{
						Topic:       "avatar",
						Key:         []byte(task.RecordID),
						Body:        task.Content,
						ContentType: "application/json",
						EventID:     task.ID,
						TaskType:    task.Type,
					}); err != nil {
						logger.Error("failed to produce", zap.Error(err))
						failed = append(failed, taskErrorDescription{
							Error: err,
							ID:    task.ID,
						})
						continue
					}
					succeed = append(succeed, task.ID)
				}

				if len(succeed) > 0 {
					if err = o.taskRepo.MarkCompleted(ctx, succeed); err != nil {
						return err
					}
				}

				for _, taskError := range failed {
					if err = o.taskRepo.MarkFailed(ctx, taskError.ID, taskError.Error.Error()); err != nil {
						return err
					}
				}

				logger.Sugar().Infof("successfully processed: %d; processed with error: %d", len(succeed), len(failed))
				return nil
			}); err != nil {
				logger.Error("outbox iteration failed", zap.Error(err))
			}

		}
	}
}

type taskErrorDescription struct {
	Error error  `json:"error"`
	ID    string `json:"id"`
}
