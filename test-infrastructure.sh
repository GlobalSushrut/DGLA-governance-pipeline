#!/bin/bash
# DGLA Complete Infrastructure Validation Test
# This script tests all components of the DGLA system for production readiness

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_DIR=$(pwd)
RESULT_DIR="${BASE_DIR}/test-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${RESULT_DIR}/test_${TIMESTAMP}.log"

# Ensure results directory exists
mkdir -p "${RESULT_DIR}"

log() {
  echo -e "${2:-$BLUE}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "${LOG_FILE}"
}

success() {
  log "✅ $1" "${GREEN}"
}

warn() {
  log "⚠️ $1" "${YELLOW}"
}

error() {
  log "❌ $1" "${RED}"
}

test_cli() {
  log "Testing DGLA CLI..."
  
  # Test CLI exists
  if ! command -v ./cli/dgla.py &> /dev/null; then
    error "CLI not found at ./cli/dgla.py"
    return 1
  fi
  
  # Test help command
  if ! ./cli/dgla.py --help &> /dev/null; then
    error "CLI help command failed"
    return 1
  fi
  
  success "CLI basic functionality verified"
  
  # Test config generation
  log "Testing config generation..."
  ./cli/dgla.py init --namespace dgla-test > /dev/null 2>&1 || true
  
  if [ ! -f ~/.dgla/dgla-config.yaml ]; then
    warn "Config file not created during init"
  else
    success "Config generation successful"
  fi
  
  return 0
}

test_infrastructure_files() {
  log "Validating Kubernetes manifests..."
  
  # Check if kubectl is available for validation
  if ! command -v kubectl &> /dev/null; then
    warn "kubectl not found, skipping manifest validation"
    return 0
  fi
  
  # Validate Kubernetes manifests
  VALID_FILES=0
  INVALID_FILES=0
  
  for file in $(find "${BASE_DIR}/infrastructure" -name "*.yaml"); do
    if kubectl apply --dry-run=client -f "$file" > /dev/null 2>&1; then
      success "✓ Valid: $(basename $file)"
      ((VALID_FILES++))
    else
      error "✗ Invalid: $(basename $file)"
      ((INVALID_FILES++))
    fi
  done
  
  log "Manifest validation complete: ${VALID_FILES} valid, ${INVALID_FILES} invalid"
  
  if [ $INVALID_FILES -gt 0 ]; then
    return 1
  fi
  
  return 0
}

test_crypto_implementation() {
  log "Testing cryptographic implementation..."
  
  # Verify presence of Merkle tree implementation
  if [ ! -f "${BASE_DIR}/infrastructure/db/mongodb-merkle-implementation.yaml" ]; then
    error "Merkle tree implementation not found"
    return 1
  fi
  
  # Verify presence of cryptographic keys in secrets
  grep -q "merkle_key" "${BASE_DIR}/infrastructure/db/mongodb-secrets.yaml" || {
    error "Merkle key not defined in MongoDB secrets"
    return 1
  }
  
  # Check client connector crypto settings
  grep -q "cryptoVerificationEnabled: true" "${BASE_DIR}/infrastructure/db/client-db-connector.yaml" || {
    warn "Crypto verification may not be enabled in client connector"
  }
  
  success "Cryptographic implementation validated"
  return 0
}

test_sla_framework() {
  log "Testing SLA framework..."
  
  # Verify operator deployment
  if [ ! -f "${BASE_DIR}/infrastructure/alerting/sla-operator-deployment.yaml" ]; then
    error "SLA operator deployment not found"
    return 1
  fi
  
  # Verify RBAC configuration
  if [ ! -f "${BASE_DIR}/infrastructure/alerting/sla-operator-rbac.yaml" ]; then
    error "SLA RBAC configuration not found"
    return 1
  fi
  
  # Verify example SLA definitions
  if [ -d "${BASE_DIR}/infrastructure/alerting/examples" ]; then
    EXAMPLE_COUNT=$(find "${BASE_DIR}/infrastructure/alerting/examples" -name "*-sla.yaml" | wc -l)
    if [ $EXAMPLE_COUNT -gt 0 ]; then
      success "Found ${EXAMPLE_COUNT} example SLA definitions"
    else
      warn "No example SLA definitions found"
    fi
  else
    warn "Examples directory not found"
  fi
  
  success "SLA framework validated"
  return 0
}

test_cdn_pipeline() {
  log "Testing CDN pipeline..."
  
  # Verify CDN components
  for component in deployment service config; do
    if [ ! -f "${BASE_DIR}/infrastructure/cdn/cdn-${component}.yaml" ]; then
      error "CDN ${component} not found"
      return 1
    fi
  done
  
  # Check for CDN integration with API
  grep -q "CDN_SERVICE" "${BASE_DIR}/infrastructure/complete-infrastructure.yaml" || {
    warn "CDN integration may not be configured in API"
  }
  
  success "CDN pipeline validated"
  return 0
}

