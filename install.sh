#!/bin/bash
# DGLA Installation Script - Quick setup for DGLA CLI and dependencies

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

INSTALL_DIR="/usr/local/bin"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${GREEN}=== DGLA Installation ===${NC}"
echo "This script will install the DGLA CLI tool and its dependencies"

# Check for required tools
echo -e "\n${YELLOW}Checking prerequisites...${NC}"
for cmd in python3 pip3 kubectl; do
    if ! command -v $cmd &> /dev/null; then
        echo -e "${RED}Error: $cmd is required but not installed.${NC}"
        exit 1
    fi
done
echo -e "✓ All prerequisites found"

# Install Python dependencies
echo -e "\n${YELLOW}Installing Python dependencies...${NC}"
pip3 install pyyaml kubernetes requests cryptography

# Create symlink to CLI tool
echo -e "\n${YELLOW}Installing DGLA CLI...${NC}"
sudo ln -sf "${SCRIPT_DIR}/cli/dgla.py" "${INSTALL_DIR}/dgla"
sudo chmod +x "${INSTALL_DIR}/dgla"

# Create config directories
echo -e "\n${YELLOW}Setting up configuration...${NC}"
mkdir -p ~/.dgla/sla
mkdir -p ~/.dgla/vendors

echo -e "\n${GREEN}=== Installation Complete ===${NC}"
echo "DGLA CLI is now installed. Run 'dgla --help' to get started."
echo -e "\n${YELLOW}Quick Start:${NC}"
echo "  1. Initialize DGLA: dgla init --namespace dgla"
echo "  2. Deploy core components: dgla deploy"
echo "  3. Create an SLA: dgla create-sla my-sla --customer MyCompany"
echo "  4. Check status: dgla status"
echo -e "\nFor complete documentation, visit: https://dgla.io/docs"
