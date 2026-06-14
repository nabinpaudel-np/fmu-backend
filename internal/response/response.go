package response

import (
	"encoding/json"
	"net/http"
)

type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type APIResponse struct {
	Success bool          `json:"success"`
	Data    any           `json:"data,omitempty"`
	Error   string        `json:"error,omitempty"`
	Errors  []ErrorDetail `json:"errors,omitempty"`
}

func Success(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(&APIResponse{
		Success: true,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(&APIResponse{
		Success: false,
		Error:   msg,
	})
}

func ValidationError(w http.ResponseWriter, status int, errors []ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(&APIResponse{
		Success: false,
		Errors:  errors,
	})
}
