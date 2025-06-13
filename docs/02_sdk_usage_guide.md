# DGLA SDK Usage Guide

## Introduction

The DGLA SDK provides a comprehensive set of tools for integrating with the Data Governance and Logging Architecture. This SDK enables applications to leverage cryptographic verification, immutable audit trails, and compliance reporting capabilities with minimal code changes.

## Installation

The DGLA SDK can be installed via pip:

```bash
pip install dgla_sdk
```

For development environments, you can install directly from the project directory:

```bash
cd data-governance-pipeline/sdk
pip install -e .
```

## Core Modules

### Client Module

The foundation for all DGLA interactions, handling authentication and communication:

```python
from dgla_sdk.client import DGLAClient

# Initialize the client
client = DGLAClient(
    base_url="http://your-dgla-api-server:8081",
    api_key="your-api-key"  # Optional
)

# Authenticate
client.authenticate("username", "password")

# Make API calls
response = client.get_status()
```

### Verify Module

Provides cryptographic proof generation and validation:

```python
# Access through the client
verify = client.verify

# Create a cryptographic proof of any data object
proof = verify.create_proof({
    "document_id": "12345",
    "hash": "a1b2c3d4e5f6",
    "timestamp": 1623412345
})

# Validate a proof
result = verify.validate_proof(proof["id"])

# Create a cryptographic hash
hash_value = verify.create_hash("content to hash")
```

### ChainLog Module

Manages immutable audit trail creation and verification:

```python
# Access through the client
chainlog = client.chainlog

# Append to the immutable audit log
log_entry = chainlog.append_log(
    entity_id="user123",
    entity_type="user",
    action="login",
    metadata={"ip": "192.168.1.1", "device": "mobile"}
)

# Verify the integrity of the log chain
verification = chainlog.verify_chain("user123", "user")

# Get logs for an entity
logs = chainlog.get_logs("user123", "user")
```

### Export Module

Enables compliance and regulatory reporting:

```python
# Access through the client
export = client.export

# Generate a compliance report
report = export.generate_compliance_report(
    report_type="REPORT_GDPR",
    entity_id="user123",
    format="pdf"
)

# Export logs for audit
logs = export.export_logs(
    entity_id="system",
    entity_type="api_server",
    start_time=1623412345,
    end_time=1623498745
)
```

## Common Usage Patterns

### Secure Document Management

```python
def store_document(content, metadata):
    # Create a hash of the document
    doc_hash = client.verify.create_hash(content)
    
    # Store document metadata with its hash
    doc_id = str(uuid.uuid4())
    doc_metadata = {
        "id": doc_id,
        "name": metadata["name"],
        "hash": doc_hash,
        "timestamp": time.time()
    }
    
    # Create a cryptographic proof of document existence
    proof = client.verify.create_proof(doc_metadata)
    
    # Log the document creation event
    client.chainlog.append_log(
        entity_id=doc_id,
        entity_type="document",
        action="create",
        metadata={"proof_id": proof["id"]}
    )
    
    return doc_id, proof["id"]
```

### Secure Access Control

```python
def verify_access(user_id, resource_id, permission):
    # Record the access request in the immutable log
    log = client.chainlog.append_log(
        entity_id=user_id,
        entity_type="user",
        action="access_request",
        metadata={
            "resource_id": resource_id,
            "permission": permission,
            "timestamp": time.time()
        }
    )
    
    # Make the access decision (actual logic would be here)
    granted = check_permission(user_id, resource_id, permission)
    
    # Create a cryptographic proof of the access decision
    proof = client.verify.create_proof({
        "user_id": user_id,
        "resource_id": resource_id,
        "permission": permission,
        "decision": granted,
        "timestamp": time.time(),
        "log_id": log["id"]
    })
    
    # Record the access decision
    client.chainlog.append_log(
        entity_id=user_id,
        entity_type="user",
        action="access_decision",
        metadata={
            "resource_id": resource_id,
            "permission": permission,
            "granted": granted,
            "proof_id": proof["id"]
        }
    )
    
    return granted, proof["id"]
```

### Compliance Reporting

```python
def generate_gdpr_report(user_id):
    # Generate a GDPR compliance report
    report = client.export.generate_compliance_report(
        report_type="REPORT_GDPR",
        entity_id=user_id,
        format="pdf"
    )
    
    # Create a proof of the report generation
    proof = client.verify.create_proof({
        "report_id": report["report_id"],
        "user_id": user_id,
        "report_type": "GDPR",
        "timestamp": time.time()
    })
    
    # Log the report generation
    client.chainlog.append_log(
        entity_id=user_id,
        entity_type="user",
        action="generate_report",
        metadata={
            "report_id": report["report_id"],
            "report_type": "GDPR",
            "proof_id": proof["id"]
        }
    )
    
    return report["report_id"], proof["id"]
```

## Advanced Features

### Zero-Knowledge Authentication

```python
def register_zk_user(username, password):
    # Generate a lattice-based commitment
    salt = os.urandom(16).hex()
    lattice_params = {"n": 1024, "q": 12289}
    
    # Create a zero-knowledge proof of password
    zk_commitment = client.verify.create_hash(
        f"{password}{salt}", 
        algorithm="lattice",
        params=lattice_params
    )
    
    # Register the user with zero-knowledge credentials
    user_id = str(uuid.uuid4())
    user_data = {
        "id": user_id,
        "username": username,
        "zk_commitment": zk_commitment,
        "salt": salt,
        "lattice_params": lattice_params
    }
    
    # Create proof of registration
    proof = client.verify.create_proof(user_data)
    
    # Log the registration
    client.chainlog.append_log(
        entity_id=user_id,
        entity_type="user",
        action="register",
        metadata={"proof_id": proof["id"]}
    )
    
    return user_id, proof["id"]
```

### AI-Resistant Fraud Detection

```python
def verify_transaction_sequence(transactions):
    # Verify the temporal causality of transactions
    results = []
    anomalies = []
    
    for i in range(1, len(transactions)):
        current = transactions[i]
        previous = transactions[i-1]
        
        # Create a proof linking the transactions
        proof = client.verify.create_proof({
            "current_id": current["id"],
            "previous_id": previous["id"],
            "time_delta": current["timestamp"] - previous["timestamp"],
            "verification_time": time.time()
        })
        
        # Check for anomalies (e.g., impossible time deltas)
        if current["timestamp"] - previous["timestamp"] < 0:
            anomalies.append({
                "type": "time_reversal",
                "current_id": current["id"],
                "previous_id": previous["id"],
                "proof_id": proof["id"]
            })
        
        results.append({
            "current_id": current["id"],
            "verified": len(anomalies) == 0,
            "proof_id": proof["id"]
        })
    
    return results, anomalies
```

## Error Handling

The SDK provides comprehensive error handling:

```python
from dgla_sdk.exceptions import DGLAAuthError, DGLAApiError

try:
    client.authenticate("username", "wrong_password")
except DGLAAuthError as e:
    print(f"Authentication failed: {e}")

try:
    client.verify.validate_proof("non_existent_proof_id")
except DGLAApiError as e:
    print(f"API error: {e}")
```

## Conclusion

The DGLA SDK provides a powerful yet simple interface to implement cryptographic security, immutable audit trails, and compliance reporting in any application. By following the patterns in this guide, developers can create applications that are fundamentally secure by design rather than by policy.
