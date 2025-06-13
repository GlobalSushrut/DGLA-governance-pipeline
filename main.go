package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/umesh/dgla/agreements"
	"github.com/umesh/dgla/api"
	"github.com/umesh/dgla/auth"
	"github.com/umesh/dgla/cache"
	"github.com/umesh/dgla/chainlog"
	"github.com/umesh/dgla/config"
	"github.com/umesh/dgla/health"
	"github.com/umesh/dgla/logger"
	"github.com/umesh/dgla/merkle"
	"github.com/umesh/dgla/metrics"
	"github.com/umesh/dgla/middleware"
	"github.com/umesh/dgla/middleware/handlers"
	"github.com/umesh/dgla/router"
	"github.com/umesh/dgla/verifier"
)

const (
	serviceVersion = "1.0.0"
	serviceID     = "dgla-governance-pipeline"
	releaseID     = "2025.06.11"
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

	// Initialize logger
	logr, err := logger.New(cfg.Log)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

	// Initialize metrics
	metricCollector := metrics.NewMetrics()

	// Initialize the cache
	var cacheImpl interface{}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
	if cfg.Cache.Type == "redis" {
		logr.Info("Initializing Redis cache")
		redisCache, err := cache.NewRedisCache(cfg.Cache)
		if err != nil {
			logr.WithField("error", err.Error()).Error("Failed to initialize Redis cache")
			log.Fatalf("Failed to initialize Redis cache: %v", err)
		}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
		cacheImpl = redisCache
	} else {
		logr.Info("Initializing in-memory cache")
		cacheImpl = cache.NewRedisLikeCache()
	}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
	
	// Initialize storage paths
	dataDir := cfg.Storage.DataDir
	if dataDir == "" {
		dataDir = "./data"
	}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
	
	exportDir := filepath.Join(dataDir, "exports")
	chainlogDir := filepath.Join(dataDir, "chainlog")
	
	// Ensure directories exist
	os.MkdirAll(exportDir, 0755)
	os.MkdirAll(chainlogDir, 0755)

	// Initialize components
	identityRouter := router.NewIdentityRouter(cacheImpl.(*cache.RedisLikeCache))
	ruleEngine := middleware.NewRuleEngine(sampleRules)

	// Initialize authentication
	jwtManager := auth.NewJWTManager(cfg.Auth)
	
	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(metricCollector)
	
	// Initialize ZK Packet Verifier
	logr.Info("Initializing ZK Packet Verifier")
	zkVerifier := verifier.NewZKPacketVerifier()
	
	// Initialize ChainLog Engine
	logr.Info("Initializing ChainLog Engine")
	chainlogEngine, err := chainlog.NewChainLogEngine(chainlogDir)
	if err != nil {
		logr.WithField("error", err.Error()).Error("Failed to initialize ChainLog Engine")
		log.Fatalf("Failed to initialize ChainLog Engine: %v", err)
	}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

	// Create an agreement
	agreement := &agreements.Agreement{
		AgreementID: "DATA_GOVERNANCE_001",
		Rules:       sampleRules,
	}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

	// Create a new server mux
	mux := http.NewServeMux()

	// Add middleware stack
	handlerChain := metricCollector.MetricsMiddleware(
		handlers.RequestLogger(logr)(
			handlers.ErrorHandler(logr)(
				jwtManager.Middleware(
					jwtManager.RoleAuthMiddleware(
						rateLimiter.RateLimit(mux),
					),
				),
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "healthy",
			"time":   time.Now(),
			"version": serviceVersion,
			"release": releaseID,
		})
	})
	
	// Initialize API handlers
	verifyHandler := api.NewVerifyHandler(zkVerifier, metricCollector)
	chainlogHandler := api.NewChainLogHandler(chainlogEngine, metricCollector)
	exportHandler := api.NewExportHandler(chainlogEngine, metricCollector, exportDir)
	
	// ZK Packet verification endpoint
	mux.Handle("/verify", verifyHandler)
	
	// ChainLog endpoints
	mux.HandleFunc("/logs", chainlogHandler.HandleLogs)
	mux.HandleFunc("/anchor", chainlogHandler.HandleAnchor)
	
	// Audit export endpoint
	mux.HandleFunc("/export", exportHandler.HandleExport)
	
	// Authentication endpoint
	mux.HandleFunc("/auth/login", jwtManager.LoginHandler())

	// Data flow request endpoint
	mux.HandleFunc("/data/flow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			handlers.SendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
			}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

			// Handle non-blocking violations
			ruleEngine.HandleViolations(violations)
		}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

		// Generate Merkle proof after processing
		merkleTiming := time.Now()
		dataItems := []interface{}{request}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
	})

	// Legacy logs endpoint (router logs - to be deprecated)
	mux.HandleFunc("/router/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			handlers.SendJSONError(w, http.StatusMethodNotAllowed, "Method not allowed", "method_not_allowed")
			return
		}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
	}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

	// Add disk space check
	healthChecker.AddDiskSpaceCheck("./", 1.0, 60*time.Second)

	// Start health checker
	healthChecker.Start()

	// Set up HTTP handlers with middleware
	http.Handle("/", handlerChain)
	
	// Add Swagger API Documentation endpoint
	http.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		
		// Serve Swagger UI HTML
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="UTF-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
			<title>DGLA API Documentation</title>
			<link href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.3.1/swagger-ui.css" rel="stylesheet">
			<style>body{margin:0;}</style>
		</head>
		<body>
			<div id="swagger-ui"></div>
			<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.3.1/swagger-ui-bundle.js"></script>
			<script>
				window.onload = function() {
					const ui = SwaggerUIBundle({
						url: "/swagger.json",
						dom_id: '#swagger-ui',
						deepLinking: true,
						presets: [
							SwaggerUIBundle.presets.apis
						],
					});
				};
			</script>
		</body>
		</html>
		`))
	})
	
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>DGLA API Documentation</title>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<link href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.3.1/swagger-ui.css" rel="stylesheet">
		</head>
		<body>
			<div id="swagger-ui"></div>
			<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5.3.1/swagger-ui-bundle.js"></script>
			<script>
			window.onload = function() {
				window.ui = SwaggerUIBundle({
					url: "/swagger.json",
					dom_id: '#swagger-ui',
					deepLinking: true,
					presets: [
						SwaggerUIBundle.presets.apis
					],
				});
			};
			</script>
		</body>
		</html>
		`))
	})
	
	// Serve Swagger JSON
	http.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		
		// Generate Swagger documentation
		swaggerDoc := generateSwaggerDoc()
		
		// Serve as JSON
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(swaggerDoc)
	})
	
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(generateSwaggerDoc())
	})

	// Initialize server with additional timeout settings
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		ReadTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
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

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
		}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

		// 4. Flush any pending logs
		logr.Info().Msg("Flushing logs...")

		// Signal that shutdown is complete
		close(done)
	}()

	// Print startup information
	logr.Info().Int("port", cfg.Server.Port).Str("version", buildInfo).Msg("Starting DGLA Data Governance Pipeline server")
	
	// Add API Documentation info
	logr.Info().Msg("API Documentation available at /docs")
	
	// Add API endpoints info
	logr.Info().Msg("Available endpoints:")
	logr.Info().Msg("  * /verify - ZK Packet verification (POST)")
	logr.Info().Msg("  * /logs - Audit log retrieval (GET) and creation (POST)")
	logr.Info().Msg("  * /anchor - Anchor ChainLog to blockchain (POST)")
	logr.Info().Msg("  * /export - Export audit logs (GET, POST)")
	logr.Info().Msg("  * /auth/login - Authenticate users (POST)")
	logr.Info().Msg("  * /metrics - Prometheus metrics (GET)")
	logr.Info().Msg("  * /health, /ready, /alive - Health check endpoints (GET)")

	// Start the server
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logr.Error().Err(err).Msg("Server failed unexpectedly")
		os.Exit(1)
	}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}

	// Wait for shutdown to complete
	<-done
	logr.Info().Msg("Server shutdown completed successfully")
}

