#!/bin/bash
# DGLA Infrastructure Deployment Script

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== DGLA Production Infrastructure Deployment ===${NC}"
echo "This script will deploy the complete DGLA infrastructure stack"

# Create namespace if it doesn't exist
echo -e "\n${YELLOW}Creating DGLA namespace...${NC}"
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: dgla
  labels:
    name: dgla
EOF

# Apply MongoDB components
echo -e "\n${YELLOW}Deploying MongoDB with Merkle Tree verification...${NC}"
kubectl apply -f infrastructure/db/mongodb-secrets.yaml
kubectl apply -f infrastructure/db/mongodb-statefulset.yaml
kubectl apply -f infrastructure/db/mongodb-service.yaml
kubectl apply -f infrastructure/db/mongodb-merkle-implementation.yaml
kubectl apply -f infrastructure/db/client-db-connector.yaml

# Apply CDN components
echo -e "\n${YELLOW}Deploying CDN infrastructure...${NC}"
kubectl apply -f infrastructure/cdn/cdn-deployment.yaml
kubectl apply -f infrastructure/cdn/cdn-service.yaml
kubectl apply -f infrastructure/cdn/cdn-config.yaml

# Apply monitoring infrastructure
echo -e "\n${YELLOW}Deploying monitoring stack...${NC}"
kubectl apply -f infrastructure/monitoring/prometheus-deployment.yaml
kubectl apply -f infrastructure/monitoring/prometheus-config.yaml
kubectl apply -f infrastructure/monitoring/grafana-deployment.yaml
kubectl apply -f infrastructure/monitoring/grafana-config.yaml

# Apply node management
echo -e "\n${YELLOW}Deploying node management system...${NC}"
kubectl apply -f infrastructure/node-management/node-manager-secret.yaml
kubectl apply -f infrastructure/node-management/node-manager-daemonset.yaml
kubectl apply -f infrastructure/node-management/control-service-deployment.yaml
kubectl apply -f infrastructure/node-management/control-service.yaml

# Apply alerting and SLA components
echo -e "\n${YELLOW}Deploying alerting system with SLA management...${NC}"
kubectl apply -f infrastructure/alerting/alertmanager-deployment.yaml
kubectl apply -f infrastructure/alerting/alertmanager-config.yaml
kubectl apply -f infrastructure/alerting/sla-service-deployment.yaml
kubectl apply -f infrastructure/alerting/sla-service.yaml
kubectl apply -f infrastructure/alerting/sla-secrets.yaml
kubectl apply -f infrastructure/alerting/sla-config.yaml
kubectl apply -f infrastructure/alerting/sla-operator-rbac.yaml
kubectl apply -f infrastructure/alerting/sla-operator-deployment.yaml
kubectl apply -f infrastructure/alerting/custom-sla-config.yaml

# Apply SLA example (optional)
if [ "$1" == "--with-examples" ]; then
  echo -e "\n${YELLOW}Deploying SLA examples...${NC}"
  kubectl apply -f infrastructure/alerting/examples/financial-customer-sla.yaml
  kubectl apply -f infrastructure/alerting/examples/healthcare-customer-sla.yaml
fi

# Apply main API and configuration
echo -e "\n${YELLOW}Deploying main DGLA API and configuration...${NC}"
kubectl apply -f infrastructure/complete-infrastructure.yaml

# Wait for deployments
echo -e "\n${YELLOW}Waiting for core services to become ready...${NC}"
kubectl wait --for=condition=available --timeout=300s deployment/dgla-api -n dgla
kubectl wait --for=condition=available --timeout=300s deployment/dgla-merkle-service -n dgla
kubectl wait --for=condition=available --timeout=300s deployment/dgla-client-db-connector -n dgla

echo -e "\n${GREEN}=== DGLA Infrastructure Deployment Complete ===${NC}"
echo "Access the DGLA API at: https://api.dgla.io/api/v1/"
echo "Access Grafana monitoring at: https://api.dgla.io/monitoring/"
echo "Access the SLA dashboard at: https://api.dgla.io/sla/"

# Test connectivity
echo -e "\n${YELLOW}Testing API connectivity...${NC}"
if curl -s -o /dev/null -w "%{http_code}" https://api.dgla.io/api/v1/health | grep -q "200"; then
  echo -e "${GREEN}API is healthy!${NC}"
else
  echo -e "${RED}API connectivity test failed. Check deployment logs.${NC}"
fi
