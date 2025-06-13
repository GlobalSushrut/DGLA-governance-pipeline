#!/bin/bash
set -e

# DGLA Swagger Testing Script
# Tests the Swagger endpoints to verify proper documentation

# Make sure we're in the project root directory
cd "$(dirname "$0")/.."

# Define colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}DGLA Swagger Documentation Test${NC}"
echo "This script will test the Swagger documentation endpoints"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
  echo -e "${RED}Error: Docker is not running or not accessible${NC}"
  exit 1
fi

# Build and start the container in test mode
echo -e "${GREEN}Building and starting test container...${NC}"
docker build -t dgla-swagger-test .

echo -e "${YELLOW}Starting container with Swagger endpoints exposed...${NC}"
CONTAINER_ID=$(docker run -d -p 8081:8081 dgla-swagger-test)

# Function to clean up container on exit
cleanup() {
  echo -e "${YELLOW}Cleaning up test container...${NC}"
  docker stop $CONTAINER_ID > /dev/null
  docker rm $CONTAINER_ID > /dev/null
}

# Set up the cleanup trap
trap cleanup EXIT

# Wait for the server to start
echo -e "${YELLOW}Waiting for server to start...${NC}"
sleep 5

# Test Swagger UI endpoint
echo -e "${BLUE}Testing /docs endpoint (Swagger UI)...${NC}"
DOCS_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/docs)

if [ "$DOCS_STATUS" -eq 200 ]; then
  echo -e "${GREEN}✓ Swagger UI endpoint (/docs) is responding correctly (HTTP $DOCS_STATUS)${NC}"
else
  echo -e "${RED}✗ Swagger UI endpoint (/docs) failed with HTTP $DOCS_STATUS${NC}"
  echo -e "${YELLOW}Checking endpoint content...${NC}"
  curl -s http://localhost:8081/docs | head -n 20
fi

# Test Swagger JSON endpoint
echo -e "${BLUE}Testing /swagger.json endpoint...${NC}"
JSON_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/swagger.json)

if [ "$JSON_STATUS" -eq 200 ]; then
  echo -e "${GREEN}✓ Swagger JSON endpoint (/swagger.json) is responding correctly (HTTP $JSON_STATUS)${NC}"
  
  # Validate JSON format
  JSON_VALID=$(curl -s http://localhost:8081/swagger.json | jq . > /dev/null 2>&1 && echo "true" || echo "false")
  
  if [ "$JSON_VALID" = "true" ]; then
    echo -e "${GREEN}✓ Swagger JSON is well-formed${NC}"
    
    # Check OpenAPI version
    OPENAPI_VERSION=$(curl -s http://localhost:8081/swagger.json | jq -r '.openapi')
    echo -e "${GREEN}✓ OpenAPI version: $OPENAPI_VERSION${NC}"
    
    # Check API title
    API_TITLE=$(curl -s http://localhost:8081/swagger.json | jq -r '.info.title')
    echo -e "${GREEN}✓ API title: $API_TITLE${NC}"
    
    # Count endpoints
    ENDPOINT_COUNT=$(curl -s http://localhost:8081/swagger.json | jq '.paths | length')
    echo -e "${GREEN}✓ Number of endpoints: $ENDPOINT_COUNT${NC}"
  else
    echo -e "${RED}✗ Swagger JSON is not valid JSON${NC}"
    echo -e "${YELLOW}Checking JSON content...${NC}"
    curl -s http://localhost:8081/swagger.json | head -n 20
  fi
else
  echo -e "${RED}✗ Swagger JSON endpoint (/swagger.json) failed with HTTP $JSON_STATUS${NC}"
fi

echo ""
echo -e "${GREEN}Swagger test completed!${NC}"
