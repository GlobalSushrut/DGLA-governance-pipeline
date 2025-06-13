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
	
	// ZK Packet verification metrics
	packetVerifications    *prometheus.CounterVec
	packetVerificationTime *prometheus.HistogramVec
	invalidPackets         *prometheus.CounterVec
	
	// ChainLog metrics
	chainLogAppends        prometheus.Counter
	chainLogAnchors        *prometheus.CounterVec
	anchoringTime          *prometheus.HistogramVec
	
	// Audit Export metrics
	auditExports           *prometheus.CounterVec
	exportTime             *prometheus.HistogramVec
	exportSize             *prometheus.HistogramVec
	
	// Rate limiting metrics
	apiRateLimit           *prometheus.CounterVec
	apiRateExceeded        *prometheus.CounterVec
	
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
	
	// ZK Packet verification metrics
	packetVerifications := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dgla_packet_verifications_total",
			Help: "Total number of packet verifications.",
		},
		[]string{"algorithm", "status"},
	)
	
	packetVerificationTime := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dgla_packet_verification_seconds",
			Help:    "Time taken to verify packets.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"algorithm"},
	)
	
	invalidPackets := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dgla_invalid_packets_total",
			Help: "Total number of invalid packets detected.",
		},
		[]string{"reason", "algorithm"},
	)
	
	// ChainLog metrics
	chainLogAppends := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dgla_chainlog_appends_total",
			Help: "Total number of ChainLog append operations.",
		},
	)
	
	chainLogAnchors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dgla_chainlog_anchors_total",
			Help: "Total number of ChainLog anchor operations.",
		},
		[]string{"target"},
	)
	
	anchoringTime := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dgla_anchoring_seconds",
			Help:    "Time taken to anchor ChainLog roots.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"target"},
	)
	
	// Audit Export metrics
	auditExports := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dgla_audit_exports_total",
			Help: "Total number of audit log exports.",
		},
		[]string{"format"},
	)
	
	exportTime := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dgla_export_seconds",
			Help:    "Time taken to export audit logs.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"format"},
	)
	
	exportSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "dgla_export_size_bytes",
			Help:    "Size of exported audit logs in bytes.",
			Buckets: []float64{1024, 10*1024, 100*1024, 1024*1024, 10*1024*1024},
		},
		[]string{"format"},
	)
	
	// Rate limiting metrics
	apiRateLimit := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dgla_rate_limit_checks_total",
			Help: "Total number of rate limit checks.",
		},
		[]string{"endpoint"},
	)
	
	apiRateExceeded := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dgla_rate_limit_exceeded_total",
			Help: "Total number of rate limit exceeded events.",
		},
		[]string{"endpoint", "client_ip"},
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
		packetVerifications,
		packetVerificationTime,
		invalidPackets,
		chainLogAppends,
		chainLogAnchors,
		anchoringTime,
		auditExports,
		exportTime,
		exportSize,
		apiRateLimit,
		apiRateExceeded,
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
		
		// ZK Packet verification metrics
		packetVerifications:    packetVerifications,
		packetVerificationTime: packetVerificationTime,
		invalidPackets:         invalidPackets,
		
		// ChainLog metrics
		chainLogAppends:        chainLogAppends,
		chainLogAnchors:        chainLogAnchors,
		anchoringTime:          anchoringTime,
		
		// Audit Export metrics
		auditExports:           auditExports,
		exportTime:             exportTime,
		exportSize:             exportSize,
		
		// Rate limiting metrics
		apiRateLimit:           apiRateLimit,
		apiRateExceeded:        apiRateExceeded,
		
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

// RecordPacketVerification records a packet verification
func (m *Metrics) RecordPacketVerification(algorithm, status string) {
	m.packetVerifications.WithLabelValues(algorithm, status).Inc()
}

// RecordPacketVerificationTime records the time taken to verify a packet
func (m *Metrics) RecordPacketVerificationTime(algorithm string, duration time.Duration) {
	m.packetVerificationTime.WithLabelValues(algorithm).Observe(duration.Seconds())
}

// RecordInvalidPacket records an invalid packet detection
func (m *Metrics) RecordInvalidPacket(reason, algorithm string) {
	m.invalidPackets.WithLabelValues(reason, algorithm).Inc()
}

// RecordChainLogAppend records a ChainLog append
func (m *Metrics) RecordChainLogAppend() {
	m.chainLogAppends.Inc()
}

// RecordChainLogAnchor records a ChainLog anchor
func (m *Metrics) RecordChainLogAnchor(target string) {
	m.chainLogAnchors.WithLabelValues(target).Inc()
}

// RecordAnchoringTime records the time taken to anchor a ChainLog root
func (m *Metrics) RecordAnchoringTime(target string, duration time.Duration) {
	m.anchoringTime.WithLabelValues(target).Observe(duration.Seconds())
}

// RecordAuditExport records an audit log export
func (m *Metrics) RecordAuditExport(format string) {
	m.auditExports.WithLabelValues(format).Inc()
}

// RecordExportTime records the time taken to export audit logs
func (m *Metrics) RecordExportTime(format string, duration time.Duration) {
	m.exportTime.WithLabelValues(format).Observe(duration.Seconds())
}

// RecordExportSize records the size of exported audit logs
func (m *Metrics) RecordExportSize(format string, sizeBytes float64) {
	m.exportSize.WithLabelValues(format).Observe(sizeBytes)
}

// RecordRateLimitCheck records a rate limit check
func (m *Metrics) RecordRateLimitCheck(endpoint string) {
	m.apiRateLimit.WithLabelValues(endpoint).Inc()
}

// RecordRateLimitExceeded records a rate limit exceeded event
func (m *Metrics) RecordRateLimitExceeded(endpoint, clientIP string) {
	m.apiRateExceeded.WithLabelValues(endpoint, clientIP).Inc()
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
