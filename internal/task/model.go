package task

import "fmt"

type TaskStatus string

const (
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

type Task struct {
	ID           string      `json:"id"`
	KarboiiTaskID string     `json:"karboii_task_id,omitempty"`
	Type         string      `json:"type"`
	Status       TaskStatus  `json:"status"`
	Message      string      `json:"message,omitempty"`
	CreatedAt    int64       `json:"created_at"`
	UpdatedAt    int64       `json:"updated_at"`
	Data         interface{} `json:"-"`
}

var (
	ErrTaskNotFound = fmt.Errorf("task not found")
)