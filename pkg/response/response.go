package response

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Data interface{} `json:"data"`
}

func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, statusCode int, message string) {
	JSON(w, statusCode, ErrorResponse{Error: message})
}

func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, SuccessResponse{Data: data})
}