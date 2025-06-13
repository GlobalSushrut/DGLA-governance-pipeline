# Kubernetes Deployment Guide for DGLA Infrastructure

This guide provides detailed instructions for deploying the DGLA infrastructure on Kubernetes, with specific configurations for local development, testing, and production environments.

## Prerequisites

- Kubernetes cluster (v1.20+)
- kubectl command-line tool configured for your cluster
- Docker for building container images
- Helm (optional, for simplified deployment)

## Deployment Architecture

The DGLA infrastructure consists of the following components:

1. **DGLA API Server**: Stateless service handling all API requests
2. **Redis Backend**: Stateful service for data storage
3. **Persistent Volumes**: For durable storage of logs and data
4. **ConfigMaps and Secrets**: For configuration and credential management

## Quick Deployment with Provided Scripts

For the fastest deployment experience, use the provided deployment script:

```bash
./deploy_dgla.sh [environment]
```

Where `[environment]` can be:
- `local` (for Minikube)
- `dev` (for development cluster)
- `prod` (for production cluster)

## Step-by-Step Deployment

If you prefer to deploy manually or need to customize the deployment, follow these steps:

### 1. Create Namespace

```bash
kubectl create namespace dgla
```

### 2. Create Kubernetes Secrets

```bash
# Create Redis password secret
kubectl create secret generic redis-password \
  --from-literal=password="your-secure-password" \
  --namespace dgla

# Create JWT secret
kubectl create secret generic jwt-secret \
  --from-literal=secret="your-jwt-signing-key" \
  --namespace dgla

# Create API keys (if needed)
kubectl create secret generic api-keys \
  --from-literal=admin-key="your-admin-api-key" \
  --namespace dgla
```

### 3. Create ConfigMaps

```bash
kubectl apply -f dgla-configmap.yaml -n dgla
```

Example ConfigMap (dgla-configmap.yaml):
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: dgla-config
  namespace: dgla
data:
  API_PORT: "8081"
  LOG_LEVEL: "INFO"
  MAX_REQUESTS_PER_MINUTE: "1000"
  REDIS_HOST: "dgla-redis"
  REDIS_PORT: "6379"
  USE_REDIS_PASSWORD: "true"
```

### 4. Create Persistent Volume Claims

```bash
kubectl apply -f dgla-pvc.yaml -n dgla
```

Example PVC (dgla-pvc.yaml):
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: redis-data
  namespace: dgla
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: dgla-logs
  namespace: dgla
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```

### 5. Deploy Redis

```bash
kubectl apply -f redis-deployment.yaml -n dgla
```

Example Redis Deployment (redis-deployment.yaml):
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dgla-redis
  namespace: dgla
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dgla-redis
  template:
    metadata:
      labels:
        app: dgla-redis
    spec:
      containers:
      - name: redis
        image: redis:6.2-alpine
        ports:
        - containerPort: 6379
        args: ["--requirepass", "$(REDIS_PASSWORD)"]
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-password
              key: password
        volumeMounts:
        - name: redis-data
          mountPath: /data
      volumes:
      - name: redis-data
        persistentVolumeClaim:
          claimName: redis-data
---
apiVersion: v1
kind: Service
metadata:
  name: dgla-redis
  namespace: dgla
spec:
  selector:
    app: dgla-redis
  ports:
  - port: 6379
    targetPort: 6379
```

### 6. Build and Deploy DGLA API Server

#### 6.1. Build the Docker Image

```bash
# For Minikube
eval $(minikube docker-env)

