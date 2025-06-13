# DGLA Command Line Interface

The DGLA CLI is a comprehensive tool for deploying, managing, and configuring the complete DGLA infrastructure stack from zero to hero. This tool handles everything from setup to testing without requiring direct interaction with configuration files or deployment manifests.

## Installation

```bash
# Quick installation
./install.sh
```

This will:
- Install required Python dependencies
- Configure the DGLA CLI globally
- Create necessary configuration directories

## Key Features

- **Complete infrastructure deployment** - Deploy all DGLA components with a single command
- **Custom SLA management** - Create and deploy customer-specific SLA agreements
- **Vendor integration** - Add third-party vendor containers to the DGLA ecosystem
- **Automated testing** - Test your deployment for functionality and compliance
- **Status monitoring** - Check component health and configuration

## Quick Start Guide

### 1. Initialize DGLA Environment

```bash
dgla init --namespace dgla --environment production
```

### 2. Deploy Core Infrastructure

```bash
# Deploy all components
dgla deploy

# Deploy specific components
dgla deploy --components mongodb,cdn,monitoring
```

### 3. Create SLA Agreements

```bash
# Create a basic SLA
dgla create-sla customer-a --customer "Customer A Inc" --tier gold

# Create SLA with custom metrics and deploy it
dgla create-sla financial-sla --customer "Financial Corp" --tier custom \
  --metrics "uptime=99.99,latency=50ms,verification=true" --apply
```

### 4. Add Vendor Containers

```bash
# Add vendor and deploy
dgla add-vendor analytics --image vendor/analytics-engine --deploy

# Add vendor with custom configuration
dgla add-vendor compliance --image vendor/compliance-service:1.2 \
  --description "Regulatory compliance service" --replicas 2 --deploy
```

### 5. Check Status

```bash
# Check all components
dgla status

# Check specific component
dgla status --component api
```

### 6. Test Deployment

```bash
# Test all components
dgla test

# Test specific vendor integration
dgla test --vendor analytics
```

## Command Reference

| Command | Description |
|---------|-------------|
| `dgla init` | Initialize DGLA environment and configuration |
| `dgla deploy` | Deploy infrastructure components |
| `dgla create-sla` | Create and deploy SLA definitions |
| `dgla add-vendor` | Add and deploy vendor containers |
| `dgla status` | Check component status |
| `dgla test` | Run tests against deployment |

## Security & Compliance

The DGLA CLI enforces:
- Cryptographic verification of data integrity
- Proper RBAC configuration for components
- Data sovereignty rules
- Ethical agreement adherence

## Customization

Custom components, SLAs, and vendor integrations can all be managed through the CLI without requiring direct editing of Kubernetes YAML files.
