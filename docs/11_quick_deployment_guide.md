# DGLA Quick Deployment Guide

## Prerequisites

- Kubernetes v1.19+
- kubectl configured
- Minikube for local testing
- Docker
- Python 3.8+

## One-Command Deployment

```bash
# Clone repository (if needed)
git clone https://github.com/example/dgla.git
cd dgla

# Deploy with single command
./deploy.sh
```

## Manual Deployment Steps

### 1. Start Kubernetes

```bash
minikube start --memory=4096 --cpus=2
# Use minikube docker env for building images
eval $(minikube docker-env)
```

### 2. Create Namespace

```bash
kubectl create namespace dgla
```

### 3. Create ConfigMaps & Secrets

```bash
# Create API server ConfigMap
kubectl apply -f kubernetes/configmaps/api-config.yaml

# Create Redis secret
kubectl create secret generic redis \
  --from-literal=password=your-redis-password \
  -n dgla

# Create API JWT secret
kubectl create secret generic jwt-secret \
  --from-literal=secret=your-jwt-secret \
  -n dgla
```

### 4. Create Persistent Volumes

```bash
kubectl apply -f kubernetes/volumes/redis-pvc.yaml
```

### 5. Deploy Redis

```bash
kubectl apply -f kubernetes/redis/deployment.yaml
kubectl apply -f kubernetes/redis/service.yaml
```

### 6. Deploy API Server

```bash
# Build API server image
docker build -t dgla-api-server:latest -f Dockerfile.api .

# Deploy API server
kubectl apply -f kubernetes/api-server/deployment.yaml
kubectl apply -f kubernetes/api-server/service.yaml
```

### 7. Verify Deployment

```bash
# Check pod status
kubectl get pods -n dgla

# Get NodePort address
export MINIKUBE_IP=$(minikube ip)
export API_PORT=$(kubectl get svc api-server -n dgla -o jsonpath='{.spec.ports[0].nodePort}')
export DGLA_API="http://$MINIKUBE_IP:$API_PORT"

# Verify API server
curl $DGLA_API/system/status
```

## SDK Installation

```bash
# From DGLA directory
cd sdk
pip install -e .
```

## Testing Deployment

```bash
# Initialize client
python -c "
from dgla_sdk import DGLAClient
client = DGLAClient(api_url='$DGLA_API', username='admin', password='admin')
status = client.system.get_status()
print('DGLA Status:', status)
"
```

## Running Demos

```bash
# Run all demos
./run_all_demos.sh

# Run specific demo
cd sdk
PYTHONPATH=.. python -m demos.standard.01_document_management
```

## Production Enhancements

Add these for production deployments:

1. **TLS**: Apply ingress with TLS certificates
2. **Authentication**: Integrate with your identity provider
3. **Scaling**: Implement HorizontalPodAutoscaler
4. **Monitoring**: Deploy Prometheus/Grafana stack
5. **Backup**: Configure persistent storage backups

## Cleanup

```bash
# Delete namespace to clean up all resources
kubectl delete namespace dgla

# Or stop minikube
minikube stop
```

## Next Steps

- See [Architecture Overview](01_architecture_overview.md)
- See [SDK Usage Guide](02_sdk_usage_guide.md)
- See [Kubernetes Deployment Guide](03_kubernetes_deployment_guide.md) for advanced configuration
