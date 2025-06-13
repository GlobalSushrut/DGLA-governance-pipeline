# DGLA Integration Guide

## Overview

This guide provides instructions for integrating DGLA with common enterprise systems. DGLA offers cryptographically secure interfaces for seamless integration with your existing infrastructure.

## Integration Methods

### REST API Integration

Use direct REST API calls for simple integrations:

```python
import requests

def api_request(endpoint, method="GET", data=None, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    
    url = f"http://{DGLA_HOST}:{DGLA_PORT}/{endpoint}"
    
    if method == "GET":
        response = requests.get(url, headers=headers)
    elif method == "POST":
        response = requests.post(url, json=data, headers=headers)
    
    response.raise_for_status()
    return response.json()

# Authentication
auth_response = api_request("auth/login", 
                           method="POST", 
                           data={"username": "user", "password": "pass"})
token = auth_response["token"]

# Create a proof
proof = api_request("verify/create-proof", 
                   method="POST", 
                   data={"document": "content"}, 
                   token=token)
```

### SDK Integration

Use the DGLA SDK for full-featured integration:

```python
from dgla_sdk import DGLAClient

# Initialize client
client = DGLAClient(
    api_url="http://dgla_host:port", 
    username="user", 
    password="pass"
)

# Use SDK features
proof = client.verify.create_proof({"document": "content"})
client.chainlog.append_log("doc123", "document", "update", {"field": "value"})
```

### Webhook Integration

Configure your application to receive DGLA events:

```python
# Flask webhook receiver example
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route("/dgla-webhook", methods=["POST"])
def dgla_webhook():
    data = request.json
    
    # Verify webhook signature
    signature = request.headers.get("X-DGLA-Signature")
    if not verify_signature(data, signature, webhook_secret):
        return jsonify({"error": "Invalid signature"}), 401
    
    # Process webhook data
    event_type = data["event"]
    if event_type == "log_append":
        process_log_event(data["log"])
    elif event_type == "verification":
        process_verification(data["verification"])
    
    return jsonify({"status": "success"})
```

## Common Integration Scenarios

### SIEM Integration

```python
# Export DGLA logs to SIEM system
def export_to_siem(start_time, end_time):
    # Get logs from DGLA
    logs = client.chainlog.get_logs(
        start_time=start_time,
        end_time=end_time
    )
    
    # Transform to SIEM format
    siem_events = []
    for log in logs:
        siem_events.append({
            "timestamp": log["timestamp"],
            "source": "DGLA",
            "event_type": log["action"],
            "entity": f"{log['entity_type']}:{log['entity_id']}",
            "actor": log["actor_id"],
            "details": log["metadata"],
            "verification": {
                "hash": log["hash"],
                "previous_hash": log["previous_hash"]
            }
        })
    
    # Send to SIEM API
    send_to_siem_api(siem_events)
```

### Identity Provider Integration

```python
# Example with OAuth 2.0
from authlib.integrations.requests_client import OAuth2Session

# Configure OAuth client
oauth = OAuth2Session(
    client_id,
    client_secret,
    scope="profile"
)

# Get authorization URL
authorization_url, state = oauth.create_authorization_url(
    'https://idp.example.com/authorize'
)

# Exchange code for token
token = oauth.fetch_token(
    'https://idp.example.com/token',
    authorization_response=redirect_url
)

# Use token with DGLA
client = DGLAClient(
    api_url="http://dgla_host:port",
    oauth_token=token["access_token"]
)
```

### CI/CD Pipeline Integration

```yaml
# Example GitHub Actions workflow
name: DGLA Integration

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  verify-deployment:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - name: Set up Python
      uses: actions/setup-python@v2
      with:
        python-version: '3.8'
    
    - name: Install DGLA SDK
      run: pip install dgla-sdk
      
    - name: Create Deployment Proof
      run: |
        python -c "
        from dgla_sdk import DGLAClient
        import os

        # Create client
        client = DGLAClient(
            api_url=os.environ['DGLA_API_URL'],
            username=os.environ['DGLA_USERNAME'],
            password=os.environ['DGLA_PASSWORD']
        )

        # Create deployment proof
        proof = client.verify.create_proof({
            'repository': '${{ github.repository }}',
            'commit': '${{ github.sha }}',
            'workflow': '${{ github.workflow }}',
            'timestamp': '${{ github.event.repository.updated_at }}'
        })

        # Output proof ID for later verification
        with open('proof_id.txt', 'w') as f:
            f.write(proof['id'])
        "
```

