# DGLA Demo Infrastructure Setup and Execution Guide

This guide provides comprehensive instructions for setting up the DGLA (Data Governance and Logging Architecture) infrastructure and running all demonstration applications.

## Prerequisites

- Kubernetes (v1.20+)
- Minikube (v1.25+)
- Docker (v20.10+)
- Python 3.8+
- kubectl command-line tool
- At least 4GB RAM and 2 CPU cores available for Minikube

## Quick Start

For those who want to quickly run all demos with a single command, we provide a convenient bash script:

```bash
./run_all_demos.sh
```

This script will:
1. Verify your Minikube and DGLA infrastructure are running correctly
2. Execute all 15 demo applications in sequence
3. Provide formatted output for easy reading of results

## Step-by-Step Setup

If you need to set up the infrastructure from scratch or troubleshoot issues, follow these steps:

### 1. Start Minikube

```bash
minikube start --memory=4096 --cpus=2
```

### 2. Create DGLA Namespace

```bash
kubectl create namespace dgla
```

### 3. Deploy Redis Backend

```bash
kubectl apply -f redis.yaml -n dgla
```

### 4. Build and Deploy DGLA API Server

```bash
# Set Docker to use Minikube's Docker daemon
eval $(minikube docker-env)

# Build the API server image
docker build -t dgla-api:latest -f Dockerfile.api .

# Deploy the API server
kubectl apply -f dgla-deployment.yaml -n dgla

# Create the PVC for logs
kubectl apply -f pvc.yaml -n dgla
```

### 5. Verify Deployment

```bash
# Check that pods are running
kubectl get pods -n dgla

# Check services
kubectl get services -n dgla
```

Expected output:
```
NAME         READY   STATUS    RESTARTS   AGE
dgla-api     1/1     Running   0          2m
dgla-redis   1/1     Running   0          3m
```

### 6. Get API URL

```bash
echo "API URL: http://$(minikube ip):30081"
```

Note this URL as you'll need it to run the demos individually.

## Running Individual Demos

Each demo can be run individually by navigating to the SDK directory and running the Python script with the appropriate API URL:

```bash
cd sdk
python3 demos/01_secure_document_manager.py --api-url="http://$(minikube ip):30081"
```

Replace the demo filename with any of the 15 available demos:

### Standard Demos
1. `demos/01_secure_document_manager.py`
2. `demos/02_api_security_gateway.py`
3. `demos/03_healthcare_compliance_system.py`
4. `demos/04_financial_transaction_monitor.py`
5. `demos/05_iot_security_monitor.py`
6. `demos/06_security_monitoring_dashboard.py`
7. `demos/07_supply_chain_verification.py`
8. `demos/08_secure_voting_system.py`
9. `demos/09_personal_data_portal.py`
10. `demos/10_regulatory_compliance_automation.py`

### Advanced Demos
11. `demos/advanced/11_quantum_resistant_zk_authentication.py`
12. `demos/advanced/12_ai_resistant_fraud_detection.py`
13. `demos/advanced/13_blockchain_level_traceability.py`
14. `demos/advanced/14_cryptographic_access_control.py`
15. `demos/advanced/15_ethical_ai_governance.py`

## Troubleshooting

### API Server Not Responding

If demos fail with connection errors:

1. Check that the pods are running:
   ```bash
   kubectl get pods -n dgla
   ```

2. Check pod logs:
   ```bash
   kubectl logs -n dgla deployment/dgla-api
   ```

3. Ensure services are properly exposed:
   ```bash
   kubectl get services -n dgla
   ```

4. Verify Minikube IP is correct:
   ```bash
   minikube ip
   ```

### Redis Connection Issues

If the API server can't connect to Redis:

1. Check Redis pod status:
   ```bash
   kubectl get pods -n dgla | grep redis
   ```

2. Check Redis logs:
   ```bash
   kubectl logs -n dgla deployment/dgla-redis
   ```

3. Verify the Redis password is correctly set in both Redis deployment and API server configuration

### Kubernetes Resources Issues

If pods are pending or crashing:

1. Check for resource constraints:
   ```bash
   kubectl describe pods -n dgla
   ```

2. Increase Minikube resources:
   ```bash
   minikube stop
   minikube start --memory=6144 --cpus=4
   ```

## Clean Up

To tear down the DGLA infrastructure:

```bash
kubectl delete namespace dgla
```

This will remove all deployed resources in the dgla namespace.

## Next Steps

After successfully running the demos, explore the comprehensive documentation in the `docs/` directory to understand the architecture and security principles behind DGLA.
