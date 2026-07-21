// Package retryablepgxpool wraps pgxpool with retry behavior.
package retryablepgxpool

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RetryConfig configures retry behavior for a PostgreSQL pool.
type RetryConfig struct {
	// MaxAttempts stores the max attempts value.
	MaxAttempts uint64
	// InitialInterval stores the initial interval value.
	InitialInterval time.Duration
	// MaxInterval stores the max interval value.
	MaxInterval time.Duration
	// Multiplier stores the multiplier value.
	Multiplier float64
	// RandomizationFactor stores the randomization factor value.
	RandomizationFactor float64
	// MaxElapsedTime stores the max elapsed time value.
	MaxElapsedTime time.Duration
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		InitialInterval:     2 * time.Second,
		MaxInterval:         15 * time.Second,
		Multiplier:          2,
		RandomizationFactor: 0.2,
		MaxElapsedTime:      30 * time.Second,
		MaxAttempts:         3,
	}
}

// Pool wraps a PostgreSQL pool with retry behavior.
type Pool struct {
	*pgxpool.Pool
	retryConfig *RetryConfig
}

// Exec executes a query with retry behavior.
func (r *Pool) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return backoff.RetryWithData(func() (pgconn.CommandTag, error) {
		tg, err := r.Pool.Exec(ctx, query, args...)
		if err != nil {
			return pgconn.CommandTag{}, retryableError(err)
		}
		return tg, nil
	}, r.withPolicy(ctx))
}

// Query executes a query with retry behavior.
func (r *Pool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return backoff.RetryWithData(func() (pgx.Rows, error) {
		rows, err := r.Pool.Query(ctx, sql, args...)
		if err != nil {
			return nil, retryableError(err)
		}
		return rows, nil
	}, r.withPolicy(ctx))
}

// QueryRow executes a single-row query with retry behavior.
func (r *Pool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return &retryRow{
		ctx:         ctx,
		retryPolicy: r.withPolicy(ctx),
		sql:         sql,
		args:        args,
		query:       r.Pool.QueryRow,
	}
}

// Ping verifies the database connection with retry behavior.
func (r *Pool) Ping(ctx context.Context) error {
	return backoff.Retry(func() error {
		err := r.Pool.Ping(ctx)
		if err != nil {
			return retryableError(err)
		}
		return nil
	}, r.withPolicy(ctx))
}

// Tx executes a transaction with retry behavior.
func (r *Pool) Tx(ctx context.Context, h func(tx pgx.Tx) error, opts pgx.TxOptions) error {
	return backoff.Retry(func() error {
		tx, err := r.BeginTx(ctx, opts)
		if err != nil {
			return retryableError(err)
		}

		err = h(tx)
		if err != nil {
			_ = tx.Rollback(ctx)
			return retryableError(err)
		}

		if err = tx.Commit(ctx); err != nil {
			return backoff.Permanent(err)
		}
		return nil
	}, r.withPolicy(ctx))
}

func (r *Pool) withPolicy(ctx context.Context) backoff.BackOff {
	e := backoff.NewExponentialBackOff()
	e.InitialInterval = r.retryConfig.InitialInterval
	e.MaxInterval = r.retryConfig.MaxInterval
	e.MaxElapsedTime = r.retryConfig.MaxElapsedTime
	e.Multiplier = r.retryConfig.Multiplier
	e.RandomizationFactor = r.retryConfig.RandomizationFactor
	maxRetries := uint64(0)
	if r.retryConfig.MaxAttempts > 0 {
		maxRetries = r.retryConfig.MaxAttempts - 1
	}
	return backoff.WithContext(backoff.WithMaxRetries(e, maxRetries), ctx)
}

// New creates an outbox worker.
func New(pool *pgxpool.Pool, retryConfig ...*RetryConfig) *Pool {
	if len(retryConfig) == 0 || retryConfig[0] == nil {
		retryConfig = []*RetryConfig{
			DefaultRetryConfig(),
		}
	}
	return &Pool{
		Pool:        pool,
		retryConfig: retryConfig[0],
	}
}

type retryRow struct {
	ctx         context.Context
	retryPolicy backoff.BackOff
	sql         string
	args        []any
	query       func(context.Context, string, ...any) pgx.Row
}

// Scan reads a database value into the receiver.
func (r *retryRow) Scan(dest ...any) error {
	return backoff.Retry(func() error {
		err := r.query(r.ctx, r.sql, r.args...).Scan(dest...)
		if err != nil {
			return retryableError(err)
		}
		return err
	}, r.retryPolicy)
}

func retryableError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return backoff.Permanent(err)
	}
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if ok && shouldRetry(pgErr) {
		return err
	}
	if pgconn.SafeToRetry(err) {
		return err
	}

	return backoff.Permanent(err)
}

func shouldRetry(pgErr *pgconn.PgError) bool {
	switch pgErr.Code {
	case pgerrcode.SerializationFailure,
		pgerrcode.DeadlockDetected,
		pgerrcode.TooManyConnections,
		pgerrcode.LockNotAvailable,
		pgerrcode.CannotConnectNow,
		pgerrcode.QueryCanceled:
		return true
	}
	return false
}
