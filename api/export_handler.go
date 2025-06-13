package api

import (
	"encoding/json"
	"net/http"
	"time"
	"strconv"
	"path/filepath"
	"os"
	"fmt"
	"io"

	"github.com/umesh/dgla/chainlog"
	"github.com/umesh/dgla/metrics"
)

// ExportHandler handles audit export API requests
type ExportHandler struct {
	chainlog *chainlog.ChainLogEngine
	metrics  *metrics.Metrics
	exportDir string
}

// NewExportHandler creates a new export handler
func NewExportHandler(chainlog *chainlog.ChainLogEngine, metrics *metrics.Metrics, exportDir string) *ExportHandler {
	// Create export directory if it doesn't exist
	if exportDir != "" {
		os.MkdirAll(exportDir, 0755)
	} else {
		exportDir = "./exports"
		os.MkdirAll(exportDir, 0755)
	}

	return &ExportHandler{
		chainlog: chainlog,
		metrics:  metrics,
		exportDir: exportDir,
	}
}

// ExportRequest represents a request to export audit logs
type ExportRequest struct {
	Format    string            `json:"format"`    // json or pdf
	Filters   map[string]string `json:"filters"`
	StartTime string            `json:"start_time"` // RFC3339 format
	EndTime   string            `json:"end_time"`   // RFC3339 format
}

// HandleExport handles requests to the /export endpoint
func (h *ExportHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	// Only allow GET and POST requests
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request parameters
	var format, startTimeStr, endTimeStr string
	var filters map[string]string

	if r.Method == http.MethodGet {
		// Parse query parameters
		query := r.URL.Query()
		format = query.Get("format")
		
		// Parse filters
		filters = make(map[string]string)
		if entityID := query.Get("entity_id"); entityID != "" {
			filters["entity_id"] = entityID
		}
		if entityType := query.Get("entity_type"); entityType != "" {
			filters["entity_type"] = entityType
		}
		if action := query.Get("action"); action != "" {
			filters["action"] = action
		}
		if userID := query.Get("user_id"); userID != "" {
			filters["user_id"] = userID
		}
		
		startTimeStr = query.Get("start_time")
		endTimeStr = query.Get("end_time")
	} else {
		// Parse JSON body
		var req ExportRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		
		format = req.Format
		filters = req.Filters
		startTimeStr = req.StartTime
		endTimeStr = req.EndTime
	}

	// Default format to JSON if not specified
	if format == "" {
		format = "json"
	}

	// Validate format
	if format != "json" && format != "pdf" {
		http.Error(w, "Invalid format. Must be 'json' or 'pdf'", http.StatusBadRequest)
		return
	}

	// Parse time range
	var startTime, endTime time.Time
	var err error
	
	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			http.Error(w, "Invalid start_time format. Must be RFC3339", http.StatusBadRequest)
			return
		}
	}
	
	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			http.Error(w, "Invalid end_time format. Must be RFC3339", http.StatusBadRequest)
			return
		}
	} else {
		endTime = time.Now()
	}
	
	// Start timer for metrics
	start := time.Now()
	
	// Generate export filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("audit_export_%s.%s", timestamp, format)
	filepath := filepath.Join(h.exportDir, filename)
	
	// Get logs
	logs, err := h.chainlog.GetLogs(filters, 0) // 0 means no limit
	if err != nil {
		http.Error(w, "Error retrieving logs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Filter logs by time range
	filteredLogs := make([]chainlog.LogEntry, 0)
	for _, log := range logs {
		if (startTime.IsZero() || !log.Timestamp.Before(startTime)) && 
		   (endTime.IsZero() || !log.Timestamp.After(endTime)) {
			filteredLogs = append(filteredLogs, log)
		}
	}
	
	// Export logs
	var fileSize int64
	if format == "json" {
		fileSize, err = h.exportToJSON(filteredLogs, filepath)
	} else {
		fileSize, err = h.exportToPDF(filteredLogs, filepath)
	}
	
	if err != nil {
		http.Error(w, "Export error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Record metrics
	duration := time.Since(start)
	h.metrics.RecordAuditExport(format)
	h.metrics.RecordExportTime(format, duration)
	h.metrics.RecordExportSize(format, float64(fileSize))
	
	// Serve the file
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	
	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
	} else {
		w.Header().Set("Content-Type", "application/pdf")
	}
	
	w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
	
	file, err := os.Open(filepath)
	if err != nil {
		http.Error(w, "Error reading export file", http.StatusInternalServerError)
		return
	}
	defer file.Close()
	
	io.Copy(w, file)
}

// exportToJSON exports logs to a JSON file
func (h *ExportHandler) exportToJSON(logs []chainlog.LogEntry, filepath string) (int64, error) {
	// Convert logs to API format
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
	
	// Create export file
	file, err := os.Create(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	
	// Write JSON to file
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(apiLogs); err != nil {
		return 0, err
	}
	
	// Get file size
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return 0, err
	}
	
	return fileInfo.Size(), nil
}

// exportToPDF exports logs to a PDF file
func (h *ExportHandler) exportToPDF(logs []chainlog.LogEntry, filepath string) (int64, error) {
	// In a real implementation, this would use the gofpdf library to create a PDF
	// For now, we'll create a simple text file with .pdf extension
	file, err := os.Create(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	
	// Write header
	file.WriteString("DGLA Audit Log Export\n")
	file.WriteString("======================\n\n")
	file.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC1123)))
	
	// Write each log entry
	for _, log := range logs {
		file.WriteString(fmt.Sprintf("ID: %s\n", log.ID))
		file.WriteString(fmt.Sprintf("Timestamp: %s\n", log.Timestamp.Format(time.RFC3339)))
		file.WriteString(fmt.Sprintf("Action: %s\n", log.Action))
		file.WriteString(fmt.Sprintf("Entity: %s (%s)\n", log.EntityID, log.EntityType))
		if log.UserID != "" {
			file.WriteString(fmt.Sprintf("User: %s\n", log.UserID))
		}
		
		if len(log.Metadata) > 0 {
			file.WriteString("Metadata:\n")
			for k, v := range log.Metadata {
				file.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
			}
		}
		
		file.WriteString("\n---\n\n")
	}
	
	// Add verification information
	file.WriteString("\nVerification Information\n")
	file.WriteString("=======================\n\n")
	file.WriteString(fmt.Sprintf("Total Records: %d\n", len(logs)))
	file.WriteString(fmt.Sprintf("Export Time: %s\n", time.Now().Format(time.RFC3339)))
	file.WriteString("Verification URL: https://verify.dgla.io\n")
	
	// Get file size
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return 0, err
	}
	
	return fileInfo.Size(), nil
}
