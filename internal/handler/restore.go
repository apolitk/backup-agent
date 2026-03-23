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

type RestoreRequest struct {
	S3Bucket        string `json:"s3_bucket,omitempty"`
	S3Key           string `json:"s3_key"`
	DestinationPath string `json:"destination_path"`
}

type RestoreTaskData struct {
	Bucket          string
	Key             string
	DestinationPath string
	TempDir         string
}

func RestoreHandler(
	tm *task.TaskManager,
	s3Client storage.S3Client,
	arc archiver.Archiver,
	tempDir string,
	log *logger.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RestoreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if err := validateRestoreRequest(req); err != nil {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}

		data := RestoreTaskData{
			Bucket:          req.S3Bucket,
			Key:             req.S3Key,
			DestinationPath: req.DestinationPath,
			TempDir:         tempDir,
		}

		taskID := tm.Create("restore", data)
		log.Info("restore task created: %s", taskID)

		go func() {
			if err := performRestore(context.Background(), tm, s3Client, arc, taskID, data, log); err != nil {
				log.Error("restore task %s failed: %v", taskID, err)
				tm.Update(taskID, task.TaskFailed, err.Error())
			} else {
				log.Info("restore task %s completed", taskID)
				tm.Update(taskID, task.TaskCompleted, "")
			}
		}()

		response.JSON(w, http.StatusOK, TaskIDResponse{TaskID: taskID})
	}
}

func validateRestoreRequest(req RestoreRequest) error {
	if req.S3Key == "" {
		return fmt.Errorf("s3_key is required")
	}
	if req.DestinationPath == "" {
		return fmt.Errorf("destination_path is required")
	}
	return nil
}

func performRestore(
	ctx context.Context,
	tm *task.TaskManager,
	s3Client storage.S3Client,
	arc archiver.Archiver,
	taskID string,
	data RestoreTaskData,
	log *logger.Logger,
) error {
	tmpFile, err := os.CreateTemp(data.TempDir, "restore-*.tar.gz")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpFileName := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFileName)

	log.Debug("downloading archive from S3: %s", data.Key)

	// Скачиваем архив
	reader, err := s3Client.Download(ctx, data.Key)
	if err != nil {
		return fmt.Errorf("download from S3: %w", err)
	}
	defer reader.Close()

	// Сохраняем во временный файл
	f, err := os.Create(tmpFileName)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := f.ReadFrom(reader); err != nil {
		return fmt.Errorf("write archive: %w", err)
	}

	log.Debug("archive downloaded, extracting to %s", data.DestinationPath)

	// Создаем целевую директорию
	if err := os.MkdirAll(data.DestinationPath, 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	// Распаковываем архив
	if err := arc.ExtractArchive(tmpFileName, data.DestinationPath, func(processed int64) {
		log.Debug("restore progress: %d bytes processed", processed)
	}); err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}

	log.Info("restore completed to %s", data.DestinationPath)
	return nil
}