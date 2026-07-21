package entity

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TaskStatus identifies an outbox task lifecycle state.
type TaskStatus int

const (
	// Created is the initial task status.
	Created TaskStatus = iota
	// PendingTaskStatus marks a task waiting for processing.
	PendingTaskStatus
	// ProcessingTaskStatus marks a task being processed.
	ProcessingTaskStatus
	// CompletedTaskStatus marks a successfully processed task.
	CompletedTaskStatus
)

// String returns the string representation.
func (s TaskStatus) String() string {
	return [...]string{"created", "pending", "processing", "completed"}[s]
}

// Scan reads a database value into the receiver.
func (s *TaskStatus) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("task status can not be nil")
	}
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("task status can not be scanned")
	}
	switch str {
	case "created":
		*s = Created
	case "pending":
		*s = PendingTaskStatus
	case "processing":
		*s = ProcessingTaskStatus
	case "completed":
		*s = CompletedTaskStatus
	}
	return nil
}

// TaskType identifies the kind of outbox task.
type TaskType string

const (
	// AvatarUploaded is the task type for uploaded avatar events.
	AvatarUploaded TaskType = "uploaded"
	// AvatarDeleted is the task type for deleted avatar events.
	AvatarDeleted TaskType = "deleted"
)

// String returns the string representation.
func (t TaskType) String() string {
	return string(t)
}

// Scan reads a database value into the receiver.
func (t *TaskType) Scan(value any) error {
	if value == nil {
		return fmt.Errorf("task type can not be nil")
	}
	var str string
	switch v := value.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("task type can not be scanned")
	}
	switch str {
	case "uploaded":
		*t = AvatarUploaded
	case "deleted":
		*t = AvatarDeleted
	}
	return nil
}

// Task defines task.
type Task struct {
	// ID stores the identifier.
	ID string
	// CreatedAt stores the created at value.
	CreatedAt time.Time
	// UpdatedAt stores the updated at value.
	UpdatedAt time.Time
	// Type stores the type value.
	Type TaskType
	// Status stores the status value.
	Status TaskStatus
	// Content stores the content value.
	Content []byte
	// RecordID stores the Kafka record identifier.
	RecordID string
	// Error stores the error value.
	Error string
	// TraceParent stores the trace parent value.
	TraceParent string
	// TraceState stores the trace state value.
	TraceState string
}

// Trace extracts the task trace context into ctx.
func (t *Task) Trace(ctx context.Context) context.Context {
	if t.TraceParent == "" {
		return ctx
	}

	carrier := propagation.MapCarrier{
		"traceparent": t.TraceParent,
	}

	if t.TraceState != "" {
		carrier.Set("tracestate", t.TraceState)
	}

	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// TaskRepository describes persistence operations for outbox tasks.
//
//go:generate mockgen -source=task.go -destination=mocks/mock_task.go -package=mocks
type TaskRepository interface {
	// Insert stores a new record.
	Insert(ctx context.Context, id string, t TaskType, content []byte) error
	// GetAll returns tasks available for processing.
	GetAll(ctx context.Context, ttl time.Duration, limit int) ([]*Task, error)
	// MarkCompleted marks tasks as completed.
	MarkCompleted(ctx context.Context, tasks []string) error
	// MarkFailed marks a task as failed.
	MarkFailed(ctx context.Context, id string, errMessage string) error
}
