package infra

import (
	"context"

	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
)

const (
	InsertInboxStmt = `INSERT INTO inbox(id, consumer, content) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;`
)

type inboxRepository struct {
	pool *retryablepgxpool.Pool
}

func NewInboxRepo(pool *retryablepgxpool.Pool) *inboxRepository {
	return &inboxRepository{
		pool: pool,
	}
}

func (r *inboxRepository) Insert(ctx context.Context, key, consumer string, content []byte) error {
	c := getCon(ctx, r.pool)
	tag, err := c.Exec(ctx, InsertInboxStmt, key, consumer, content)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}
