#!/bin/bash
set -e

# Security hardening
umask 027

# Check environment variables
if [ -z "$LOCKBOX_JWT_SECRET" ]; then
  echo "ERROR: LOCKBOX_JWT_SECRET environment variable is required"
  exit 1
fi

# Create required directories
mkdir -p /app/logs /app/data

# Wait for dependent services
echo "Waiting for registry and nanobond services..."
sleep 5

echo "Starting DGLA Image Lockbox Service..."
exec uvicorn lockbox:app --host 0.0.0.0 --port 8080
