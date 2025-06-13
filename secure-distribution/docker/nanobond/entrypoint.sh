#!/bin/bash
set -e

# Security hardening
umask 027

# Check environment variables
if [ -z "$NANOBOND_STORAGE_PATH" ]; then
  NANOBOND_STORAGE_PATH="/data"
  echo "Using default storage path: $NANOBOND_STORAGE_PATH"
fi

# Create required directories
mkdir -p $NANOBOND_STORAGE_PATH

# Wait for dependent services if specified
if [ ! -z "$WAIT_HOSTS" ]; then
  echo "Waiting for dependent services..."
  sleep 5
fi

echo "Starting NanoBond Ledger Service..."
exec uvicorn nanobond:app --host 0.0.0.0 --port 9090
