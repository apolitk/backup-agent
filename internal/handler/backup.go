package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"agent/internal/archiver"
	"agent/internal/logger"
	"agent/internal/storage"
	"agent/internal/task"
	"agent/pkg/response"
)

type BackupRequest struct {
	SourcePath string `json:"source_path"`
	S3Bucket   string `json:"s3_bucket,omitempty"`
	S3Key      string `json:"s3_key"`
}

type TaskIDResponse struct {
	TaskID string `json:"task_id"`
}

type BackupTaskData struct {
	SourcePath string
	Bucket     string
	Key        string
	TempDir    string
}

func BackupHandler(
	tm *task.TaskManager,
	s3Client storage.S3Client,
	arc archiver.Archiver,
	tempDir string,
	log *logger.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BackupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := validateBackupRequest(req); err != nil {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		data := BackupTaskData{
			SourcePath: req.SourcePath,
			Bucket:     req.S3Bucket,
			Key:        req.S3Key,
			TempDir:    tempDir,
		}

		taskID := tm.Create("backup", data)
		log.Info("backup task created: %s", taskID)

		go func() {
			if err := performBackup(context.Background(), tm, s3Client, arc, taskID, data, log); err != nil {
				log.Error("backup task %s failed: %v", taskID, err)
				tm.Update(taskID, task.TaskFailed, err.Error())
			} else {
				log.Info("backup task %s completed", taskID)
				tm.Update(taskID, task.TaskCompleted, "")
			}
		}()

		response.JSON(w, http.StatusOK, TaskIDResponse{TaskID: taskID})
	}
}

func validateBackupRequest(req BackupRequest) error {
	if req.SourcePath == "" {
		return fmt.Errorf("source_path is required")
	}
	if req.S3Key == "" {
		return fmt.Errorf("s3_key is required")
	}
	if _, err := os.Stat(req.SourcePath); err != nil {
		return fmt.Errorf("source_path does not exist: %w", err)
	}
	return nil
}

func performBackup(
	ctx context.Context,
	tm *task.TaskManager,
	s3Client storage.S3Client,
	arc archiver.Archiver,
	taskID string,
	data BackupTaskData,
	log *logger.Logger,
) error {
	tmpFile, err := os.CreateTemp(data.TempDir, "backup-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	log.Debug("creating archive at %s", tmpFileName)

	// Архивирование с прогрессом
	if err := arc.CreateArchive(data.SourcePath, tmpFileName, func(processed int64) {
		log.Debug("backup progress: %d bytes processed", processed)
	}); err != nil {
		return fmt.Errorf("create archive: %w", err)
	}

	// Получаем размер архива
	fi, err := os.Stat(tmpFileName)
	if err != nil {
		return fmt.Errorf("stat archive: %w", err)
	}

	log.Debug("archive created, size: %d bytes", fi.Size())

	// Загружаем в S3
	f, err := os.Open(tmpFileName)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	if err := s3Client.Upload(ctx, data.Key, f, fi.Size()); err != nil {
		return fmt.Errorf("upload to S3: %w", err)
	}

	log.Info("backup uploaded to S3: %s (size: %d bytes)", data.Key, fi.Size())
	return nil
}