# Build image
docker build -t dgla-api:latest -f Dockerfile.api .
```

#### 6.2. Deploy the API Server

```bash
kubectl apply -f dgla-api-deployment.yaml -n dgla
```

Example API Server Deployment (dgla-api-deployment.yaml):
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dgla-api
  namespace: dgla
spec:
  replicas: 2
  selector:
    matchLabels:
      app: dgla-api
  template:
    metadata:
      labels:
        app: dgla-api
    spec:
      containers:
      - name: api
        image: dgla-api:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8081
        env:
        - name: REDIS_HOST
          valueFrom:
            configMapKeyRef:
              name: dgla-config
              key: REDIS_HOST
        - name: REDIS_PORT
          valueFrom:
            configMapKeyRef:
              name: dgla-config
              key: REDIS_PORT
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: redis-password
              key: password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: jwt-secret
              key: secret
        - name: API_PORT
          valueFrom:
            configMapKeyRef:
              name: dgla-config
              key: API_PORT
        volumeMounts:
        - name: dgla-logs
          mountPath: /app/logs
      volumes:
      - name: dgla-logs
        persistentVolumeClaim:
          claimName: dgla-logs
---
apiVersion: v1
kind: Service
metadata:
  name: dgla-api
  namespace: dgla
spec:
  selector:
    app: dgla-api
  ports:
  - port: 8081
    targetPort: 8081
    nodePort: 30081
  type: NodePort
```

## Production-Grade Deployment Enhancements

For production environments, consider these additional configurations:

### Horizontal Pod Autoscaling

```yaml
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: dgla-api-hpa
  namespace: dgla
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: dgla-api
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: dgla-api-policy
  namespace: dgla
spec:
  podSelector:
    matchLabels:
      app: dgla-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          access: dgla-api
    ports:
    - protocol: TCP
      port: 8081
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: dgla-redis
    ports:
    - protocol: TCP
      port: 6379
```

### Resource Limits

Update your deployment to include resource limits:

```yaml
resources:
  limits:
    cpu: "1"
    memory: "1Gi"
  requests:
    cpu: "500m"
    memory: "512Mi"
```

### TLS Termination

For secure communications, deploy an Ingress controller with TLS:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: dgla-api-ingress
  namespace: dgla
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - dgla-api.example.com
    secretName: dgla-api-tls
  rules:
  - host: dgla-api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: dgla-api
            port:
              number: 8081
```

## Monitoring and Observability

### Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: dgla-api-monitor
  namespace: dgla
spec:
  selector:
    matchLabels:
      app: dgla-api
  endpoints:
  - port: 8081
    path: /metrics
    interval: 15s
```

### Pod Disruption Budget

```yaml
apiVersion: policy/v1beta1
kind: PodDisruptionBudget
metadata:
  name: dgla-api-pdb
  namespace: dgla
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: dgla-api
```

## Verification and Testing

After deployment, verify the installation:

```bash
# Check pods are running
kubectl get pods -n dgla

# Check services are exposed
kubectl get services -n dgla

# Get API URL for external access
export NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')
export API_PORT=$(kubectl get service dgla-api -n dgla -o jsonpath='{.spec.ports[0].nodePort}')
echo "API URL: http://$NODE_IP:$API_PORT"

# Test the API
curl http://$NODE_IP:$API_PORT/health
```

## Troubleshooting

### Common Issues and Solutions

1. **Pods stuck in Pending state**
   - Check PVC binding: `kubectl describe pvc -n dgla`
   - Check node resources: `kubectl describe nodes`

2. **Connection refused errors**
   - Check service endpoints: `kubectl get endpoints -n dgla`
   - Check pod logs: `kubectl logs -n dgla deployment/dgla-api`

3. **Authentication failures**
   - Verify secrets are correctly mounted: `kubectl describe pods -n dgla`
   - Check for error logs: `kubectl logs -n dgla deployment/dgla-api | grep ERROR`

4. **Performance issues**
   - Check resource usage: `kubectl top pods -n dgla`
   - Consider increasing resource limits

## Backup and Disaster Recovery

1. **Redis Data Backup**
   ```bash
   # Create a backup job
   kubectl apply -f redis-backup-job.yaml
   ```

2. **Backup Verification**
   ```bash
   # List backup jobs
   kubectl get jobs -n dgla
   ```

3. **Restore Procedure**
   ```bash
   # Apply the restore job
   kubectl apply -f redis-restore-job.yaml
   ```

## Conclusion

This deployment guide provides a comprehensive approach to deploying the DGLA infrastructure on Kubernetes. By following these instructions, you can create a secure, scalable, and resilient environment for running the DGLA system that delivers fundamentally secure cryptographic verification, immutable audit logs, and compliance reporting capabilities.
