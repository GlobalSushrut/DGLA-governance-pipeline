#!/bin/bash
# DGLA Demos Runner
# This script runs all DGLA demos against a configured Kubernetes DGLA infrastructure

set -e

# Colors for pretty output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get API URL from minikube
API_URL="http://$(minikube ip):30081"
echo -e "${BLUE}Using API URL: ${API_URL}${NC}"

# Function to run a demo with proper formatting
run_demo() {
    local demo_path=$1
    local demo_name=$2
    
    echo -e "\n${YELLOW}═════════════════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}Running Demo: ${demo_name}${NC}"
    echo -e "${YELLOW}═════════════════════════════════════════════════════════${NC}"
    
    cd sdk
    python3 "$demo_path" --api-url="$API_URL"
    cd ..
    
    echo -e "\n${GREEN}✓ Demo completed successfully${NC}"
    echo -e "${YELLOW}═════════════════════════════════════════════════════════${NC}"
    
    # Wait for user to press enter to continue
    read -p "Press Enter to run the next demo..." </dev/tty
}

# Check if minikube is running
echo -e "${BLUE}Checking Minikube status...${NC}"
if ! minikube status | grep -q "host: Running"; then
    echo -e "${RED}Error: Minikube is not running. Please start minikube with 'minikube start' and try again.${NC}"
    exit 1
fi

# Check if required namespace exists
echo -e "${BLUE}Checking for DGLA namespace...${NC}"
if ! kubectl get namespace dgla &> /dev/null; then
    echo -e "${RED}Error: DGLA namespace does not exist. Please create it with 'kubectl create namespace dgla' and deploy the required resources.${NC}"
    exit 1
fi

# Check if DGLA pods are running
echo -e "${BLUE}Checking DGLA deployment status...${NC}"
if ! kubectl get pods -n dgla | grep -q "Running"; then
    echo -e "${RED}Error: DGLA pods are not running. Please check your deployment.${NC}"
    exit 1
fi

echo -e "\n${GREEN}✓ DGLA infrastructure is running properly${NC}"
echo -e "${BLUE}Starting demo execution sequence...${NC}"

# Run Standard Demos
echo -e "\n${YELLOW}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║                  RUNNING STANDARD DEMOS                ║${NC}"
echo -e "${YELLOW}╚══════════════════════════════════════════════════════╝${NC}"

run_demo "demos/01_secure_document_manager.py" "Secure Document Manager"
run_demo "demos/02_api_security_gateway.py" "API Security Gateway"
run_demo "demos/03_healthcare_compliance_system.py" "Healthcare Compliance System"
run_demo "demos/04_financial_transaction_monitor.py" "Financial Transaction Monitor"
run_demo "demos/05_iot_security_monitor.py" "IoT Security Monitor"
run_demo "demos/06_security_monitoring_dashboard.py" "Security Monitoring Dashboard"
run_demo "demos/07_supply_chain_verification.py" "Supply Chain Verification"
run_demo "demos/08_secure_voting_system.py" "Secure Voting System"
run_demo "demos/09_personal_data_portal.py" "Personal Data Portal"
run_demo "demos/10_regulatory_compliance_automation.py" "Regulatory Compliance Automation"

# Run Advanced Demos
echo -e "\n${YELLOW}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${YELLOW}║                  RUNNING ADVANCED DEMOS                ║${NC}"
echo -e "${YELLOW}╚══════════════════════════════════════════════════════╝${NC}"

run_demo "demos/advanced/11_quantum_resistant_zk_authentication.py" "Quantum-Resistant Authentication"
run_demo "demos/advanced/12_ai_resistant_fraud_detection.py" "AI-Resistant Fraud Detection"
run_demo "demos/advanced/13_blockchain_level_traceability.py" "Blockchain-Level Traceability"
run_demo "demos/advanced/14_cryptographic_access_control.py" "Cryptographic Access Control"
run_demo "demos/advanced/15_ethical_ai_governance.py" "Ethical AI Governance"

echo -e "\n${GREEN}✓ All demos completed successfully!${NC}"
echo -e "${BLUE}Thank you for exploring the DGLA infrastructure.${NC}"
echo -e "${BLUE}For more information, refer to the documentation in the docs/ directory.${NC}"
