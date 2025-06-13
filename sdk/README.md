# DGLA SDK - Data Governance & Logging Architecture

Client SDK for interacting with the DGLA infrastructure components.

## Installation

```bash
pip install dgla-sdk
```

## Quick Start

```python
from dgla_sdk import DGLAClient

# Initialize client
client = DGLAClient(
    base_url="https://api.dgla.io",
    api_key="your_api_key"
)

# Check if the API is alive
status = client.alive()
print(f"API Status: {status}")

# Authenticate
auth_response = client.auth.login(
    username="yourusername",
    password="yourpassword"
)

# Create immutable log entry
log_entry = client.chainlog.append_log(
    entity_id="document_123",
    entity_type="document",
    action="view",
    metadata={"ip": "192.168.1.1", "user_agent": "Mozilla/5.0"}
)

# Create a compliance report
report = client.export.generate_compliance_report(
    report_type="gdpr",
    start_time="2025-01-01T00:00:00Z",
    end_time="2025-01-31T23:59:59Z"
)
```

## Features

- **Authentication**: Secure JWT-based authentication
- **Immutable Logging**: Blockchain-anchored audit logs
- **Verification**: Cryptographic data integrity verification
- **Compliance Reporting**: Automated compliance report generation
- **Metrics**: Real-time monitoring and alerting

## Documentation

For detailed documentation on each module and method, visit [https://docs.dgla.io](https://docs.dgla.io)

## License

MIT License
