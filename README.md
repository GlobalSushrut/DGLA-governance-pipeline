# DGLA Data Governance and Lineage Tracking System

A production-grade system that automates data governance and lineage tracking with cryptographic proofs. This enterprise-ready solution features JWT authentication, Redis caching, Prometheus metrics, structured logging, and comprehensive health checks to ensure robust performance and reliability in mission-critical environments.

## Project Overview

This production-ready system provides robust data governance capabilities including:

- **Identity Tracking**: Routes and logs all data flows with identity information
- **Rule Enforcement**: Applies configurable rules to data transfers
- **Cryptographic Proofs**: Creates Merkle tree hashes for data integrity and auditability
- **Protocol Specs**: Defines and validates data governance agreements
- **Authentication**: Secure JWT-based authentication with role-based access control
- **Caching**: High-performance Redis caching for improved scalability
- **Metrics**: Prometheus metrics for monitoring system performance
- **Structured Logging**: JSON-formatted logs with configurable levels
- **Health Checks**: Comprehensive health monitoring for production environments

## Architecture

The system consists of these primary components:

1. **Redis Cache**: High-performance storage for data flows and session management
2. **Identity Router**: Tracks all data flows and associated metadata
3. **Rule Engine**: Middleware that enforces data governance rules
4. **Merkle Tree**: Creates cryptographic proofs for data integrity
5. **Agreement Parser**: Handles custom YAML format for data protocols
6. **Authentication Service**: JWT-based authentication and authorization
7. **Metrics Collector**: Prometheus metrics for monitoring and alerting
8. **Health Check System**: Comprehensive monitoring of system components
9. **Config Manager**: Environment-based configuration with overrides

## Directory Structure

```
/data-governance-pipeline
├── auth/            - JWT authentication and authorization
├── benchmark/       - Performance benchmarking suite
│   └── cmd/         - Benchmark command-line tool
├── cache/           - Redis and in-memory caching implementation
├── config/          - Configuration management
├── health/          - Health check system
├── logger/          - Structured logging
├── metrics/         - Prometheus metrics collection
├── middleware/      - Rule engine and request handlers
│   └── handlers/    - Error handling and request logging
├── router/          - Identity tracking and routing
├── merkle/          - Cryptographic proof generation
├── agreements/      - YAML protocol parser and validator
├── tests/           - Test scenarios
├── prometheus/      - Prometheus configuration
├── main.go          - Application entry point
├── Dockerfile       - Container definition
├── docker-compose.yml - Production deployment setup
├── go.mod           - Go module dependencies
└── README.md        - Project documentation
```

## Getting Started

### Prerequisites

- Go 1.18 or higher
- Docker and Docker Compose (for containerized deployment)
- Redis (optional, for production caching)

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd data-governance-pipeline

# Install dependencies
go mod download

# Build the application
go build -o dgla
```

### Running the Application

#### Local Development

```bash
# Start the HTTP server with default configuration
./dgla --config config.json
```

#### Production Deployment

```bash
# Deploy the complete system with Redis, Prometheus, and Grafana
docker-compose up -d
```

The server will start on port 8081 by default (configurable via environment variables).

### Running Tests

```bash
# Run unit tests
go test ./tests

# Run benchmark tests
./benchmark-tool --workload=standard --users=10 --requests=1000

# Generate HTML benchmark report
./benchmark-tool --workload=compliance_audit --output=html --report=report.html
```

## API Endpoints

### Core Endpoints
- `POST /data/flow` - Submit a data flow request for processing
- `GET /logs` - Get all logged data flows
- `GET /agreement` - Get the current data governance agreement

### Monitoring Endpoints
- `GET /health` - Overall health status (for general system health)
- `GET /ready` - Readiness probe (for container orchestration)
- `GET /alive` - Liveness probe (for basic health checks)
- `GET /metrics` - Prometheus metrics endpoint

## Data Flow Example

A compliant data flow request:

```json
{
  "job_id": "MLModel123_EU",
  "data_asset": "customer_pii_table",
  "region": "EU",
  "action": "read",
  "is_pii": true,
  "source": "database",
  "destination": "EU",
  "metadata": {
    "purpose": "model_training"
  }
}
```

## Rule Example

A rule preventing EU PII data from being transferred outside the EU:

```yaml
rule_id: EU_PII_REGION_LOCK
condition:
  if: data.region == 'EU' and data.is_pii == true
actions:
  - ensure: destination.region == 'EU'
violation_response:
  block_transfer: true
  alert: DataPrivacyTeam
```

## Configuration

Configuration can be provided via:

1. **JSON File**: Loaded with `--config` flag
2. **Environment Variables**: Override file settings
3. **Command-line Flags**: Take highest precedence

### Example Environment Variables

```
DGLA_SERVER_PORT=8081
DGLA_LOG_LEVEL=info
DGLA_AUTH_ENABLED=true
DGLA_AUTH_JWT_SECRET=your_secret_key
DGLA_CACHE_TYPE=redis
DGLA_CACHE_REDIS_HOST=redis
```

## Benchmarking

The system includes a comprehensive benchmarking suite for performance testing:

```bash
# Run basic benchmark
./benchmark-tool --server=http://localhost:8081 --users=10 --requests=100

# Run compliance audit workload
./benchmark-tool --workload=compliance_audit --output=html

# Compare with competitors without running actual tests
./benchmark-tool --compare --output=html
```

## Production Features

- **Graceful Shutdown**: Handles termination signals properly
- **Health Monitoring**: Comprehensive health checks for all components
- **Resource Controls**: Container limits for memory and CPU
- **Security**: Runs as non-root user with minimal permissions
- **High Availability**: Configurable for cluster deployment
- **Observability**: Metrics, structured logs, and health endpoints

## Documentation

See additional documentation files:

- [Log & Benchmark Data](log.md): Details about logging and benchmarks
- [Real-World Applications](real_world_applications.md): Industry use cases

## Future Enhancements

- Machine learning for rule optimization
- Web dashboard for monitoring and configuration
- Integration with CI/CD pipelines
- Distributed deployment for high availability
- Support for additional cryptographic proof algorithms

## License

This project is licensed under a proprietary license that restricts use, modification, and distribution.
Contact the repository owner for collaboration and consulting options.
