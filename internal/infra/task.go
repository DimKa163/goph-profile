package infra

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/DimKa163/goph-profile/internal/entity"
	"github.com/DimKa163/goph-profile/pkg/retryablepgxpool"
	"github.com/jackc/pgx/v5/pgtype"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	// SelectForProcessingStmt defines the select for processing stmt value.
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
								RETURNING id, type, content, record_id, traceparent, tracestate;`
	// InsertForProcessingStmt defines the insert for processing stmt value.
	InsertForProcessingStmt = `INSERT INTO tasks (id, type, content, record_id, traceparent, tracestate) 
								VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT(id) DO NOTHING;`
	// MarkProcessingStmt defines the mark processing stmt value.
	MarkProcessingStmt = `UPDATE tasks
							SET status = 'completed', updated_at = now()
							WHERE id = ANY($1::text[])`
	// MarkProcessingFailedStmt defines the mark processing failed stmt value.
	MarkProcessingFailedStmt = `UPDATE tasks
									SET error = $2, status = 'failed', updated_at = now(),
									     attempt = attempt + 1
								WHERE id = $1;`
)

type taskRepository struct {
	pool *retryablepgxpool.Pool
}

// NewTaskRepository creates a task repository.
func NewTaskRepository(pool *retryablepgxpool.Pool) *taskRepository {
	return &taskRepository{pool}
}

// GetAll returns tasks available for processing.
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
		var traceParent, traceState sql.NullString
		if err = rows.Scan(
			&t.ID,
			&t.Type,
			&t.Content,
			&t.RecordID,
			&traceParent,
			&traceState,
		); err != nil {
			return nil, err
		}
		if traceParent.Valid {
			t.TraceParent = traceParent.String
		}
		if traceState.Valid {
			t.TraceState = traceState.String
		}
		tasks = append(tasks, &t)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

// Insert stores a new record.
func (r *taskRepository) Insert(ctx context.Context, id string, t entity.TaskType, content []byte) error {
	c := getCon(ctx, r.pool)
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	_, err := c.Exec(
		ctx,
		InsertForProcessingStmt,
		fmt.Sprintf("%s:%s", t, id),
		t.String(),
		content,
		id,
		carrier.Get("traceparent"),
		carrier.Get("tracestate"),
	)
	if err != nil {
		return err
	}
	return nil
}

// MarkCompleted marks tasks as completed.
func (r *taskRepository) MarkCompleted(ctx context.Context, ids []string) error {
	_, err := getCon(ctx, r.pool).Exec(ctx, MarkProcessingStmt, ids)
	if err != nil {
		return err
	}
	return nil
}

// MarkFailed marks a task as failed.
func (r *taskRepository) MarkFailed(ctx context.Context, id string, errMessage string) error {
	_, err := getCon(ctx, r.pool).Exec(ctx, MarkProcessingFailedStmt, id, errMessage)
	if err != nil {
		return err
	}
	return nil
}
