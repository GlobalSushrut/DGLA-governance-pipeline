# DGLA Performance Optimization

## Overview

This guide provides optimization strategies for DGLA deployments to ensure maximum performance without sacrificing security.

## API Server Optimization

### Connection Pooling

```python
# SDK configuration with connection pooling
client = DGLAClient(
    api_url="http://dgla_host:port",
    username="user",
    password="pass",
    pool_connections=20,
    pool_maxsize=20
)
```

### Batch Operations

```python
# Batch log appends (efficient)
batch_results = client.chainlog.batch_append([
    {"entity_id": "doc1", "entity_type": "document", "action": "read"},
    {"entity_id": "doc2", "entity_type": "document", "action": "update"},
    {"entity_id": "doc3", "entity_type": "document", "action": "read"}
])

# Individual appends (inefficient)
# Avoid this pattern for multiple logs
for doc in docs:
    client.chainlog.append(doc.id, "document", "read")
```

## Redis Optimization

### Memory Management

```bash
# Apply Redis memory optimizations
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: dgla
data:
  redis.conf: |
    maxmemory 2gb
    maxmemory-policy allkeys-lru
    lazyfree-lazy-eviction yes
    activedefrag yes
EOF
```

### Persistence Configuration

```bash
# Optimize Redis persistence settings
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-config
  namespace: dgla
data:
  redis.conf: |
    appendonly yes
    appendfsync everysec
    no-appendfsync-on-rewrite yes
EOF
```

## Kubernetes Optimization

### Resource Limits

```yaml
# Optimal resource settings
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
  namespace: dgla
spec:
  template:
    spec:
      containers:
      - name: api-server
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 1
            memory: 1Gi
```

### Horizontal Scaling

```yaml
# Auto-scaling configuration
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-server
  namespace: dgla
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-server
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Client-Side Optimization

### Async Operations

```python
# Async client usage
import asyncio
from dgla_sdk.async_client import AsyncDGLAClient

async def process_documents(docs):
    client = AsyncDGLAClient(
        api_url="http://dgla_host:port",
        username="user",
        password="pass"
    )
    
    # Create tasks for all documents
    tasks = [
        client.chainlog.append_async(
            doc.id, 
            "document", 
            "process"
        )
        for doc in docs
    ]
    
    # Run concurrently
    results = await asyncio.gather(*tasks)
    return results

# Run the async function
results = asyncio.run(process_documents(documents))
```

### Local Caching

```python
# Implement strategic caching
from cachetools import TTLCache

# Create cache with 1-minute TTL
token_cache = TTLCache(maxsize=100, ttl=60)

def get_cached_token(username, password):
    cache_key = f"{username}:{password}"
    
    if cache_key in token_cache:
        return token_cache[cache_key]
    
    # Get new token
    client = DGLAClient(
        api_url="http://dgla_host:port",
        username=username,
        password=password
    )
    
    # Cache the token
    token = client.auth.get_token()
    token_cache[cache_key] = token
    
    return token
```

## Verification Performance

### Hash Selection

```python
# Use optimized hash algorithm for large data
import hashlib

def efficient_hash_for_size(data, size_bytes):
    if size_bytes < 1024:  # <1KB
        # Blake2b is fast for small data
        return hashlib.blake2b(data).hexdigest()
    else:
        # SHA-256 more efficient for larger data
        return hashlib.sha256(data).hexdigest()
```

### Proof Optimization

```python
# Selective proof creation
def create_optimized_proof(document, importance):
    if importance == "high":
        # Full cryptographic proof with all metadata
        return client.verify.create_proof(
            document,
            include_metadata=True,
            signature_type="full"
        )
    else:
        # Lightweight proof for routine operations
        return client.verify.create_proof(
            document,
            include_metadata=False,
            signature_type="compact"
        )
```

## API Request Reduction

### Bulk Operations

```python
# Use bulk endpoints
bulk_verification = client.verify.bulk_validate([
    "proof-id-1",
    "proof-id-2",
    "proof-id-3"
])

# Instead of individual calls
# verification1 = client.verify.validate("proof-id-1")
# verification2 = client.verify.validate("proof-id-2")
# verification3 = client.verify.validate("proof-id-3")
```

### Webhook Notifications

```python
# Register for push notifications
client.webhooks.register(
    url="https://your-service.example.com/dgla-webhook",
    events=["log_append", "verification", "compliance"],
    secret="your-webhook-secret"
)

# Instead of polling
# logs = client.chainlog.get_new_logs(last_timestamp)
```

## Query Optimization

### Efficient Filtering

```python
# Efficient: Specific query with time bounds
logs = client.chainlog.get_logs(
    entity_id="doc123",
    entity_type="document",
    start_time=yesterday,
    end_time=now,
    limit=10
)

# Inefficient: Retrieving all logs then filtering
# all_logs = client.chainlog.get_logs()
# filtered = [log for log in all_logs if log["entity_id"] == "doc123"]
```

### Pagination

```python
# Use pagination for large result sets
all_results = []
page_token = None

while True:
    page, page_token = client.chainlog.get_logs(
        entity_type="document",
        limit=100,
        page_token=page_token
    )
    
    all_results.extend(page)
    
    if not page_token:
        break
```

## Monitoring Performance

### Tracking Metrics

```python
# Add timing to track performance
import time

def timed_operation(operation_name):
    def decorator(func):
        def wrapper(*args, **kwargs):
            start_time = time.time()
            result = func(*args, **kwargs)
            duration = time.time() - start_time
            
            # Log or export metric
            print(f"{operation_name} took {duration:.3f}s")
            
            return result
        return wrapper
    return decorator

@timed_operation("create_proof")
def create_document_proof(document):
    return client.verify.create_proof(document)
```

### Performance Dashboard

Create a Grafana dashboard with these key metrics:

- API response time (p50, p95, p99)
- Redis operation latency
- Log append throughput
- Proof verification rate
- Error rate by endpoint

## Benchmarks

| Operation | Standard Config | Optimized Config |
|-----------|-----------------|------------------|
| Authentication | 120ms | 45ms |
| Log Append | 80ms | 25ms |
| Chain Verification | 350ms | 120ms |
| Proof Creation | 150ms | 60ms |
| Proof Validation | 120ms | 40ms |
| Bulk Log Retrieval (100) | 500ms | 180ms |

## Production Checklist

- [ ] Configure Redis with appropriate memory limits
- [ ] Implement connection pooling for API clients
- [ ] Enable horizontal scaling for API servers
- [ ] Use batch operations for multiple records
- [ ] Implement efficient caching strategy
- [ ] Configure proper index TTL for expired records
- [ ] Enable compression for large data
- [ ] Deploy with CDN for static assets
- [ ] Implement proper timeout handling
