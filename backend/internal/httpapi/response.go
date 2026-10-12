package httpapi

import (
	"encoding/json"
	"net/http"
)

// Standard response envelope matching docs/API.md
type ApiResponse struct {
	Success bool        `json:"success"`
	Data    any         `json:"data,omitempty"`
	Error   *ApiError   `json:"error,omitempty"`
	Meta    *Pagination `json:"meta,omitempty"`
}

type ApiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

// JSON sends a successful JSON response with HTTP status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ApiResponse{
		Success: true,
		Data:    data,
	})
}

// JSONWithMeta sends a successful JSON response with pagination metadata.
func JSONWithMeta(w http.ResponseWriter, status int, data any, meta *Pagination) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ApiResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// ErrorJSON sends a structured JSON error response.
func ErrorJSON(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ApiResponse{
		Success: false,
		Error: &ApiError{
			Code:    code,
			Message: message,
			Fields:  fields,
		},
	})
}
