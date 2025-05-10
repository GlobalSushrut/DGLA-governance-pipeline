package handlers

import (
	"net/http"
	"time"

	"github.com/umesh/dgla/logger"
)

// RequestLogger provides middleware for logging HTTP requests
func RequestLogger(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture the status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Log request details
			requestLogger := log.WithFields(map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"remote_ip":  r.RemoteAddr,
				"user_agent": r.UserAgent(),
			})

			requestLogger.Info("Request started")

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Log response details
			duration := time.Since(start)
			requestLogger.WithFields(map[string]interface{}{
				"status":       rw.statusCode,
				"duration_ms":  duration.Milliseconds(),
				"content_size": rw.size,
			}).Info("Request completed")
		})
	}
}

// responseWriter is a wrapper around http.ResponseWriter to capture response details
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
	written    bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
	rw.written = true
}

// Write captures the response size
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}
