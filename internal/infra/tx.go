package infra

import (
	"context"

	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type txContextKey struct{}

var txKey = txContextKey{}

type pool interface {
	// Exec executes a query with retry behavior.
	Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error)
	// Query executes a query with retry behavior.
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	// QueryRow executes a single-row query with retry behavior.
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	// Begin starts a transaction.
	Begin(ctx context.Context) (pgx.Tx, error)
}

type transactor struct {
	pool *retryablepgxpool.Pool
}

// NewTX creates a transaction manager.
func NewTX(pool *retryablepgxpool.Pool) *transactor {
	return &transactor{
		pool: pool,
	}
}

// WithTx executes fn inside a transaction.
func (x *transactor) WithTx(ctx context.Context, fn func(context.Context) error) error {
	tx, ok := ctx.Value(txKey).(pool)
	if !ok {
		return x.pool.Tx(ctx, func(tx pgx.Tx) error {
			return fn(context.WithValue(ctx, txKey, tx))
		}, pgx.TxOptions{})
	}
	var err error
	subTx, err := tx.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = subTx.Rollback(ctx)
		} else {
			_ = subTx.Commit(ctx)
		}
	}()
	if err = fn(context.WithValue(ctx, txKey, subTx)); err != nil {
		return err
	}
	return nil
}

func getCon(ctx context.Context, p *retryablepgxpool.Pool) pool {
	tx, ok := ctx.Value(txKey).(pool)
	if !ok {
		tx = p
	}
	return tx
}
