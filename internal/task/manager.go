package task

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskManager struct {
	mu          sync.RWMutex
	tasks       map[string]*Task
	taskTimeout time.Duration
	cleanupTick *time.Ticker
	stopChan    chan struct{}
}

func NewTaskManager(taskTimeout time.Duration) *TaskManager {
	tm := &TaskManager{
		tasks:       make(map[string]*Task),
		taskTimeout: taskTimeout,
		cleanupTick: time.NewTicker(1 * time.Minute),
		stopChan:    make(chan struct{}),
	}
	go tm.cleanup()
	return tm
}

func (tm *TaskManager) Create(taskType string, data interface{}) string {
	id := uuid.New().String()
	now := time.Now()
	task := &Task{
		ID:        id,
		Type:      taskType,
		Status:    TaskRunning,
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
		Data:      data,
	}
	tm.mu.Lock()
	tm.tasks[id] = task
	tm.mu.Unlock()
	return id
}

func (tm *TaskManager) Update(id string, status TaskStatus, message string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if t, ok := tm.tasks[id]; ok {
		t.Status = status
		t.Message = message
		t.UpdatedAt = time.Now().Unix()
		return nil
	}
	return ErrTaskNotFound
}

func (tm *TaskManager) Get(id string) (*Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	if t, ok := tm.tasks[id]; ok {
		return t, nil
	}
	return nil, ErrTaskNotFound
}

func (tm *TaskManager) cleanup() {
	for {
		select {
		case <-tm.stopChan:
			return
		case <-tm.cleanupTick.C:
			tm.mu.Lock()
			now := time.Now().Unix()
			for id, t := range tm.tasks {
				if t.Status != TaskRunning &&
					(now-t.UpdatedAt) > int64(tm.taskTimeout.Seconds()) {
					delete(tm.tasks, id)
				}
			}
			tm.mu.Unlock()
		}
	}
}

func (tm *TaskManager) Close() {
	tm.cleanupTick.Stop()
	close(tm.stopChan)
}