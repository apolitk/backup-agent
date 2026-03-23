package handler

import (
	"context"
	"net/http"

	"agent/internal/logger"
	"agent/internal/storage"
	"agent/pkg/response"
)

func ListHandler(s3Client storage.S3Client, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("prefix")
		log.Debug("listing backups with prefix: %s", prefix)

		objects, err := s3Client.List(context.Background(), prefix)
		if err != nil {
			response.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		response.JSON(w, http.StatusOK, objects)
	}
}