package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agent/internal/archiver"
	"agent/internal/config"
	"agent/internal/handler"
	"agent/internal/logger"
	"agent/internal/polling"
	"agent/internal/reporter"
	"agent/internal/storage"
	"agent/internal/task"
	"agent/internal/worker"
)

func main() {
	cfg := config.Load()

	if cfg.Token == "" {
		fmt.Fprintf(os.Stderr, "AGENT_TOKEN is not set\n")
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	log.Info("starting agent")
	log.Debug("config: port=%s, s3_endpoint=%s, s3_bucket=%s", cfg.Port, cfg.S3Endpoint, cfg.S3Bucket)

	s3Client, err := storage.NewS3Client(storage.Config{
		Endpoint:  cfg.S3Endpoint,
		Region:    cfg.S3Region,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		UseSSL:    cfg.S3UseSSL,
		Bucket:    cfg.S3Bucket,
		Timeout:   cfg.S3Timeout,
	})
	if err != nil {
		log.Error("failed to create S3 client: %v", err)
		os.Exit(1)
	}

	taskManager := task.NewTaskManager(1 * time.Hour)
	defer taskManager.Close()

	// Инициализация reporter для отправки статуса в Karboii
	var statusReporter *reporter.Service
	if cfg.KarboiiEndpoint != "" && cfg.KarboiiToken != "" && cfg.ProjectID != "" {
		statusReporter = reporter.New(cfg.KarboiiEndpoint, cfg.KarboiiToken, cfg.ProjectID, taskManager, log)
		taskManager = task.NewTaskManagerWithCallback(1*time.Hour, func(t *task.Task) {
			if statusReporter != nil {
				result := make(map[string]interface{})
				if t.Data != nil {
					// Можно добавить логику для извлечения результата из t.Data
				}
				statusReporter.ReportStatus(t.KarboiiTaskID, t.Status, t.Message, result)
			}
		})
		defer statusReporter.Stop()
		go statusReporter.Start()
	}

	// Инициализация polling service для получения задач от Karboii
	var pollingService *polling.Service
	if cfg.KarboiiEndpoint != "" && cfg.KarboiiToken != "" && cfg.ProjectID != "" && cfg.PollingInterval > 0 {
		pollingService = polling.New(cfg.KarboiiEndpoint, cfg.KarboiiToken, cfg.ProjectID, cfg.PollingInterval, taskManager, s3Client, arc, cfg.TempDir, log)
		defer pollingService.Stop()
		go pollingService.Start()
	}

	workerPool := worker.New(cfg.MaxWorkers)
	defer workerPool.Stop()

	arc := archiver.New()

	mux := http.NewServeMux()

	// Применяем middleware
	authMiddleware := handler.AuthMiddleware(cfg.Token)

	// Регистрируем роуты
	mux.Handle("/api/v1/backup", authMiddleware(
		handler.BackupHandler(taskManager, s3Client, arc, cfg.TempDir, log),
	))
	mux.Handle("/api/v1/restore", authMiddleware(
		handler.RestoreHandler(taskManager, s3Client, arc, cfg.TempDir, log),
	))
	mux.Handle("/api/v1/task/", authMiddleware(
		handler.StatusHandler(taskManager, log),
	))
	mux.Handle("/api/v1/backups", authMiddleware(
		handler.ListHandler(s3Client, log),
	))
	mux.Handle("/api/v1/backups/", authMiddleware(
		handler.DeleteHandler(s3Client, log),
	))

	// Health check без аутентификации
	mux.HandleFunc("/health", handler.HealthHandler())

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// Канал для ошибок сервера
	serverErrors := make(chan error, 1)

	// Запуск сервера в отдельной goroutine
	go func() {
		log.Info("starting server on %s", server.Addr)
		if cfg.TLSCert != "" && cfg.TLSKey != "" {
			serverErrors <- server.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			serverErrors <- server.ListenAndServe()
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			log.Error("server error: %v", err)
			os.Exit(1)
		}
	case sig := <-sigChan:
		log.Info("received signal: %v", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error("shutdown error: %v", err)
			os.Exit(1)
		}

		log.Info("server stopped gracefully")
	}
}