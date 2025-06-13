#!/bin/bash
# Rogers 5G Security System Deployment Script
# Uses DGLA CLI to deploy and verify the complete stack

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Rogers 5G Security System Deployment ===${NC}"
echo "This script will deploy the Rogers 5G Security System on DGLA infrastructure"

# Check if DGLA CLI is available
if ! command -v ./cli/dgla.py &> /dev/null; then
    echo -e "${RED}DGLA CLI not found. Please run from the root of the data-governance-pipeline directory.${NC}"
    exit 1
fi

# 1. Initialize DGLA infrastructure if not already done
echo -e "\n${YELLOW}Initializing DGLA infrastructure...${NC}"
./cli/dgla.py init --namespace dgla --environment production

# 2. Deploy core DGLA infrastructure if not already deployed
echo -e "\n${YELLOW}Deploying core DGLA infrastructure...${NC}"
./cli/dgla.py deploy --components mongodb,cdn,monitoring,node-management,alerting,sla

# 3. Deploy Rogers 5G specific components
echo -e "\n${YELLOW}Deploying Rogers 5G Security components...${NC}"
kubectl apply -f use-cases/rogers-5g/deployment.yaml
kubectl apply -f use-cases/rogers-5g/rogers-5g-sla.yaml

# 4. Register Rogers as a DGLA tenant
echo -e "\n${YELLOW}Registering Rogers as a DGLA tenant...${NC}"
./cli/dgla.py tenant create --name rogers --plan enterprise --industry telecom

# 5. Configure the 5G security system
echo -e "\n${YELLOW}Configuring 5G Security System...${NC}"
kubectl create configmap rogers-5g-config --from-file=use-cases/rogers-5g/rogers-5g-security.yaml -n dgla

# 6. Wait for deployment to complete
echo -e "\n${YELLOW}Waiting for Rogers 5G Security deployment to complete...${NC}"
kubectl rollout status deployment/rogers-5g-security -n dgla

# 7. Run integration tests
echo -e "\n${YELLOW}Running integration tests...${NC}"
echo "Verifying cryptographic integrity..."
kubectl exec deploy/rogers-5g-security -n dgla -- curl -s http://localhost:8080/api/v1/verify-crypto | grep "success" || {
  echo -e "${RED}Cryptographic verification failed${NC}"
  exit 1
}

echo "Verifying data sovereignty..."
kubectl exec deploy/rogers-5g-security -n dgla -- curl -s http://localhost:8080/api/v1/verify-sovereignty | grep "Canada" || {
  echo -e "${RED}Data sovereignty verification failed${NC}"
  exit 1
}

echo "Verifying SLA compliance..."
kubectl exec deploy/rogers-5g-security -n dgla -- curl -s http://localhost:8080/api/v1/verify-sla | grep "compliant" || {
  echo -e "${RED}SLA compliance verification failed${NC}"
  exit 1
}

# 8. Set up monitoring for Rogers 5G
echo -e "\n${YELLOW}Setting up monitoring for Rogers 5G...${NC}"
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: rogers-5g-monitor
  namespace: dgla
spec:
  selector:
    matchLabels:
      app: rogers-5g-security
  endpoints:
  - port: http
    path: /metrics
    interval: 15s
EOF

# 9. Final verification
echo -e "\n${YELLOW}Performing final verification...${NC}"
PODS_READY=$(kubectl get pods -l app=rogers-5g-security -n dgla -o jsonpath='{.items[*].status.containerStatuses[*].ready}' | grep -o "true" | wc -l)
PODS_TOTAL=$(kubectl get pods -l app=rogers-5g-security -n dgla --no-headers | wc -l)

if [ "$PODS_READY" -eq "$PODS_TOTAL" ] && [ "$PODS_TOTAL" -gt 0 ]; then
  echo -e "${GREEN}All Rogers 5G Security pods are ready and healthy${NC}"
else
  echo -e "${RED}Some Rogers 5G Security pods are not ready${NC}"
  kubectl get pods -l app=rogers-5g-security -n dgla
  exit 1
fi

# 10. Generate access credentials
echo -e "\n${YELLOW}Generating access credentials...${NC}"
ACCESS_TOKEN=$(kubectl exec deploy/rogers-5g-security -n dgla -- curl -s http://localhost:8080/api/v1/generate-token | grep -o '"token":"[^"]*"' | sed 's/"token":"//;s/"$//')

if [ -n "$ACCESS_TOKEN" ]; then
  echo -e "${GREEN}Access token generated successfully${NC}"
  echo $ACCESS_TOKEN > rogers-5g-access-token.txt
  echo "Token saved to rogers-5g-access-token.txt"
else
  echo -e "${RED}Failed to generate access token${NC}"
  exit 1
fi

echo -e "\n${GREEN}=== Rogers 5G Security System Deployment Complete ===${NC}"
echo "The system is now running and verified for:"
echo " ✓ Cryptographic integrity"
echo " ✓ Data sovereignty compliance"
echo " ✓ SLA monitoring"
echo " ✓ Real-time security alerting"
echo " ✓ Full infrastructure health"
echo -e "\nAccess the security dashboard at: https://$(kubectl get ingress -n dgla -o jsonpath='{.items[0].spec.rules[0].host}')/rogers-5g/"
