package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics collects and exposes system metrics
type Metrics struct {
	// Request metrics
	requestCount      *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	requestsInFlight  prometheus.Gauge
	
	// Cache metrics
	cacheHits         prometheus.Counter
	cacheMisses       prometheus.Counter
	
	// Rule engine metrics
	ruleEvaluations   *prometheus.CounterVec
	ruleViolations    *prometheus.CounterVec
	
	// Merkle tree metrics
	merkleTreeTime    prometheus.Histogram
	
	registry          *prometheus.Registry
	mutex             sync.Mutex
}

// NewMetrics creates a new metrics collection
func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	
	requestCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dgla_requests_total",
			Help: "Total number of requests processed.",
		},
		[]string{"method", "endpoint", "status"},
	)
	
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dgla_request_duration_seconds",
			Help:    "Request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
	
	requestsInFlight := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "dgla_requests_in_flight",
			Help: "Current number of requests being processed.",
		},
	)
	
	cacheHits := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dgla_cache_hits_total",
			Help: "Total number of cache hits.",
		},
	)
	
	cacheMisses := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dgla_cache_misses_total",
			Help: "Total number of cache misses.",
		},
	)
	
	ruleEvaluations := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dgla_rule_evaluations_total",
			Help: "Total number of rule evaluations.",
		},
		[]string{"rule_id"},
	)
	
	ruleViolations := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dgla_rule_violations_total",
			Help: "Total number of rule violations.",
		},
		[]string{"rule_id"},
	)
	
	merkleTreeTime := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "dgla_merkle_tree_creation_seconds",
			Help:    "Time taken to create Merkle trees.",
			Buckets: prometheus.DefBuckets,
		},
	)
	
	// Register all metrics
	registry.MustRegister(
		requestCount,
		requestDuration,
		requestsInFlight,
		cacheHits,
		cacheMisses,
		ruleEvaluations,
		ruleViolations,
		merkleTreeTime,
	)
	
	return &Metrics{
		requestCount:     requestCount,
		requestDuration:  requestDuration,
		requestsInFlight: requestsInFlight,
		cacheHits:        cacheHits,
		cacheMisses:      cacheMisses,
		ruleEvaluations:  ruleEvaluations,
		ruleViolations:   ruleViolations,
		merkleTreeTime:   merkleTreeTime,
		registry:         registry,
	}
}

// Handler returns an HTTP handler for exposing metrics
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// RecordRequest records request metrics
func (m *Metrics) RecordRequest(method, endpoint string, status int, duration time.Duration) {
	m.requestCount.WithLabelValues(method, endpoint, string(status)).Inc()
	m.requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RequestStarted increments the in-flight requests counter
func (m *Metrics) RequestStarted() {
	m.requestsInFlight.Inc()
}

// RequestCompleted decrements the in-flight requests counter
func (m *Metrics) RequestCompleted() {
	m.requestsInFlight.Dec()
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit() {
	m.cacheHits.Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss() {
	m.cacheMisses.Inc()
}

// RecordRuleEvaluation records a rule evaluation
func (m *Metrics) RecordRuleEvaluation(ruleID string) {
	m.ruleEvaluations.WithLabelValues(ruleID).Inc()
}

// RecordRuleViolation records a rule violation
func (m *Metrics) RecordRuleViolation(ruleID string) {
	m.ruleViolations.WithLabelValues(ruleID).Inc()
}

// RecordMerkleTreeTime records the time taken to create a Merkle tree
func (m *Metrics) RecordMerkleTreeTime(duration time.Duration) {
	m.merkleTreeTime.Observe(duration.Seconds())
}

// MetricsMiddleware adds metrics collection to HTTP handlers
func (m *Metrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.RequestStarted()
		
		// Create a response writer wrapper to capture the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Call the next handler
		next.ServeHTTP(rw, r)
		
		// Record metrics
		duration := time.Since(start)
		m.RecordRequest(r.Method, r.URL.Path, rw.statusCode, duration)
		m.RequestCompleted()
	})
}

// responseWriter is a wrapper around http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
	rw.written = true
}

// Write captures that a response has been written
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}
