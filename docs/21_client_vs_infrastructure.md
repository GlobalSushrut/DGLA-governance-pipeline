# DGLA: Client SDK vs. Server Infrastructure

## Architecture Separation

```
┌──────────────────────┐                  ┌──────────────────────┐
│                      │                  │                      │
│    CLIENT DOMAIN     │                  │  SERVER DOMAIN       │
│                      │                  │                      │
│  ┌────────────────┐  │                  │  ┌────────────────┐  │
│  │                │  │                  │  │                │  │
│  │   DGLA SDK     │  │      HTTPS       │  │  DGLA API      │  │
│  │                │──┼─────────────────►│  │  Servers       │  │
│  └────────────────┘  │                  │  │                │  │
│          │           │                  │  └────────┬───────┘  │
│          ▼           │                  │           │          │
│  ┌────────────────┐  │                  │           ▼          │
│  │   Client       │  │                  │  ┌────────────────┐  │
│  │   Application  │  │                  │  │                │  │
│  │                │  │                  │  │  Redis Backend │  │
│  └────────────────┘  │                  │  │                │  │
│                      │                  │  └────────────────┘  │
└──────────────────────┘                  └──────────────────────┘
   Customer Environment                     DGLA Infrastructure
```

## Client SDK

### Independent Deployment

The client SDK is completely separate from the DGLA infrastructure and can be deployed independently:

```
pip install dgla-sdk
```

### Minimal Dependencies

```
dgla-sdk/
├── requirements.txt
│   ├── requests>=2.25.0
│   ├── cryptography>=36.0.0
│   ├── pyjwt>=2.3.0
│   └── redis>=4.0.0 (optional - local verification only)
├── dgla_sdk/
│   ├── __init__.py
│   ├── client.py       # Core API client
│   ├── verify.py       # Verification utilities
│   ├── chainlog.py     # Audit logging functions
│   └── export.py       # Compliance reporting tools
└── examples/
    └── minimal_client.py
```

### Client Configuration

```python
# Basic configuration - only needs API endpoint
from dgla_sdk import DGLAClient

client = DGLAClient(
    api_url="https://dgla-api.example.com",
    api_key="client_api_key_here"
)

# Advanced config with local features
client = DGLAClient(
    api_url="https://dgla-api.example.com",
    api_key="client_api_key_here",
    options={
        "batch_size": 100,            # Batch operations for efficiency
        "local_verification": True,    # Enable client-side verification
        "retry_strategy": "exponential",
        "cache_size": 1000            # Local cache entries
    }
)
```

### Client-Only Features

```python
# Local operations that don't depend on server
# 1. Generate cryptographic proofs
proof = client.verify.generate_proof(data)

# 2. Local verification
result = client.verify.local_verify(data, proof)

# 3. Offline logging (synchronized later)
client.chainlog.queue_append(
    entity_id="resource_123",
    entity_type="document",
    action="viewed",
    metadata={"user": "user_456"}
)

# 4. Batch synchronization
client.sync()
```

## Server Infrastructure

### Independent Deployment

The DGLA server infrastructure is deployed separately from clients, typically in Kubernetes:

```bash
# Server deployment script
kubectl apply -f dgla-infrastructure/
```

### Infrastructure Components

```
dgla-infrastructure/
├── kubernetes/
│   ├── api-deployment.yaml    # API server pods
│   ├── api-service.yaml       # Service for API
│   ├── redis-statefulset.yaml # Redis database
│   ├── redis-service.yaml     # Service for Redis
│   ├── ingress.yaml           # Ingress controller
│   ├── secrets.yaml           # Encrypted secrets
│   └── configmap.yaml         # Configuration
├── docker/
│   ├── api/Dockerfile         # API server image
│   └── init/Dockerfile        # Init container image
└── terraform/
    └── main.tf                # Infrastructure as code
```

### Server-Side Security Hardening

```yaml
# kubernetes/api-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dgla-api
spec:
  replicas: 3
  template:
    spec:
      securityContext:
        runAsUser: 1000
        fsGroup: 2000
      containers:
      - name: dgla-api
        image: dgla/api-server:1.0.0
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
        resources:
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
        volumeMounts:
        - name: tmp
          mountPath: /tmp
        - name: certs
          mountPath: /etc/certs
          readOnly: true
      volumes:
      - name: tmp
        emptyDir: {}
      - name: certs
        secret:
          secretName: dgla-tls-certs
```

## Independent Development Cycles

### Client SDK Release Cycle

```
1. GitHub Release
   │
   ▼
2. PyPI Publication
   │
   ▼
3. Client Adoption
   (No infrastructure changes needed)
```

### Infrastructure Release Cycle

```
1. Infrastructure Update
   │
   ▼
2. Kubernetes Deployment
   │
   ▼
3. API Version Management
   (Backwards compatible with older clients)
```

## Concrete Usage Examples

### Client in Production App

