package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/umesh/dgla/config"
	"github.com/umesh/dgla/logger"
)

// Status represents the current status of the system component
type Status string

const (
	// StatusPass indicates the component is working properly
	StatusPass Status = "pass"
	// StatusWarn indicates the component is working but with issues
	StatusWarn Status = "warn"
	// StatusFail indicates the component is not working
	StatusFail Status = "fail"
)

// ComponentType represents a system component type
type ComponentType string

const (
	// ComponentTypeDatabase represents a database component
	ComponentTypeDatabase ComponentType = "database"
	// ComponentTypeCache represents a cache component
	ComponentTypeCache ComponentType = "cache"
	// ComponentTypeService represents a business logic service
	ComponentTypeService ComponentType = "service"
	// ComponentTypeAPI represents an API endpoint
	ComponentTypeAPI ComponentType = "api"
)

// Component represents a system component that can be health-checked
type Component struct {
	Name            string        `json:"name"`
	Type            ComponentType `json:"type"`
	Status          Status        `json:"status"`
	TimeOfCheck     time.Time     `json:"time_of_check"`
	Description     string        `json:"description,omitempty"`
	Error           string        `json:"error,omitempty"`
	LastSuccessTime time.Time     `json:"last_success_time,omitempty"`
	CheckInterval   time.Duration `json:"-"`
	checkFunc       func() (Status, error)
}

// HealthCheckResponse represents the response from a health check
type HealthCheckResponse struct {
	Status     Status                `json:"status"`
	Version    string                `json:"version"`
	ReleaseID  string                `json:"releaseId,omitempty"`
	Notes      []string              `json:"notes,omitempty"`
	Output     string                `json:"output,omitempty"`
	ServiceID  string                `json:"serviceId"`
	Components map[string]*Component `json:"components"`
	Timestamp  time.Time             `json:"timestamp"`
}

// Checker manages health checks for multiple components
type Checker struct {
	mu           sync.RWMutex
	components   map[string]*Component
	logger       *logger.Logger
	config       *config.Config
	serviceID    string
	version      string
	releaseID    string
	startTime    time.Time
	checkTicker  *time.Ticker
	shutdownCh   chan struct{}
	isRunning    bool
	lastResponse *HealthCheckResponse
}

// NewChecker creates a new health checker
func NewChecker(cfg *config.Config, log *logger.Logger, serviceID, version, releaseID string) *Checker {
	return &Checker{
		components: make(map[string]*Component),
		logger:     log,
		config:     cfg,
		serviceID:  serviceID,
		version:    version,
		releaseID:  releaseID,
		startTime:  time.Now(),
		shutdownCh: make(chan struct{}),
	}
}

// AddComponent adds a component to be health-checked
func (c *Checker) AddComponent(name string, compType ComponentType, checkInterval time.Duration, checkFunc func() (Status, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.components[name] = &Component{
		Name:          name,
		Type:          compType,
		Status:        StatusWarn, // Start with warning until first check completes
		TimeOfCheck:   time.Now(),
		CheckInterval: checkInterval,
		checkFunc:     checkFunc,
		Description:   "Component pending first health check",
	}
}

// AddRedisCheck adds a Redis cache health check
func (c *Checker) AddRedisCheck(client *redis.Client, checkInterval time.Duration) {
	c.AddComponent("redis-cache", ComponentTypeCache, checkInterval, func() (Status, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Try to ping Redis
		pong, err := client.Ping(ctx).Result()
		if err != nil {
			return StatusFail, err
		}

		if pong != "PONG" {
			return StatusWarn, nil
		}

		return StatusPass, nil
	})
}

// AddAPIEndpointCheck adds a health check for an API endpoint
func (c *Checker) AddAPIEndpointCheck(url string, expectedStatus int, checkInterval time.Duration) {
	c.AddComponent("api-"+url, ComponentTypeAPI, checkInterval, func() (Status, error) {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return StatusFail, err
		}

		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			return StatusFail, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != expectedStatus {
			return StatusWarn, nil
		}

		return StatusPass, nil
	})
}