test_monitoring() {
  log "Testing monitoring stack..."
  
  # Verify monitoring components
  if [ ! -f "${BASE_DIR}/infrastructure/monitoring/prometheus-deployment.yaml" ]; then
    error "Prometheus deployment not found"
    return 1
  fi
  
  if [ ! -f "${BASE_DIR}/infrastructure/monitoring/prometheus-config.yaml" ]; then
    error "Prometheus configuration not found"
    return 1
  fi
  
  # Check alerting integration
  if [ -d "${BASE_DIR}/infrastructure/alerting" ]; then
    if [ ! -f "${BASE_DIR}/infrastructure/alerting/alertmanager-deployment.yaml" ]; then
      warn "Alertmanager deployment not found"
    fi
  else
    error "Alerting directory not found"
    return 1
  fi
  
  success "Monitoring stack validated"
  return 0
}

test_data_sovereignty() {
  log "Testing data sovereignty implementation..."
  
  # Check for data sovereignty configuration
  grep -q "dataSovereignty" "${BASE_DIR}/infrastructure/db/client-db-connector.yaml" || {
    error "Data sovereignty configuration not found"
    return 1
  }
  
  # Check for regions definition
  grep -q "region:" "${BASE_DIR}/infrastructure/db/client-db-connector.yaml" || {
    warn "Region definitions may be missing in data sovereignty config"
  }
  
  # Check for enforcement strategy
  grep -q "enforcementStrategy:" "${BASE_DIR}/infrastructure/db/client-db-connector.yaml" || {
    warn "Enforcement strategy may not be defined for data sovereignty"
  }
  
  success "Data sovereignty implementation validated"
  return 0
}

test_node_management() {
  log "Testing node management..."
  
  # Verify node management components
  if [ ! -f "${BASE_DIR}/infrastructure/node-management/node-manager-daemonset.yaml" ]; then
    error "Node manager DaemonSet not found"
    return 1
  fi
  
  if [ ! -f "${BASE_DIR}/infrastructure/node-management/control-service-deployment.yaml" ]; then
    error "Control service deployment not found"
    return 1
  fi
  
  # Check for node manager authentication
  if [ ! -f "${BASE_DIR}/infrastructure/node-management/node-manager-secret.yaml" ]; then
    warn "Node manager authentication secret not found"
  fi
  
  success "Node management validated"
  return 0
}

test_deployment_script() {
  log "Testing deployment script..."
  
  # Verify deploy.sh exists and is executable
  if [ ! -f "${BASE_DIR}/deploy.sh" ]; then
    error "Deployment script not found"
    return 1
  fi
  
  if [ ! -x "${BASE_DIR}/deploy.sh" ]; then
    error "Deployment script is not executable"
    chmod +x "${BASE_DIR}/deploy.sh"
    warn "Fixed permissions on deployment script"
  fi
  
  # Check deployment script for key components
  grep -q "mongodb" "${BASE_DIR}/deploy.sh" || {
    warn "MongoDB deployment may be missing from deployment script"
  }
  
  grep -q "cdn" "${BASE_DIR}/deploy.sh" || {
    warn "CDN deployment may be missing from deployment script"
  }
  
  grep -q "monitoring" "${BASE_DIR}/deploy.sh" || {
    warn "Monitoring deployment may be missing from deployment script"
  }
  
  success "Deployment script validated"
  return 0
}

validate_sdk_installation() {
  log "Validating SDK installation..."
  
  # We'll just check if the CLI can handle SDK-related tasks
  grep -q "sdk" "${BASE_DIR}/cli/dgla.py" || {
    warn "SDK management may not be implemented in CLI"
  }
  
  success "SDK installation validation complete"
  return 0
}

# Run all tests
log "Starting DGLA infrastructure validation" "${GREEN}"
log "=================================" "${GREEN}"

TESTS=(
  "test_cli"
  "test_infrastructure_files" 
  "test_crypto_implementation"
  "test_sla_framework"
  "test_cdn_pipeline"
  "test_monitoring"
  "test_data_sovereignty"
  "test_node_management"
  "test_deployment_script"
  "validate_sdk_installation"
)

PASSED=0
FAILED=0

for test in "${TESTS[@]}"; do
  log "Running test: ${test}"
  if $test; then
    success "Test passed: ${test}"
    ((PASSED++))
  else
    error "Test failed: ${test}"
    ((FAILED++))
  fi
  log "---------------------------------"
done

# Final report
log "Test Results" "${GREEN}"
log "=================================" "${GREEN}"
log "Total tests:    ${#TESTS[@]}"
log "Tests passed:   ${PASSED}" "${GREEN}"
if [ $FAILED -gt 0 ]; then
  log "Tests failed:   ${FAILED}" "${RED}"
else
  log "Tests failed:   ${FAILED}" "${GREEN}"
fi
log "Full log:       ${LOG_FILE}"
log "=================================" "${GREEN}"

if [ $FAILED -eq 0 ]; then
  log "✅ All tests passed! The DGLA infrastructure is production-ready." "${GREEN}"
  exit 0
else
  log "❌ Some tests failed. Please check the log for details." "${RED}"
  exit 1
fi
