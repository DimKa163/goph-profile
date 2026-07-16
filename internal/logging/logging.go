package logging

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

var Key = loggerKey{}

func Logger(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(Key).(*zap.Logger)
	if !ok || logger == nil {
		return zap.NewNop()
	}
	return logger
}

func SetLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, Key, logger)
}
