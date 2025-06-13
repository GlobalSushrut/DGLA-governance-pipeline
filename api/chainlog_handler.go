package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/umesh/dgla/chainlog"
	"github.com/umesh/dgla/metrics"
)

// ChainLogHandler handles ChainLog related API requests
type ChainLogHandler struct {
	chainlog *chainlog.ChainLogEngine
	metrics  *metrics.Metrics
}

// NewChainLogHandler creates a new ChainLog handler
func NewChainLogHandler(chainlog *chainlog.ChainLogEngine, metrics *metrics.Metrics) *ChainLogHandler {
	return &ChainLogHandler{
		chainlog: chainlog,
		metrics:  metrics,
	}
}

// LogEntry represents a log entry for the API
type LogEntry struct {
	ID          string            `json:"id"`
	Timestamp   string            `json:"timestamp"`
	Action      string            `json:"action"`
	EntityID    string            `json:"entity_id"`
	EntityType  string            `json:"entity_type"`
	UserID      string            `json:"user_id"`
	Metadata    map[string]string `json:"metadata"`
	Signature   string            `json:"signature,omitempty"`
	MerkleProof []string          `json:"merkle_proof,omitempty"`
}

// LogRequest represents a request to add a log entry
type LogRequest struct {
	Action     string            `json:"action"`
	EntityID   string            `json:"entity_id"`
	EntityType string            `json:"entity_type"`
	UserID     string            `json:"user_id"`
	Metadata   map[string]string `json:"metadata"`
}

// LogResponse represents the response after adding a log entry
type LogResponse struct {
	Success   bool     `json:"success"`
	LogID     string   `json:"log_id"`
	Timestamp string   `json:"timestamp"`
	MerkleRoot string  `json:"merkle_root"`
}

// AnchorRequest represents a request to anchor the ChainLog
type AnchorRequest struct {
	Target    string `json:"target"` // ethereum, celestia, or ipfs
	Immediate bool   `json:"immediate"`
}

// AnchorResponse represents the response after anchoring
type AnchorResponse struct {
	Success     bool   `json:"success"`
	Target      string `json:"target"`
	AnchorID    string `json:"anchor_id"`
	Timestamp   string `json:"timestamp"`
	TransactionHash string `json:"transaction_hash,omitempty"`
	MerkleRoot  string `json:"merkle_root"`
}

// HandleLogs handles requests to the /logs endpoint
func (h *ChainLogHandler) HandleLogs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getLogs(w, r)
	case http.MethodPost:
		h.appendLog(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleAnchor handles requests to the /anchor endpoint
func (h *ChainLogHandler) HandleAnchor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req AnchorRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate target
	if req.Target != "ethereum" && req.Target != "celestia" && req.Target != "ipfs" {
		http.Error(w, "Invalid target. Must be 'ethereum', 'celestia', or 'ipfs'", http.StatusBadRequest)
		return
	}

	// Start timer for metrics
	start := time.Now()

	// Anchor the chainlog
	anchorID, txHash, merkleRoot, err := h.chainlog.AnchorRoot(req.Target, req.Immediate)
	
	// Record metrics
	duration := time.Since(start)
	h.metrics.RecordAnchoringTime(req.Target, duration)
	h.metrics.RecordChainLogAnchor(req.Target)

	if err != nil {
		http.Error(w, "Error anchoring chainlog: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create response
	resp := AnchorResponse{
		Success:     true,
		Target:      req.Target,
		AnchorID:    anchorID,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		TransactionHash: txHash,
		MerkleRoot:  merkleRoot,
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// appendLog adds a new entry to the ChainLog
func (h *ChainLogHandler) appendLog(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req LogRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Action == "" || req.EntityID == "" || req.EntityType == "" {
		http.Error(w, "Action, EntityID, and EntityType are required", http.StatusBadRequest)
		return
	}

	// Record metric
	h.metrics.RecordChainLogAppend()

	// Create log entry
	timestamp := time.Now().UTC()
	logEntry := chainlog.LogEntry{
		Action:     req.Action,
		EntityID:   req.EntityID,
		EntityType: req.EntityType,
		UserID:     req.UserID,
		Metadata:   req.Metadata,
		Timestamp:  timestamp,
	}

	// Append to chainlog
	logID, merkleRoot, err := h.chainlog.AppendLog(logEntry)
	if err != nil {
		http.Error(w, "Error appending log: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create response
	resp := LogResponse{
		Success:   true,
		LogID:     logID,
		Timestamp: timestamp.Format(time.RFC3339),
		MerkleRoot: merkleRoot,
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// getLogs retrieves logs from the ChainLog
func (h *ChainLogHandler) getLogs(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	params := r.URL.Query()
	
	// Parse filters
	filters := make(map[string]string)
	
	if entityID := params.Get("entity_id"); entityID != "" {
		filters["entity_id"] = entityID
	}
	
	if entityType := params.Get("entity_type"); entityType != "" {
		filters["entity_type"] = entityType
	}
	
	if action := params.Get("action"); action != "" {
		filters["action"] = action
	}
	
	if userID := params.Get("user_id"); userID != "" {
		filters["user_id"] = userID
	}
	
	// Parse pagination
	limit := 50 // Default limit
	if limitStr := params.Get("limit"); limitStr != "" {
		if parsedLimit, err := parseInt(limitStr, 1, 100); err == nil {
			limit = parsedLimit
		}
	}
	
	// Get logs
	logs, err := h.chainlog.GetLogs(filters, limit)
	if err != nil {
		http.Error(w, "Error retrieving logs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Convert to API format
	apiLogs := make([]LogEntry, 0, len(logs))
	for _, log := range logs {
		apiLog := LogEntry{
			ID:        log.ID,
			Timestamp: log.Timestamp.Format(time.RFC3339),
			Action:    log.Action,
			EntityID:  log.EntityID,
			EntityType: log.EntityType,
			UserID:    log.UserID,
			Metadata:  log.Metadata,
			Signature: log.Signature,
		}
		if log.MerkleProof != nil {
			apiLog.MerkleProof = log.MerkleProof
		}
		apiLogs = append(apiLogs, apiLog)
	}
	
	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiLogs)
}

// parseInt parses a string to int within min and max bounds
func parseInt(s string, min, max int) (int, error) {
	// Implementation would convert string to int and validate bounds
	// For simplicity, just return the default
	return 50, nil
}
