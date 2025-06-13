# DGLA Compliance Guide

## Overview

DGLA provides cryptographically verifiable compliance capabilities that surpass traditional compliance systems by providing mathematical proof rather than assertions.

## Key Compliance Features

- **Immutable Audit Trails**: Tamper-evident records of all system actions
- **Cryptographic Proof**: Mathematical verification of compliance status
- **Automated Reporting**: Pre-built templates for common regulations
- **Continuous Verification**: Real-time compliance monitoring
- **Evidence Export**: Compliance-ready evidence with verification

## Supported Regulations

| Regulation | Key DGLA Features |
|------------|------------------|
| GDPR | Data subject access, immutable processing logs, export/delete |
| HIPAA | PHI access controls, audit trails, integrity verification |
| PCI DSS | Cardholder data protection, access logging, integrity monitoring |
| SOX | Financial record verification, chain of custody, access controls |
| CCPA | Data subject access, record processing, right to delete |
| NIST 800-53 | Cryptographic verification, audit, authentication controls |

## Implementation Examples

### GDPR Compliance

```python
# Data Subject Access Request implementation
def process_dsar(user_id):
    # Get all records about this user with proof
    user_data = client.data.get_user_data(user_id)
    
    # Create access log with verification
    access_log = client.chainlog.append(
        entity_id=user_id,
        entity_type="user",
        action="dsar_access",
        metadata={"requester": "user", "timestamp": time.time()}
    )
    
    # Generate GDPR compliance report
    report = client.export.generate_compliance_report(
        report_type=client.constants.REPORT_GDPR,
        entity_id=user_id,
        start_time=user_data["created_at"],
        end_time=time.time(),
        format="pdf"
    )
    
    return {
        "user_data": user_data,
        "access_proof": access_log["id"],
        "report_url": report["download_url"],
        "verification_id": report["proof_id"]
    }
```

### HIPAA Audit Controls

```python
# HIPAA required access tracking
def access_patient_record(patient_id, user_id, reason):
    # Verify user has authorization
    authorization = client.verify.validate_access(
        user_id=user_id,
        resource_id=patient_id,
        resource_type="patient_record"
    )
    
    if not authorization["authorized"]:
        # Log unauthorized access attempt
        client.chainlog.append(
            entity_id=patient_id,
            entity_type="patient_record",
            action="unauthorized_access",
            metadata={
                "user_id": user_id,
                "reason": reason,
                "denied_reason": authorization["reason"]
            }
        )
        raise AccessDeniedException(authorization["reason"])
    
    # Log authorized access with required HIPAA fields
    access_log = client.chainlog.append(
        entity_id=patient_id,
        entity_type="patient_record",
        action="access",
        metadata={
            "user_id": user_id,
            "reason": reason,
            "timestamp": time.time(),
            "location": get_user_location(user_id)
        }
    )
    
    # Return patient data and access proof
    patient_data = get_patient_data(patient_id)
    return {
        "data": patient_data,
        "access_proof": access_log["id"]
    }
```

### PCI DSS Cardholder Data

```python
# PCI compliant payment processing
def process_payment(payment_info, amount, merchant_id):
    # Create tokenized version of card data
    tokenized_card = client.verify.create_protected_data(
        data=payment_info["card"],
        protection_level="pci-tokenized"
    )
    
    # Log payment processing with minimal PII
    processing_log = client.chainlog.append(
        entity_id=tokenized_card["token_id"],
        entity_type="payment",
        action="process",
        metadata={
            "amount": amount,
            "merchant_id": merchant_id,
            "last_four": payment_info["card"][-4:],
            "timestamp": time.time()
        }
    )
    
    # Process with payment processor
    payment_result = payment_processor.charge(
        token=tokenized_card["token"],
        amount=amount,
        merchant=merchant_id
    )
    
    # Log payment result with verification
    result_log = client.chainlog.append(
        entity_id=tokenized_card["token_id"],
        entity_type="payment",
        action="result",
        metadata={
            "status": payment_result["status"],
            "processor_id": payment_result["id"],
            "timestamp": time.time()
        }
    )
    
    return {
        "status": payment_result["status"],
        "transaction_id": payment_result["id"],
        "proof_id": result_log["id"]
    }
```

## Compliance Reports

### Available Reports

| Report Type | Constant | Purpose |
|-------------|----------|---------|
| General Compliance | `REPORT_GENERAL` | Overview of system compliance |
| GDPR | `REPORT_GDPR` | Data subject information |
| HIPAA | `REPORT_HIPAA` | Protected health information access |
| PCI DSS | `REPORT_PCI` | Cardholder data security |
| SOX | `REPORT_SOX` | Financial records integrity |
| Access Control | `REPORT_ACCESS` | Access decisions and patterns |
| Data Processing | `REPORT_PROCESSING` | Data workflow activities |

