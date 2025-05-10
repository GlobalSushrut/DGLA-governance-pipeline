package handlers

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/umesh/dgla/logger"
)

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// ErrorHandler middleware provides centralized error handling
func ErrorHandler(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a panic recovery function
			defer func() {
				if err := recover(); err != nil {
					// Log the stack trace
					stackTrace := debug.Stack()
					log.WithFields(map[string]interface{}{
						"panic":       err,
						"stack_trace": string(stackTrace),
						"method":      r.Method,
						"path":        r.URL.Path,
					}).Error("Panic recovered in HTTP handler")

					// Generate a trace ID for this error
					traceID := generateTraceID()

					// Return a standard error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(ErrorResponse{
						Status:  http.StatusInternalServerError,
						Message: "An unexpected error occurred",
						Error:   "internal_server_error",
						TraceID: traceID,
					})
				}
			}()

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// generateTraceID creates a unique identifier for error tracing
func generateTraceID() string {
	// In a real implementation, this would generate a truly unique ID
	// using something like UUID
	return "trace-" + "12345"
}

// SendJSONError sends a standardized JSON error response
func SendJSONError(w http.ResponseWriter, status int, message, errorCode string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Status:  status,
		Message: message,
		Error:   errorCode,
	})
}
