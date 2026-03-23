package handler

import (
	"net/http"
	"strings"

	"agent/internal/logger"
	"agent/internal/task"
	"agent/pkg/response"
)

func StatusHandler(tm *task.TaskManager, log *logger.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/task/")
		if taskID == "" {
			response.Error(w, http.StatusBadRequest, "task_id is required")
			return
		}

		t, err := tm.Get(taskID)
		if err != nil {
			response.Error(w, http.StatusNotFound, "task not found")
			return
		}

		response.JSON(w, http.StatusOK, t)
	}
}