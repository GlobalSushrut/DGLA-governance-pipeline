#!/bin/bash
set -e

# DGLA Kubernetes Deployment Script
# This script helps deploy the DGLA stack to Kubernetes

# Make sure we're in the project root directory
cd "$(dirname "$0")/.."

# Define colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Default values
NAMESPACE="dgla"
REGISTRY=""
VERSION="latest"
KUBECONFIG=""
CONTEXT=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --registry)
      REGISTRY="$2"
      shift
      shift
      ;;
    --version)
      VERSION="$2"
      shift
      shift
      ;;
    --kubeconfig)
      KUBECONFIG="$2"
      shift
      shift
      ;;
    --context)
      CONTEXT="$2"
      shift
      shift
      ;;
    --help)
      echo -e "${BLUE}DGLA Kubernetes Deployment Script${NC}"
      echo ""
      echo "Usage: $0 [options]"
      echo ""
      echo "Options:"
      echo "  --registry    Docker registry URL (required)"
      echo "  --version     Image version tag (default: latest)"
      echo "  --kubeconfig  Path to kubeconfig file"
      echo "  --context     Kubernetes context to use"
      echo "  --help        Show this help"
      echo ""
      echo "Example:"
      echo "  $0 --registry example.azurecr.io --version 1.0.0"
      exit 0
      ;;
    *)
      echo "Unknown option: $key"
      exit 1
      ;;
  esac
done

# Validate required parameters
if [[ -z "$REGISTRY" ]]; then
  echo -e "${RED}Error: --registry is required${NC}"
  echo "Run $0 --help for usage information"
  exit 1
fi

# Set kubeconfig if provided
if [[ -n "$KUBECONFIG" ]]; then
  export KUBECONFIG="$KUBECONFIG"
fi

# Set context if provided
if [[ -n "$CONTEXT" ]]; then
  kubectl config use-context "$CONTEXT"
fi

# Check kubectl connection
echo -e "${BLUE}Checking connection to Kubernetes cluster...${NC}"
if ! kubectl cluster-info > /dev/null; then
  echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
  exit 1
fi

# Update YAML files with registry and version
REGISTRY_URL="$REGISTRY"
sed -i "s|\${REGISTRY_URL}|$REGISTRY_URL|g" kubernetes/dgla-deployment.yaml
sed -i "s|\${VERSION}|$VERSION|g" kubernetes/dgla-deployment.yaml

echo -e "${BLUE}Starting deployment process...${NC}"

# Create namespace if it doesn't exist
echo -e "${GREEN}Creating namespace...${NC}"
kubectl apply -f kubernetes/dgla-namespace.yaml

# Create secrets (prompting for sensitive values)
echo -e "${YELLOW}Creating secrets...${NC}"
echo -n "Enter JWT secret for production (leave empty to generate): "
read -s JWT_SECRET
echo ""
if [[ -z "$JWT_SECRET" ]]; then
  JWT_SECRET=$(openssl rand -base64 32)
  echo "Generated secure JWT secret"
fi

echo -n "Enter Redis password for production (leave empty to generate): "
read -s REDIS_PASSWORD
echo ""
if [[ -z "$REDIS_PASSWORD" ]]; then
  REDIS_PASSWORD=$(openssl rand -base64 16)
  echo "Generated secure Redis password"
fi

# Create a temporary file for secrets with actual values
TEMP_SECRET=$(mktemp)
cat kubernetes/dgla-secrets.yaml | \
  sed "s/REPLACE_WITH_SECURE_JWT_SECRET/$JWT_SECRET/g" | \
  sed "s/REPLACE_WITH_SECURE_REDIS_PASSWORD/$REDIS_PASSWORD/g" > "$TEMP_SECRET"

kubectl apply -f "$TEMP_SECRET"
rm "$TEMP_SECRET"

# Apply ConfigMap
echo -e "${GREEN}Applying ConfigMap...${NC}"
kubectl apply -f kubernetes/dgla-configmap.yaml

# Apply persistent volume claim
echo -e "${GREEN}Creating storage...${NC}"
kubectl apply -f kubernetes/dgla-pvc.yaml

# Deploy Redis
echo -e "${GREEN}Deploying Redis...${NC}"
kubectl apply -f kubernetes/dgla-redis.yaml

echo -e "${YELLOW}Waiting for Redis to be ready...${NC}"
kubectl -n "$NAMESPACE" wait --for=condition=ready pod -l app=dgla-redis --timeout=120s

# Deploy DGLA application
echo -e "${GREEN}Deploying DGLA application...${NC}"
kubectl apply -f kubernetes/dgla-deployment.yaml
kubectl apply -f kubernetes/dgla-service.yaml

echo -e "${YELLOW}Waiting for DGLA pods to be ready...${NC}"
kubectl -n "$NAMESPACE" wait --for=condition=ready pod -l app=dgla --timeout=120s

# Deploy ingress
echo -e "${GREEN}Deploying ingress...${NC}"
kubectl apply -f kubernetes/dgla-ingress.yaml

# Apply autoscaling
echo -e "${GREEN}Configuring autoscaling...${NC}"
kubectl apply -f kubernetes/dgla-hpa.yaml

echo -e "${GREEN}Deployment completed successfully!${NC}"
echo ""
echo -e "${BLUE}API will be available at:${NC} https://api.dgla.example.com"
echo -e "${BLUE}Swagger UI:${NC} https://api.dgla.example.com/docs"
echo -e "${BLUE}Swagger JSON:${NC} https://api.dgla.example.com/swagger.json"

# Restore the placeholder variables in the files
sed -i "s|$REGISTRY_URL|\${REGISTRY_URL}|g" kubernetes/dgla-deployment.yaml
sed -i "s|$VERSION|\${VERSION}|g" kubernetes/dgla-deployment.yaml

echo -e "${YELLOW}Note: Update the hostname in kubernetes/dgla-ingress.yaml with your actual domain name${NC}"