### Generating Reports

```python
# Generate a compliance report
report = client.export.generate_compliance_report(
    report_type=client.constants.REPORT_GDPR,
    entity_id="user123",
    start_time=1622548800,  # June 1, 2021
    end_time=1625140800,    # July 1, 2021
    format="pdf"            # Options: pdf, json, csv
)

# Access report
report_url = report["download_url"]
verification_id = report["proof_id"]

# Verify report hasn't been tampered with
verification = client.verify.validate_proof(verification_id)
if verification["valid"]:
    print("Report is cryptographically verified")
```

## Continuous Compliance Monitoring

```python
# Set up continuous compliance alerts
client.compliance.create_monitor(
    name="GDPR Data Retention",
    rule="data.retention_days > 90 AND data.category = 'personal'",
    entity_type="document",
    alert_level="warning",
    webhook_url="https://example.com/compliance-alerts"
)

# Set up scheduled compliance reports
client.compliance.schedule_report(
    report_type=client.constants.REPORT_ACCESS,
    schedule="weekly",
    params={
        "entity_type": "patient_record",
        "format": "pdf"
    },
    delivery={
        "email": ["compliance@example.com", "security@example.com"],
        "webhook": "https://example.com/compliance-reports"
    }
)
```

## Evidence Collection

### Immutable Evidence Chain

```python
# Build an evidence chain for an incident
def collect_incident_evidence(incident_id, start_time, end_time):
    # Create an evidence container
    evidence = client.verify.create_evidence_container(
        name=f"Incident {incident_id} Evidence",
        description="Automatically collected evidence"
    )
    
    # Collect system logs
    system_logs = client.chainlog.get_logs(
        entity_type="system",
        start_time=start_time,
        end_time=end_time
    )
    evidence.add_logs(system_logs)
    
    # Collect user access logs
    access_logs = client.chainlog.get_logs(
        entity_type="user",
        action="authentication",
        start_time=start_time,
        end_time=end_time
    )
    evidence.add_logs(access_logs)
    
    # Collect relevant documents with proof
    docs = client.data.get_documents_updated_between(start_time, end_time)
    evidence.add_documents(docs)
    
    # Seal the evidence to prevent changes
    sealed = evidence.seal()
    
    return {
        "evidence_id": sealed["id"],
        "verification_id": sealed["proof_id"],
        "item_count": sealed["item_count"]
    }
```

### Chain of Custody

```python
# Maintain chain of custody for sensitive data
def transfer_custody(asset_id, from_user, to_user, reason):
    # Record custody transfer with proof
    transfer = client.chainlog.append(
        entity_id=asset_id,
        entity_type="asset",
        action="custody_transfer",
        metadata={
            "from_user": from_user,
            "to_user": to_user,
            "reason": reason,
            "timestamp": time.time(),
            "location": get_location_info()
        }
    )
    
    # Verify custody chain is intact
    chain_verification = client.chainlog.verify_chain(asset_id, "asset")
    if not chain_verification["verified"]:
        raise CustodyChainException("Broken custody chain detected")
    
    return {
        "transfer_id": transfer["id"],
        "chain_verified": chain_verification["verified"]
    }
```

## Compliance Verification

```python
# Verify compliance requirements are met
def verify_compliance_status(system_id, framework="nist"):
    # Get compliance controls for framework
    controls = client.compliance.get_controls(framework)
    
    status = {
        "compliant": True,
        "controls": {},
        "verification_id": None
    }
    
    # Check each control
    for control in controls:
        control_status = client.compliance.check_control(
            system_id=system_id,
            control_id=control["id"]
        )
        
        status["controls"][control["id"]] = control_status
        if not control_status["compliant"]:
            status["compliant"] = False
    
    # Create verification proof of assessment
    verification = client.verify.create_proof({
        "system_id": system_id,
        "framework": framework,
        "timestamp": time.time(),
        "status": status
    })
    
    status["verification_id"] = verification["id"]
    return status
```

## Recommendations

1. **Map Requirements**: Document mapping between regulations and DGLA features
2. **Automate Evidence**: Set up automated evidence collection for all compliance needs
3. **Schedule Reports**: Configure regular compliance report generation
4. **Alert on Violations**: Implement real-time compliance monitoring
5. **Document Access**: Maintain detailed records of all compliance evidence access
6. **Verify Everything**: Use verification functions to validate compliance state
7. **Implement Training**: Ensure team understands cryptographic compliance benefits
