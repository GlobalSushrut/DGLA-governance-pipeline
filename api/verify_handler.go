package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/umesh/dgla/metrics"
	"github.com/umesh/dgla/verifier"
)

// VerifyHandler handles ZK packet verification requests
type VerifyHandler struct {
	verifier *verifier.ZKPacketVerifier
	metrics  *metrics.Metrics
}

// NewVerifyHandler creates a new verification handler
func NewVerifyHandler(verifier *verifier.ZKPacketVerifier, metrics *metrics.Metrics) *VerifyHandler {
	return &VerifyHandler{
		verifier: verifier,
		metrics:  metrics,
	}
}

// VerifyRequest represents the verification request
type VerifyRequest struct {
	Packet    string `json:"packet"`
	Proof     string `json:"proof"`
	Algorithm string `json:"algorithm"`
}

// VerifyResponse represents the verification response
type VerifyResponse struct {
	Valid       bool   `json:"valid"`
	Message     string `json:"message,omitempty"`
	ProcessedAt string `json:"processed_at"`
	RequestID   string `json:"request_id"`
}

// ServeHTTP handles HTTP requests for verification
func (h *VerifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse request JSON
	var req VerifyRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Packet == "" || req.Proof == "" {
		http.Error(w, "Packet and proof are required", http.StatusBadRequest)
		return
	}

	// Default to SHA-256 if algorithm not specified
	algorithm := req.Algorithm
	if algorithm == "" {
		algorithm = "sha256"
	}

	// Measure verification time
	start := time.Now()

	// Verify the packet
	valid, err := h.verifier.VerifyPacket(req.Packet, req.Proof, algorithm)
	
	// Record metrics
	duration := time.Since(start)
	h.metrics.RecordPacketVerificationTime(algorithm, duration)
	
	status := "success"
	if !valid || err != nil {
		status = "failure"
	}
	h.metrics.RecordPacketVerification(algorithm, status)

	// Handle verification errors
	if err != nil {
		h.metrics.RecordInvalidPacket("error", algorithm)
		http.Error(w, "Verification error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create response
	resp := VerifyResponse{
		Valid:       valid,
		ProcessedAt: time.Now().UTC().Format(time.RFC3339),
		RequestID:   generateRequestID(),
	}

	if !valid {
		resp.Message = "Invalid proof for packet"
		h.metrics.RecordInvalidPacket("invalid_proof", algorithm)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	if !valid {
		w.WriteHeader(http.StatusUnprocessableEntity) // 422 for invalid proof
	}
	json.NewEncoder(w).Encode(resp)
}

// generateRequestID creates a unique ID for the request
func generateRequestID() string {
	// In a real implementation, this would use a proper UUID library
	return "req_" + time.Now().UTC().Format("20060102150405") + "_" + generateRandomString(8)
}

// generateRandomString creates a random string of given length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}