```python
# In a customer's application
from dgla_sdk import DGLAClient
import logging

# Application code
class SecureFinancialApp:
    def __init__(self):
        # Initialize DGLA client
        self.dgla = DGLAClient(
            api_url=os.environ.get("DGLA_API_URL"),
            api_key=os.environ.get("DGLA_API_KEY")
        )
        
    def process_transaction(self, user_id, amount, description):
        try:
            # Business logic
            transaction_id = self.core_banking.create_transaction(
                user_id=user_id,
                amount=amount,
                description=description
            )
            
            # Log to DGLA (audit trail)
            self.dgla.chainlog.append(
                entity_id=user_id,
                entity_type="user",
                action="transaction",
                metadata={
                    "transaction_id": transaction_id,
                    "amount": amount,
                    "description": description
                }
            )
            
            return transaction_id
        except Exception as e:
            logging.error(f"Transaction failed: {e}")
            raise
```

### Infrastructure Deployment

```bash
# In DGLA provider's infrastructure
# Launch server infrastructure in production

# 1. Create namespace
kubectl create namespace dgla-prod

# 2. Deploy secrets
kubectl create secret generic dgla-keys \
  --from-file=jwt-key=./keys/jwt.key \
  --from-literal=redis-password=$(openssl rand -hex 32) \
  -n dgla-prod

# 3. Deploy Redis backend
kubectl apply -f kubernetes/redis-statefulset.yaml -n dgla-prod
kubectl apply -f kubernetes/redis-service.yaml -n dgla-prod

# 4. Deploy API servers
kubectl apply -f kubernetes/api-deployment.yaml -n dgla-prod
kubectl apply -f kubernetes/api-service.yaml -n dgla-prod

# 5. Configure ingress
kubectl apply -f kubernetes/ingress.yaml -n dgla-prod

# 6. Monitor deployment
kubectl get pods -n dgla-prod
```

## Security Boundaries

### Client-Side Security

- API keys stored in client environment
- Local verification capabilities
- Client-side proof generation
- Request signing

### Server-Side Security

- TLS termination
- JWT validation
- Rate limiting
- IP allowlisting
- Tenant isolation
- Encryption at rest
- Network policies

## Operational Independence

| Aspect | Client SDK | Server Infrastructure |
|--------|-----------|------------------------|
| **Updates** | Client teams update independently | Server team manages deployments |
| **Scaling** | No scaling concerns | Horizontal pod autoscaling |
| **Monitoring** | Client-side metrics | Full observability stack |
| **Dependencies** | Minimal, python packages | Complete Kubernetes stack |
| **Security** | Request signing, local verification | Network isolation, encryption |

## Connecting Client to Infrastructure

### API Key Provisioning

```bash
# Generate API key for client
CLIENT_ID=$(openssl rand -hex 8)
API_KEY=$(openssl rand -hex 32)

# Register in DGLA infrastructure
kubectl exec -it dgla-api-0 -n dgla-prod -- \
  dgla-admin create-client \
  --client-id "$CLIENT_ID" \
  --api-key "$API_KEY" \
  --name "Customer Financial App" \
  --rate-limit 1000

# Share with client for SDK configuration
echo "Client ID: $CLIENT_ID"
echo "API Key: $API_KEY"
```

### Network Configuration

```
┌─────────────────────────┐      ┌─────────────────────────┐
│                         │      │                         │
│  Customer Environment   │      │     DGLA Provider       │
│                         │      │                         │
│  ┌─────────────────┐    │      │    ┌─────────────────┐  │
│  │ Client App with │    │      │    │                 │  │
│  │ DGLA SDK        │────┼──────┼───►│ Load Balancer   │  │
│  └─────────────────┘    │      │    └────────┬────────┘  │
│                         │      │             │           │
│  - Outbound HTTPS       │      │             ▼           │
│    to DGLA API          │      │    ┌─────────────────┐  │
│                         │      │    │ API Servers     │  │
│  - No inbound           │      │    │                 │  │
│    connectivity needed  │      │    └─────────────────┘  │
│                         │      │                         │
└─────────────────────────┘      └─────────────────────────┘
```

## Client-Infrastructure Version Compatibility

```
┌─────────────────┐
│ SDK Version 2.1 │───┐
└─────────────────┘   │
                      │
┌─────────────────┐   │     ┌─────────────────┐
│ SDK Version 2.0 │───┼────►│ API Version 2.0 │
└─────────────────┘   │     └─────────────────┘
                      │
┌─────────────────┐   │
│ SDK Version 1.9 │───┘
└─────────────────┘
```

## Testing Independence

### Client SDK Testing

```python
# Client SDK unit tests (independent of infrastructure)
def test_local_verification():
    # Create a client with mock server
    client = DGLAClient(
        api_url="mock://dgla-api",
        api_key="test_key"
    )
    
    # Test local verification
    data = {"test": "data"}
    proof = client.verify.generate_proof(data)
    result = client.verify.local_verify(data, proof)
    
    assert result["verified"] == True
```

### Infrastructure Testing

```bash
# Infrastructure integration tests
# Deploy test instance
kubectl apply -f kubernetes/test/

# Run integration test suite
pytest integration_tests/ \
  --server-url https://test-api.dgla-internal \
  --admin-key "${TEST_ADMIN_KEY}"
```
