package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"agent/internal/logger"
	"agent/internal/task"
)

type StatusUpdate struct {
	Status string                 `json:"status"`
	Result map[string]interface{} `json:"result,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

type Service struct {
	endpoint  string
	token     string
	projectID string
	taskManager *task.TaskManager
	logger    *logger.Logger
	stopChan  chan struct{}
}

func New(endpoint, token, projectID string, taskManager *task.TaskManager, logger *logger.Logger) *Service {
	return &Service{
		endpoint:    endpoint,
		token:       token,
		projectID:   projectID,
		taskManager: taskManager,
		logger:      logger,
		stopChan:    make(chan struct{}),
	}
}

func (s *Service) Start() {
	s.logger.Info("starting status reporter service")
	// Периодически проверяем завершенные задачи и отправляем статус
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("stopping status reporter")
			return
		case <-ticker.C:
			s.checkAndReportCompletedTasks()
		}
	}
}

func (s *Service) Stop() {
	close(s.stopChan)
}

func (s *Service) checkAndReportCompletedTasks() {
	// Получаем все задачи и проверяем завершенные с KarboiiTaskID
	// Поскольку TaskManager не имеет метода для получения всех задач,
	// нужно добавить его или использовать другой подход

	// Для простоты, будем полагаться на то, что после обновления статуса
	// вызывается метод ReportStatus
}

func (s *Service) ReportStatus(karboiiTaskID string, status task.TaskStatus, message string, result map[string]interface{}) {
	if karboiiTaskID == "" {
		return // Не Karboii задача
	}

	url := fmt.Sprintf("%s/%s/tasks/%s/status", s.endpoint, s.projectID, karboiiTaskID)

	update := StatusUpdate{
		Status: string(status),
		Result: result,
	}
	if status == task.TaskFailed {
		update.Error = message
	}

	data, err := json.Marshal(update)
	if err != nil {
		s.logger.Error("failed to marshal status update: %v", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		s.logger.Error("failed to create status request: %v", err)
		return
	}

	req.Header.Set("X-Auth-Token", s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("failed to send status update: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.logger.Info("successfully reported status for task %s: %s", karboiiTaskID, status)
	} else {
		s.logger.Error("failed to report status for task %s, status code: %d", karboiiTaskID, resp.StatusCode)
	}
}