package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/umesh/dgla/auth"
	"github.com/umesh/dgla/cache"
	"github.com/umesh/dgla/config"
	"github.com/umesh/dgla/health"
	"github.com/umesh/dgla/logger"
	"github.com/umesh/dgla/metrics"
	"github.com/umesh/dgla/middleware"
	"github.com/umesh/dgla/middleware/handlers"
	"github.com/umesh/dgla/router"
)

const (
	serviceVersion = "1.0.0"
	serviceID     = "dgla-governance-pipeline"
	releaseID     = "2025.05.1"
)

// Sample rules for demonstration
var sampleRules = []middleware.Rule{
	{
		RuleID: "EU_PII_REGION_LOCK",
		Condition: middleware.Condition{
			If: "data.region == 'EU' and data.is_pii == true",
		},
		Actions: []middleware.Action{
			{Ensure: "destination.region == 'EU'"},
		},
		ViolationResponse: middleware.ViolationResponse{
			BlockTransfer: true,
			Alert:         "DataPrivacyTeam",
		},
	},
	{
		RuleID: "US_PII_REGION_LOCK",
		Condition: middleware.Condition{
			If: "data.region == 'US' and data.is_pii == true",
		},
		Actions: []middleware.Action{
			{Ensure: "destination.region == 'US'"},
		},
		ViolationResponse: middleware.ViolationResponse{
			BlockTransfer: true,
			Alert:         "DataPrivacyTeam",
		},
	},
}

