package router

import (
	"log"
	"time"

	"github.com/umesh/dgla/cache"
)

// DataRequest represents a data flow request with source, destination and metadata
type DataRequest struct {
	JobID      string    `json:"job_id"`
	DataAsset  string    `json:"data_asset"`
	Region     string    `json:"region"`
	Action     string    `json:"action"`
	IsPII      bool      `json:"is_pii"`
	Timestamp  time.Time `json:"timestamp"`
	Source     string    `json:"source"`
	Destination string   `json:"destination"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// IdentityRouter tracks data flows and records identity information
type IdentityRouter struct {
	cache *cache.RedisLikeCache
	logs  []DataRequest
}

// NewIdentityRouter creates a new identity router
func NewIdentityRouter(cache *cache.RedisLikeCache) *IdentityRouter {
	return &IdentityRouter{
		cache: cache,
		logs:  make([]DataRequest, 0),
	}
}

// Route processes a data flow request and tracks identity information
func (r *IdentityRouter) Route(request DataRequest) error {
	// Set current timestamp if not provided
	if request.Timestamp.IsZero() {
		request.Timestamp = time.Now()
	}

	// Log the event
	log.Printf("Routing: %s -> %s (JobID: %s, Region: %s, PII: %v)", 
		request.DataAsset, request.Destination, request.JobID, request.Region, request.IsPII)
	
	// Store in cache for quick lookups
	r.cache.Set(
		request.JobID+"_latest", 
		request, 
		24*time.Hour, // Cache for 24 hours
	)
	
	// Add to logs for auditing
	r.logs = append(r.logs, request)
	
	return nil
}

// GetLogs returns all logged data requests
func (r *IdentityRouter) GetLogs() []DataRequest {
	return r.logs
}

// GetLatestRequest retrieves the latest request for a specific job
func (r *IdentityRouter) GetLatestRequest(jobID string) (DataRequest, bool) {
	data, found := r.cache.Get(jobID + "_latest")
	if !found {
		return DataRequest{}, false
	}
	
	request, ok := data.(DataRequest)
	return request, ok
}
