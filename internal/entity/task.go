package entity

import (
	"context"
	"fmt"
	"time"
)

type TaskStatus int

const (
	Created TaskStatus = iota
	PendingTaskStatus
	ProcessingTaskStatus
	CompletedTaskStatus
)

func (s TaskStatus) String() string {
	return [...]string{"created", "pending", "processing", "completed"}[s]
}

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

type TaskType string

const (
	AvatarUploaded TaskType = "uploaded"
	AvatarDeleted  TaskType = "deleted"
)

func (t TaskType) String() string {
	return string(t)
}

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

type Task struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Type      TaskType
	Status    TaskStatus
	Content   []byte
	RecordID  string
	Error     string
}

//go:generate mockgen -source=task.go -destination=mocks/mock_task.go -package=mocks
type TaskRepository interface {
	Insert(ctx context.Context, id string, t TaskType, content []byte) error
	GetAll(ctx context.Context, ttl time.Duration, limit int) ([]*Task, error)
	MarkCompleted(ctx context.Context, tasks []string) error
	MarkFailed(ctx context.Context, id string, errMessage string) error
}
