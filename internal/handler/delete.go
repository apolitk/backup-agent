package handler

import (
	"context"
	"net/http"
	"strings"

	"agent/internal/logger"
	"agent/internal/storage"
	"agent/pkg/response"
)

func DeleteHandler(s3Client storage.S3Client, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/api/v1/backups/")
		if key == "" {
			response.Error(w, http.StatusBadRequest, "key is required")
			return
		}

		log.Debug("deleting backup: %s", key)

		if err := s3Client.Delete(context.Background(), key); err != nil {
			response.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		log.Info("backup deleted: %s", key)
		w.WriteHeader(http.StatusNoContent)
	}
}