## Enterprise Bus Integration

```java
// Java integration with Enterprise Service Bus
import org.apache.camel.builder.RouteBuilder;

public class DGLARouteBuilder extends RouteBuilder {
    @Override
    public void configure() throws Exception {
        // Authentication route
        from("direct:dglaAuth")
            .setHeader("Content-Type", constant("application/json"))
            .setBody(constant("{\"username\":\"{{dgla.username}}\",\"password\":\"{{dgla.password}}\"}}"))
            .to("http://{{dgla.host}}:{{dgla.port}}/auth/login")
            .process(exchange -> {
                String token = exchange.getMessage().getBody(String.class);
                exchange.setProperty("dglaToken", token);
            });
            
        // Log verification route
        from("direct:verifyLogs")
            .enrich("direct:dglaAuth", (original, auth) -> {
                original.setProperty("dglaToken", auth.getProperty("dglaToken"));
                return original;
            })
            .setHeader("Authorization", simple("Bearer ${property.dglaToken}"))
            .toD("http://{{dgla.host}}:{{dgla.port}}/chainlog/verify/${header.entityId}/${header.entityType}");
    }
}
```

## Database Integration

```sql
-- PostgreSQL stored procedure for DGLA verification
CREATE OR REPLACE FUNCTION verify_document(doc_id TEXT) RETURNS BOOLEAN AS $$
DECLARE
    dgla_token TEXT;
    verification JSONB;
BEGIN
    -- Get DGLA token (simplified for example)
    SELECT http_post(
        'http://dgla_host:port/auth/login',
        '{"username":"db_service","password":"db_password"}',
        'application/json'
    ) INTO dgla_token;
    
    -- Verify document using DGLA
    SELECT http_get(
        'http://dgla_host:port/chainlog/verify/' || doc_id || '/document',
        'Authorization: Bearer ' || dgla_token::json->>'token'
    ) INTO verification;
    
    RETURN verification::json->>'verified';
END;
$$ LANGUAGE plpgsql;
```

## Container Integration

```dockerfile
# Dockerfile with DGLA sidecar
FROM python:3.8-slim

# Install app and dependencies
COPY app /app
WORKDIR /app
RUN pip install -r requirements.txt

# Install DGLA SDK
RUN pip install dgla-sdk

# DGLA initialization script
COPY dgla_init.py /app/
ENV DGLA_API_URL=http://dgla-service:8081
ENV DGLA_USERNAME=service-account
ENV DGLA_PASSWORD=service-password

# Start command with DGLA initialization
CMD ["sh", "-c", "python dgla_init.py && python app.py"]
```

## Monitoring Integration

```python
# Prometheus metrics example
from prometheus_client import Counter, Histogram, start_http_server

# Define metrics
dgla_requests = Counter('dgla_requests_total', 'Total DGLA API requests', ['endpoint'])
dgla_request_duration = Histogram('dgla_request_duration_seconds', 'DGLA request duration')
dgla_verification_results = Counter('dgla_verification_results', 'DGLA verification results', ['result'])

# Wrap client methods
def track_dgla_requests(func):
    def wrapper(*args, **kwargs):
        endpoint = func.__name__
        dgla_requests.labels(endpoint=endpoint).inc()
        
        with dgla_request_duration.time():
            result = func(*args, **kwargs)
            
        if "verified" in result:
            dgla_verification_results.labels(
                result="success" if result["verified"] else "failure"
            ).inc()
            
        return result
    return wrapper

# Start metrics server
start_http_server(8000)
```

## Security Considerations

1. **API Authentication**: Always use secure authentication methods
2. **Token Management**: Store tokens securely, rotate regularly
3. **Verification**: Verify all responses cryptographically
4. **Network Security**: Use TLS for all API communications
5. **Access Control**: Implement least privilege for service accounts

## Troubleshooting Integration

1. **Authentication Issues**
   - Verify credentials
   - Check token expiration
   - Confirm API endpoint

2. **Connection Problems**
   - Check network connectivity
   - Verify Kubernetes service discovery
   - Confirm firewalls allow traffic

3. **Verification Failures**
   - Validate data formats
   - Check for data corruption
   - Verify cryptographic parameters

## Example Integration Code

Find complete integration examples in:
- `/examples/integrations/siem`
- `/examples/integrations/idp`
- `/examples/integrations/cicd`
- `/examples/integrations/esb`
