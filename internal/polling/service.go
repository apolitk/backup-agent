package polling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"agent/internal/archiver"
	"agent/internal/logger"
	"agent/internal/storage"
	"agent/internal/task"
)

type KarboiiTask struct {
	TaskID    string                 `json:"task_id"`
	Type      string                 `json:"type"`
	Params    map[string]interface{} `json:"params"`
}

type Service struct {
	endpoint      string
	token         string
	projectID     string
	interval      time.Duration
	taskManager   *task.TaskManager
	s3Client      storage.S3Client
	archiver      archiver.Archiver
	tempDir       string
	logger        *logger.Logger
	stopChan      chan struct{}
}

func New(endpoint, token, projectID string, interval time.Duration, taskManager *task.TaskManager, s3Client storage.S3Client, archiver archiver.Archiver, tempDir string, logger *logger.Logger) *Service {
	return &Service{
		endpoint:    endpoint,
		token:       token,
		projectID:   projectID,
		interval:    interval,
		taskManager: taskManager,
		s3Client:    s3Client,
		archiver:    archiver,
		tempDir:     tempDir,
		logger:      logger,
		stopChan:    make(chan struct{}),
	}
}

func (s *Service) Start() {
	s.logger.Info("starting polling service for Karboii endpoint: %s", s.endpoint)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			s.logger.Info("stopping polling service")
			return
		case <-ticker.C:
			s.pollTasks()
		}
	}
}

func (s *Service) Stop() {
	close(s.stopChan)
}

func (s *Service) pollTasks() {
	url := fmt.Sprintf("%s/%s/tasks?status=pending", s.endpoint, s.projectID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s.logger.Error("failed to create request: %v", err)
		return
	}

	req.Header.Set("X-Auth-Token", s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error("failed to poll tasks: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("polling failed with status: %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("failed to read response: %v", err)
		return
	}

	var tasks []KarboiiTask
	if err := json.Unmarshal(body, &tasks); err != nil {
		s.logger.Error("failed to parse tasks: %v", err)
		return
	}

	s.logger.Debug("polled %d tasks from Karboii", len(tasks))

	for _, kt := range tasks {
		s.processTask(kt)
	}
}

func (s *Service) processTask(kt KarboiiTask) {
	s.logger.Info("processing task %s of type %s", kt.TaskID, kt.Type)

	// Создаем локальную задачу
	localTaskID := s.taskManager.CreateWithKarboiiID(kt.Type, kt.TaskID, kt.Params)

	// Запускаем выполнение в зависимости от типа задачи
	go s.executeTask(localTaskID, kt)
}

func (s *Service) executeTask(localTaskID string, kt KarboiiTask) {
	ctx := context.Background()
	var err error

	switch kt.Type {
	case "backup":
		err = s.performBackup(ctx, localTaskID, kt.Params)
	case "restore":
		err = s.performRestore(ctx, localTaskID, kt.Params)
	case "delete":
		err = s.performDelete(ctx, localTaskID, kt.Params)
	default:
		err = fmt.Errorf("unknown task type: %s", kt.Type)
	}

	if err != nil {
		s.logger.Error("task %s failed: %v", localTaskID, err)
		s.taskManager.Update(localTaskID, task.TaskFailed, err.Error())
	} else {
		s.logger.Info("task %s completed", localTaskID)
		s.taskManager.Update(localTaskID, task.TaskCompleted, "")
	}
}

func (s *Service) performBackup(ctx context.Context, taskID string, params map[string]interface{}) error {
	sourcePath, ok := params["source_path"].(string)
	if !ok {
		return fmt.Errorf("source_path not provided")
	}
	s3Key, ok := params["s3_key"].(string)
	if !ok {
		return fmt.Errorf("s3_key not provided")
	}

	tmpFile, err := os.CreateTemp(s.tempDir, "backup-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := s.archiver.Archive(sourcePath, tmpFile.Name()); err != nil {
		return fmt.Errorf("failed to archive: %w", err)
	}

	bucket := "" // Используем bucket из конфига или params
	if b, ok := params["s3_bucket"].(string); ok {
		bucket = b
	}

	if err := s.s3Client.Upload(ctx, bucket, s3Key, tmpFile.Name()); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	return nil
}

func (s *Service) performRestore(ctx context.Context, taskID string, params map[string]interface{}) error {
	s3Key, ok := params["s3_key"].(string)
	if !ok {
		return fmt.Errorf("s3_key not provided")
	}
	destPath, ok := params["destination_path"].(string)
	if !ok {
		return fmt.Errorf("destination_path not provided")
	}

	tmpFile, err := os.CreateTemp(s.tempDir, "restore-*.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	bucket := "" // Используем bucket из конфига или params
	if b, ok := params["s3_bucket"].(string); ok {
		bucket = b
	}

	if err := s.s3Client.Download(ctx, bucket, s3Key, tmpFile.Name()); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	if err := s.archiver.Extract(tmpFile.Name(), destPath); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	return nil
}

func (s *Service) performDelete(ctx context.Context, taskID string, params map[string]interface{}) error {
	s3Key, ok := params["s3_key"].(string)
	if !ok {
		return fmt.Errorf("s3_key not provided")
	}

	bucket := "" // Используем bucket из конфига или params
	if b, ok := params["s3_bucket"].(string); ok {
		bucket = b
	}

	return s.s3Client.Delete(ctx, bucket, s3Key)
}