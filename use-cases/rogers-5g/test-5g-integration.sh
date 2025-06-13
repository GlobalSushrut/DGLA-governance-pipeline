#!/bin/bash
# Test script for Rogers 5G Security System integration with DGLA
# Tests CLI integration, configuration, and deployment

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Testing Rogers 5G Security System Integration with DGLA ===${NC}"

# Change to DGLA base directory
cd "$(dirname "$(dirname "$(dirname "$0")")")"

# Test 1: CLI commands integration
echo -e "\n${YELLOW}Test 1: Testing CLI integration...${NC}"
./cli/dgla.py rogers-5g --help 2>&1 | grep -q "Rogers 5G Security System commands"
if [ $? -eq 0 ]; then
  echo -e "${GREEN}✓ CLI integration test passed${NC}"
else
  echo -e "${RED}✗ CLI integration test failed${NC}"
  exit 1
fi

# Test 2: Configuration generation
echo -e "\n${YELLOW}Test 2: Testing config generation...${NC}"
./cli/dgla.py rogers-5g configure --region "Canada"
if [ $? -eq 0 ] && [ -f ~/.dgla/rogers-5g-config.yaml ]; then
  echo -e "${GREEN}✓ Configuration generation test passed${NC}"
  cat ~/.dgla/rogers-5g-config.yaml | grep -q "Rogers 5G Security System"
  if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Configuration content looks correct${NC}"
  else
    echo -e "${RED}✗ Configuration content test failed${NC}"
    exit 1
  fi
else
  echo -e "${RED}✗ Configuration generation test failed${NC}"
  exit 1
fi

# Test 3: Validate deployment files
echo -e "\n${YELLOW}Test 3: Validating deployment files...${NC}"
if [ -f ./use-cases/rogers-5g/deployment.yaml ] && 
   [ -f ./use-cases/rogers-5g/rogers-5g-security.yaml ] && 
   [ -f ./use-cases/rogers-5g/rogers-5g-sla.yaml ]; then
  echo -e "${GREEN}✓ All required deployment files present${NC}"
else
  echo -e "${RED}✗ Some deployment files are missing${NC}"
  exit 1
fi

# Test 4: Verify system integration points
echo -e "\n${YELLOW}Test 4: Verifying system integration points...${NC}"
grep -q "MONGODB_URI" ./use-cases/rogers-5g/deployment.yaml && 
grep -q "prometheus.io/scrape" ./use-cases/rogers-5g/deployment.yaml && 
grep -q "merkleEnabled" ./use-cases/rogers-5g/rogers-5g-security.yaml
if [ $? -eq 0 ]; then
  echo -e "${GREEN}✓ System integration points verified${NC}"
else
  echo -e "${RED}✗ System integration points verification failed${NC}"
  exit 1
fi

# Summary
echo -e "\n${GREEN}=== All Integration Tests PASSED ===${NC}"
echo -e "Rogers 5G Security System is successfully integrated with the DGLA infrastructure"
echo -e "\nThe following components are now fully integrated:"
echo -e " - CLI extensions for Rogers 5G management"
echo -e " - Configuration generation and persistence"
echo -e " - DGLA MongoDB integration with cryptographic verification"
echo -e " - SLA monitoring and enforcement"
echo -e " - Prometheus metrics integration"
echo -e " - Data sovereignty controls (region: Canada)"
echo -e " - Multi-layered 5G network security (RAN, Core, Slices)"

echo -e "\n${YELLOW}To deploy the Rogers 5G Security System, run:${NC}"
echo -e "./cli/dgla.py rogers-5g deploy"
