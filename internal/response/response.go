package response

import (
	"encoding/json"
	"net/http"
)

type ErrorDetail struct {
	Field   string `json:"field" example:"email"`
	Message string `json:"message" example:"email is required"`
}

// APIResponse is the standard envelope used by every endpoint. Successful
// responses carry the payload under `data`; failed responses carry either an
// `error` string or a list of `errors` describing per-field validation failures.
type APIResponse struct {
	Success bool          `json:"success" example:"true"`
	Data    any           `json:"data,omitempty"`
	Error   string        `json:"error,omitempty" example:"something went wrong"`
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
