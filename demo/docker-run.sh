#!/bin/bash
# Docker-based run script for DGLA Cybersecurity Demo
# This script runs the cybersecurity demo against the real infrastructure using Docker

set -e

echo "🔒 DGLA Cybersecurity Infrastructure Demo"
echo "=========================================="
echo

# Create logs directory if it doesn't exist
mkdir -p ../logs

# Check if infrastructure is running
echo "Checking if infrastructure components are running..."
if ! docker ps | grep -q dgla-redis || ! docker ps | grep -q dgla-prometheus || ! docker ps | grep -q dgla-grafana; then
    echo "❌ Some infrastructure components are not running."
    echo "Please start the infrastructure first with:"
    echo "cd /home/umesh/Documents/DGLA_progects/data-governance-pipeline"
    echo "docker-compose -f test-infra-docker-compose.yml up -d"
    exit 1
fi

echo "✅ Infrastructure is running."

# Build the cybersecurity demo container
echo "Building the cybersecurity demo container..."
docker build -t dgla-cyber-demo .

# Run the container with network access to the existing infrastructure
echo "Running the cybersecurity demonstration..."
docker run --rm --network data-governance-pipeline_dgla-network \
  -e REDIS_HOST=dgla-redis \
  -e REDIS_PORT=6379 \
  -e PROMETHEUS_URL=http://dgla-prometheus:9090 \
  -e GRAFANA_URL=http://dgla-grafana:3000 \
  -v "$(pwd)/../logs:/app/logs" \
  dgla-cyber-demo python run_security_scenarios.py

echo
echo "Demo complete! You can view the audit logs in ../logs/audit_trail.log"
echo "To explore the metrics in Prometheus, visit: http://localhost:9090"
echo "To view security dashboards in Grafana, visit: http://localhost:3000"
