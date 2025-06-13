#!/bin/bash
# Run script for DGLA Cybersecurity Demo
# This script runs the cybersecurity demo against the real infrastructure

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

# Set up Python virtual environment if needed
if [ ! -d "venv" ]; then
    echo "Setting up Python virtual environment..."
    python3 -m venv venv
fi

# Activate virtual environment
echo "Activating virtual environment..."
source venv/bin/activate

# Install dependencies
echo "Installing dependencies..."
pip install -r requirements.txt

# Export environment variables for local testing
export REDIS_HOST=localhost
export REDIS_PORT=6379
export PROMETHEUS_URL=http://localhost:9090
export GRAFANA_URL=http://localhost:3000

echo
echo "Starting cybersecurity demonstration..."
echo "This will demonstrate how DGLA infrastructure provides advanced"
echo "security capabilities beyond traditional solutions."
echo

# Run the demonstration
python run_security_scenarios.py

echo
echo "Demo complete! You can view the audit logs in ../logs/audit_trail.log"
echo "To explore the metrics in Prometheus, visit: http://localhost:9090"
echo "To view security dashboards in Grafana, visit: http://localhost:3000"
