#!/bin/bash
set -e

# DGLA Development Helper Script
# This script helps manage common development tasks with Docker

# Make sure we're in the project root directory
cd "$(dirname "$0")/.."

# Load environment variables from .env file if it exists
if [ -f ".env" ]; then
  echo "Loading environment variables from .env file"
  export $(cat .env | grep -v '^#' | xargs)
fi

# Define colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

show_help() {
  echo -e "${BLUE}DGLA Development Helper Script${NC}"
  echo ""
  echo "Usage: $0 [command]"
  echo ""
  echo "Commands:"
  echo "  build        - Build Docker images"
  echo "  start        - Start the DGLA stack"
  echo "  stop         - Stop the DGLA stack"
  echo "  restart      - Restart the DGLA stack"
  echo "  logs         - View logs from the DGLA service"
  echo "  test         - Run tests"
  echo "  swagger-test - Test Swagger endpoints"
  echo "  shell        - Start a shell in the running DGLA container"
  echo "  status       - Show status of containers"
  echo "  clean        - Remove all containers and volumes"
  echo "  help         - Show this help"
}

build() {
  echo -e "${GREEN}Building Docker images...${NC}"
  BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
  COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
  
  docker-compose build --build-arg BUILD_DATE="$BUILD_DATE" --build-arg COMMIT_HASH="$COMMIT_HASH"
}

start() {
  echo -e "${GREEN}Starting DGLA stack...${NC}"
  docker-compose up -d
  echo ""
  echo -e "${GREEN}Services are starting:${NC}"
  echo -e "  - DGLA API:          ${BLUE}http://localhost:8081${NC}"
  echo -e "  - Swagger UI:        ${BLUE}http://localhost:8081/docs${NC}"
  echo -e "  - Swagger JSON:      ${BLUE}http://localhost:8081/swagger.json${NC}"
  echo -e "  - Prometheus:        ${BLUE}http://localhost:9090${NC}"
  echo -e "  - Grafana:           ${BLUE}http://localhost:3000${NC} (admin/admin)"
}

stop() {
  echo -e "${YELLOW}Stopping DGLA stack...${NC}"
  docker-compose down
}

restart() {
  stop
  start
}

logs() {
  docker-compose logs -f dgla
}

test() {
  echo -e "${GREEN}Running tests...${NC}"
  docker-compose run --rm dgla go test -v ./...
}

swagger_test() {
  echo -e "${GREEN}Testing Swagger endpoints...${NC}"
  echo -e "${BLUE}Testing /docs endpoint:${NC}"
  curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/docs
  echo ""
  
  echo -e "${BLUE}Testing /swagger.json endpoint:${NC}"
  curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/swagger.json
  echo ""
  
  echo -e "${BLUE}Swagger JSON content:${NC}"
  curl -s http://localhost:8081/swagger.json | jq '.' || echo "Failed to get Swagger JSON"
}

shell() {
  docker-compose exec dgla sh
}

status() {
  docker-compose ps
}

clean() {
  echo -e "${RED}Warning: This will remove all containers and volumes!${NC}"
  read -p "Are you sure? (y/N): " confirm
  if [[ $confirm =~ ^[Yy]$ ]]; then
    docker-compose down -v
    echo -e "${GREEN}All containers and volumes removed${NC}"
  else
    echo "Operation canceled"
  fi
}

# Main logic
case "$1" in
  build)
    build
    ;;
  start)
    start
    ;;
  stop)
    stop
    ;;
  restart)
    restart
    ;;
  logs)
    logs
    ;;
  test)
    test
    ;;
  swagger-test)
    swagger_test
    ;;
  shell)
    shell
    ;;
  status)
    status
    ;;
  clean)
    clean
    ;;
  help)
    show_help
    ;;
  *)
    show_help
    exit 1
    ;;
esac

exit 0