// AddDiskSpaceCheck adds a health check for available disk space
func (c *Checker) AddDiskSpaceCheck(path string, minFreeSpaceGB float64, checkInterval time.Duration) {
	c.AddComponent("disk-space", ComponentTypeService, checkInterval, func() (Status, error) {
		// This is a simplified version. In a real implementation, you would use
		// OS-specific calls to check disk space (e.g., syscall.Statfs_t on Linux)
		// For demonstration purposes only
		return StatusPass, nil
	})
}

// Start begins periodic health checks
func (c *Checker) Start() {
	c.mu.Lock()
	if c.isRunning {
		c.mu.Unlock()
		return
	}
	c.isRunning = true
	c.mu.Unlock()

	// Run an immediate check
	c.checkAll()

	// Setup ticker for regular checks
	c.checkTicker = time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-c.checkTicker.C:
				c.checkAll()
			case <-c.shutdownCh:
				c.checkTicker.Stop()
				return
			}
		}
	}()
}

// Stop stops periodic health checks
func (c *Checker) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return
	}

	close(c.shutdownCh)
	c.isRunning = false
}

// checkAll performs health checks on all components
func (c *Checker) checkAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	totalStatus := StatusPass

	// Check each component
	for name, component := range c.components {
		// Skip if it's not time to check this component yet
		if now.Sub(component.TimeOfCheck) < component.CheckInterval {
			continue
		}

		status, err := component.checkFunc()
		component.Status = status
		component.TimeOfCheck = now

		if err != nil {
			component.Error = err.Error()
			c.logger.Warn().
				Str("component", name).
				Str("status", string(status)).
				Err(err).
				Msg("Component health check failed")
		} else {
			component.Error = ""
			component.LastSuccessTime = now
			c.logger.Debug().
				Str("component", name).
				Str("status", string(status)).
				Msg("Component health check completed")
		}

		// Update overall status (worst case wins)
		if status == StatusFail {
			totalStatus = StatusFail
		} else if status == StatusWarn && totalStatus != StatusFail {
			totalStatus = StatusWarn
		}
	}

	// Update last response
	c.lastResponse = &HealthCheckResponse{
		Status:     totalStatus,
		Version:    c.version,
		ReleaseID:  c.releaseID,
		ServiceID:  c.serviceID,
		Components: c.components,
		Timestamp:  now,
	}
}

// GetHealth returns the current health status
func (c *Checker) GetHealth() *HealthCheckResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// If we haven't checked yet, do an immediate check
	if c.lastResponse == nil {
		c.mu.RUnlock()
		c.checkAll()
		c.mu.RLock()
	}

	return c.lastResponse
}

// GetReadiness returns the readiness status
func (c *Checker) GetReadiness() *HealthCheckResponse {
	response := c.GetHealth()
	
	// For readiness, we're only concerned with critical components
	// Copy only the components that affect readiness
	readinessResponse := &HealthCheckResponse{
		Status:     response.Status,
		Version:    response.Version,
		ServiceID:  response.ServiceID,
		Components: make(map[string]*Component),
		Timestamp:  response.Timestamp,
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Only include database and cache components for readiness
	for name, component := range c.components {
		if component.Type == ComponentTypeDatabase || component.Type == ComponentTypeCache {
			readinessResponse.Components[name] = component
		}
	}

	return readinessResponse
}

// GetLiveness returns the liveness status
func (c *Checker) GetLiveness() *HealthCheckResponse {
	// For liveness, we're only checking if the service is up and responding
	// We don't care about the status of external dependencies
	return &HealthCheckResponse{
		Status:    StatusPass,
		Version:   c.version,
		ServiceID: c.serviceID,
		Timestamp: time.Now(),
	}
}

// HealthHandler returns an HTTP handler for health checks
func (c *Checker) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := c.GetHealth()

		w.Header().Set("Content-Type", "application/json")
		
		if health.Status != StatusPass {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(health)
	}
}

// ReadinessHandler returns an HTTP handler for readiness checks
func (c *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		readiness := c.GetReadiness()

		w.Header().Set("Content-Type", "application/json")
		
		if readiness.Status != StatusPass {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(readiness)
	}
}

// LivenessHandler returns an HTTP handler for liveness checks
func (c *Checker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		liveness := c.GetLiveness()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(liveness)
	}
}