func main() {
	// Define command-line flags
	configPath := flag.String("config", "./config.json", "Path to configuration file")
	flag.Parse()

	// Capture build information
	buildInfo := fmt.Sprintf("%s (%s) - Go %s", serviceVersion, releaseID, runtime.Version())

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logr, err := logger.New(cfg.Log)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// Initialize metrics
	metricCollector := metrics.NewMetrics()

	// Initialize the cache
	var cacheImpl interface{}
	if cfg.Cache.Type == "redis" {
		logr.Info("Initializing Redis cache")
		redisCache, err := cache.NewRedisCache(cfg.Cache)
		if err != nil {
			logr.WithField("error", err.Error()).Error("Failed to initialize Redis cache")
			log.Fatalf("Failed to initialize Redis cache: %v", err)
		}
		cacheImpl = redisCache
	} else {
		logr.Info("Initializing in-memory cache")
		cacheImpl = cache.NewRedisLikeCache()
	}

	// Initialize components
	identityRouter := router.NewIdentityRouter(cacheImpl.(*cache.RedisLikeCache))
	ruleEngine := middleware.NewRuleEngine(sampleRules)

	// Initialize authentication
	jwtManager := auth.NewJWTManager(cfg.Auth)

	// Create an agreement
	agreement := &agreements.Agreement{
		AgreementID: "DATA_GOVERNANCE_001",
		Rules:       sampleRules,
	}

	// Create a new server mux
	mux := http.NewServeMux()

	// Add middleware stack
	handlerChain := metricCollector.MetricsMiddleware(
		handlers.RequestLogger(logr)(
			handlers.ErrorHandler(logr)(
				jwtManager.Middleware(mux),
			),
		),
	)

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			handlers.SendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"time":   time.Now(),
			"version": "1.0.0",
		})
	})

	// Data flow request endpoint
	mux.HandleFunc("/data/flow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			handlers.SendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

		logr.WithFields(map[string]interface{}{
			"client_ip": r.RemoteAddr,
			"endpoint":  "/data/flow",
		}).Info("Processing data flow request")

		var request router.DataRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			logr.WithField("error", err.Error()).Error("Failed to decode request body")
			handlers.SendJSONError(w, http.StatusBadRequest, "Invalid request format", "invalid_request")
			return
		}

		// Log all requests, even if they'll be blocked later
		// to capture full audit trail
		identityRouter.Route(request)

		// Process the request through rule engine
		start := time.Now()
		compliant, violations := ruleEngine.Evaluate(request)
		metricCollector.RecordRuleEvaluation("all_rules")

		if !compliant {
			// Check if any violations require blocking
			blocked := false
			for _, v := range violations {
				metricCollector.RecordRuleViolation("rule_violation")

				if v.BlockTransfer {
					blocked = true
					break
				}
			}

			if blocked {
				logr.WithFields(map[string]interface{}{
					"job_id":     request.JobID,
					"data_asset": request.DataAsset,
					"violations": violations,
				}).Warn("Data flow blocked due to rule violations")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":     "blocked",
					"violations": violations,
				})
				return
			}

			// Handle non-blocking violations
			ruleEngine.HandleViolations(violations)
		}

		// Generate Merkle proof after processing
		merkleTiming := time.Now()
		dataItems := []interface{}{request}
		tree, err := merkle.NewMerkleTree(dataItems)
		if err != nil {
			logr.WithField("error", err.Error()).Error("Failed to create Merkle tree")
		} else {
			metricCollector.RecordMerkleTreeTime(time.Since(merkleTiming))
			proof := tree.GenerateProof()
			agreements.UpdateProof(agreement, proof)

			logr.WithFields(map[string]interface{}{
				"job_id":      request.JobID,
				"merkle_root": proof.Root,
				"duration_ms": time.Since(start).Milliseconds(),
			}).Info("Data flow processed successfully")

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":      "processed",
				"merkle_root": proof.Root,
				"job_id":      request.JobID,
			})
		}
	})

	// Get logs endpoint
	mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			handlers.SendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"logs": identityRouter.GetLogs(),
		})
	})

	// Get agreement endpoint
	mux.HandleFunc("/agreement", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			handlers.SendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agreement)
	})

	// Initialize health checker
	healthChecker := health.NewChecker(cfg, logr, serviceID, serviceVersion, releaseID)

	// Add health checks
	if cfg.Cache.Type == "redis" {
		redisClient, ok := cacheImpl.(*cache.RedisCache)
		if ok && redisClient != nil {
			healthChecker.AddRedisCheck(redisClient.GetClient(), 30*time.Second)
		}
	}

	// Add disk space check
	healthChecker.AddDiskSpaceCheck("./", 1.0, 60*time.Second)

	// Start health checker
	healthChecker.Start()

	// Set up HTTP handlers
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthChecker.HealthHandler())
	http.HandleFunc("/ready", healthChecker.ReadinessHandler())
	http.HandleFunc("/alive", healthChecker.LivenessHandler())

	// Initialize server with additional timeout settings
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		ReadTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	// Set up graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create a done channel to signal when server is completely shut down
	done := make(chan struct{})

	go func() {
		// Wait for shutdown signal
		sig := <-sigCh
		logr.Info().Str("signal", sig.String()).Msg("Shutdown signal received, gracefully shutting down...")

		// Create a context with timeout for shutdown operations
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Begin shutdown actions in order

		// 1. Stop accepting new connections
		logr.Info().Msg("Stopping HTTP server...")
		if err := server.Shutdown(ctx); err != nil {
			logr.Error().Err(err).Msg("Server shutdown failed")
		}

		// 2. Stop health checker
		logr.Info().Msg("Stopping health checker...")
		healthChecker.Stop()

		// 3. Close cache connections if applicable
		logr.Info().Msg("Closing cache connections...")
		if closer, ok := cacheImpl.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				logr.Warn().Err(err).Msg("Error closing cache connection")
			}
		}

		// 4. Flush any pending logs
		logr.Info().Msg("Flushing logs...")

		// Signal that shutdown is complete
		close(done)
	}()

	// Print startup information
	logr.Info().Int("port", cfg.Server.Port).Str("version", buildInfo).Msg("Starting DGLA Data Governance Pipeline server")

	// Start the server
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logr.Error().Err(err).Msg("Server failed unexpectedly")
		os.Exit(1)
	}

	// Wait for shutdown to complete
	<-done
	logr.Info().Msg("Server shutdown completed successfully")
}
