package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	SelectForProcessingStmt = `UPDATE tasks
								SET status = 'processing',
								    updated_at = now()
								WHERE id IN
								(
								    SELECT 
    					   				id
							 		FROM tasks
									WHERE status = 'pending' OR (status = 'processing' AND updated_at < now() - $1::interval) OR (status = 'failed' AND attempt < 4)
									ORDER BY created_at ASC
									LIMIT $2 FOR UPDATE SKIP LOCKED
								)
								RETURNING id, type, content, record_id;`
	InsertForProcessingStmt = `INSERT INTO tasks (id, type, content, record_id) VALUES ($1, $2, $3, $4) ON CONFLICT(id) DO NOTHING;`
	MarkProcessingStmt      = `UPDATE tasks
							SET status = 'completed', updated_at = now()
							WHERE id = ANY($1::text[])`
	MarkProcessingFailedStmt = `UPDATE tasks
									SET error = $2, status = 'failed', updated_at = now(),
									     attempt = attempt + 1
								WHERE id = $1;`
)

type taskRepository struct {
	pool *retryablepgxpool.Pool
}

func NewTaskRepository(pool *retryablepgxpool.Pool) *taskRepository {
	return &taskRepository{pool}
}

func (r *taskRepository) GetAll(ctx context.Context, ttl time.Duration, limit int) ([]*entity.Task, error) {
	c := getCon(ctx, r.pool)
	rows, err := c.Query(ctx, SelectForProcessingStmt, pgtype.Interval{
		Microseconds: ttl.Microseconds(),
		Valid:        true,
	}, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tasks := make([]*entity.Task, 0, limit)
	for rows.Next() {
		var t entity.Task
		if err = rows.Scan(&t.ID, &t.Type, &t.Content, &t.RecordID); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *taskRepository) Insert(ctx context.Context, id string, t entity.TaskType, content []byte) error {
	c := getCon(ctx, r.pool)
	_, err := c.Exec(ctx, InsertForProcessingStmt, fmt.Sprintf("%s:%s", t, id), t.String(), content, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *taskRepository) MarkCompleted(ctx context.Context, ids []string) error {
	_, err := getCon(ctx, r.pool).Exec(ctx, MarkProcessingStmt, ids)
	if err != nil {
		return err
	}
	return nil
}

func (r *taskRepository) MarkFailed(ctx context.Context, id string, errMessage string) error {
	_, err := getCon(ctx, r.pool).Exec(ctx, MarkProcessingFailedStmt, id, errMessage)
	if err != nil {
		return err
	}
	return nil
}
