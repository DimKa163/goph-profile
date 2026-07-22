// Package logging provides context-aware logger helpers.
package logging

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

// Key is the context key used to store a logger.
var key = loggerKey{}

// Logger returns the logger stored in ctx or a default logger.
func Logger(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(key).(*zap.Logger)
	if !ok || logger == nil {
		return zap.NewNop()
	}
	return logger
}

// SetLogger stores logger in ctx.
func SetLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, key, logger)
}