// Missing content removed - this was a duplicated fragment of the generateSwaggerDoc function
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title": "DGLA Data Governance Pipeline API",
			"description": "API for the Data Governance and Logging Architecture",
			"version": serviceVersion,
			"contact": map[string]interface{}{
				"name": "DGLA Team",
				"email": "support@dgla.example.com",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url": "/",
				"description": "DGLA API Server",
			},
		},
		"paths": map[string]interface{}{
			"/verify": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Verify a ZK packet with proof",
					"description": "Verifies a zero-knowledge packet proof using specified algorithm",
					"tags": []string{"Verification"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier"}},
					},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"packet": map[string]interface{}{
											"type": "string",
											"description": "The ZK packet to verify",
										},
										"proof": map[string]interface{}{
											"type": "string",
											"description": "The proof for verification",
										},
										"algorithm": map[string]interface{}{
											"type": "string",
											"description": "The verification algorithm to use",
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Successful verification",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"valid": map[string]interface{}{
												"type": "boolean",
											},
											"verification_time_ms": map[string]interface{}{
												"type": "number",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{"description": "Invalid request"},
						"401": map[string]interface{}{"description": "Unauthorized"},
						"403": map[string]interface{}{"description": "Forbidden - insufficient role permissions"},
					},
				},
			},
			"/logs": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Retrieve chainlog entries",
					"description": "Get chainlog entries with optional filtering and pagination",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
					"parameters": []map[string]interface{}{
						{
							"name": "limit",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Maximum number of logs to return",
						},
						{
							"name": "offset",
							"in": "query",
							"schema": map[string]interface{}{"type": "integer"},
							"description": "Log offset for pagination",
						},
						{
							"name": "start_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "Start time for log filtering",
						},
						{
							"name": "end_time",
							"in": "query",
							"schema": map[string]interface{}{"type": "string", "format": "date-time"},
							"description": "End time for log filtering",
						},
					},
				},
				"post": map[string]interface{}{
					"summary": "Add new log entry",
					"description": "Add a new log entry to the chainlog",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"verifier", "admin"}},
					},
				},
			},
			"/anchor": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Anchor chainlog to target",
					"description": "Anchor the chainlog to blockchain or IPFS",
					"tags": []string{"ChainLog"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/export": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Export audit logs",
					"description": "Export audit logs in JSON or PDF format",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
				"post": map[string]interface{}{
					"summary": "Create export job",
					"description": "Create a new export job with specific parameters",
					"tags": []string{"Export"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"auditor", "admin"}},
					},
				},
			},
			"/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Authenticate user",
					"description": "Authenticate user and get JWT token with role claims",
					"tags": []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"username": map[string]interface{}{
											"type": "string",
										},
										"password": map[string]interface{}{
											"type": "string",
										},
										"role": map[string]interface{}{
											"type": "string",
											"enum": []string{"verifier", "auditor", "admin"},
										},
									},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Authentication successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"token": map[string]interface{}{
												"type": "string",
											},
											"expires": map[string]interface{}{
												"type": "string",
												"format": "date-time",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{"description": "Authentication failed"},
					},
				},
			},
			"/metrics": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Prometheus metrics",
					"description": "Get Prometheus metrics for monitoring",
					"tags": []string{"Monitoring"},
					"security": []map[string][]string{
						{"bearerAuth": []string{"admin"}},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"description": "API health check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/ready": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Readiness check",
					"description": "API readiness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
			"/alive": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Liveness check",
					"description": "API liveness check endpoint",
					"tags": []string{"Monitoring"},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type": "http",
					"scheme": "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		"tags": []map[string]string{
			{"name": "Authentication", "description": "Authentication operations"},
			{"name": "Verification", "description": "ZK Packet verification operations"},
			{"name": "ChainLog", "description": "ChainLog operations"},
			{"name": "Export", "description": "Audit log export operations"},
			{"name": "Monitoring", "description": "Monitoring and health check operations"},
		},
	}
}
