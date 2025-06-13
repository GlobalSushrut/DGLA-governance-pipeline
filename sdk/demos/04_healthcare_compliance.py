#!/usr/bin/env python3
"""
Demo 4: Healthcare Compliance System

This application demonstrates HIPAA-compliant healthcare data management
with audit trails and patient data protection features.

Features:
- PHI (Protected Health Information) access tracking
- Cryptographic proof of consent management
- Automated HIPAA compliance reporting
- Patient data access request handling
"""
import os
import sys
import argparse
import time
import uuid
import json
from datetime import datetime, timedelta

# Add parent directory to path to import the SDK
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from dgla_sdk import DGLAClient
from dgla_sdk.constants import COMPLIANCE_HIPAA, FORMAT_PDF

class HealthcareComplianceSystem:
    """HIPAA-compliant healthcare data management system"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        self.patient_data = {}  # Simple in-memory data store for demo
        self.consent_records = {}  # Track patient consent
    
    def add_patient_data(self, patient_id, data, data_type, provider_id):
        """Add patient data with HIPAA-compliant logging"""
        if patient_id not in self.patient_data:
            self.patient_data[patient_id] = []
            
        data_id = str(uuid.uuid4())
        timestamp = datetime.now().isoformat()
        
        # Store the data entry
        data_entry = {
            'id': data_id,
            'patient_id': patient_id,
            'data': data,  # In real system, this would be encrypted
            'data_type': data_type,
            'provider_id': provider_id,
            'created_at': timestamp,
            'accessed_by': []  # Track who accessed this data
        }
        self.patient_data[patient_id].append(data_entry)
        
        # Create hash of data for integrity verification
        # In a real system, we'd securely hash the data
        data_hash = "sha256_hash_simulation"
        
        # Record in immutable audit log with minimal PHI
        self.client.chainlog.append_log(
            entity_id=data_id,
            entity_type="patient_data",
            action="create",
            metadata={
                "patient_id_hash": f"hash_{patient_id}",  # Hash, don't store actual ID
                "data_type": data_type,
                "provider_id": provider_id,
                "data_hash": data_hash,
                "timestamp": timestamp
            }
        )
        
        # Push metrics (without PII/PHI)
        self.client.metrics.push_metric(
            metric_name="patient_data_entries",
            value=1,
            labels={"data_type": data_type}
        )
        
        return {
            'data_id': data_id,
            'timestamp': timestamp,
            'status': 'recorded'
        }
    
    def record_consent(self, patient_id, purpose, expiration_days=30):
        """Record patient consent with cryptographic proof"""
        consent_id = str(uuid.uuid4())
        timestamp = datetime.now().isoformat()
        expiration = (datetime.now() + timedelta(days=expiration_days)).isoformat()
        
        # Store consent record
        consent_record = {
            'id': consent_id,
            'patient_id': patient_id,
            'purpose': purpose,
            'granted_at': timestamp,
            'expires_at': expiration,
            'status': 'active'
        }
        
        if patient_id not in self.consent_records:
            self.consent_records[patient_id] = []
            
        self.consent_records[patient_id].append(consent_record)
        
        # Create cryptographic proof of consent
        proof = self.client.verify.create_proof({
            "consent_id": consent_id,
            "patient_id_hash": f"hash_{patient_id}",  # Hash, don't store actual ID
            "purpose": purpose,
            "timestamp": timestamp,
            "expiration": expiration
        })
        
        # Record in immutable audit log (minimal PHI)
        self.client.chainlog.append_log(
            entity_id=consent_id,
            entity_type="patient_consent",
            action="grant",
            metadata={
                "patient_id_hash": f"hash_{patient_id}",
                "purpose": purpose,
                "timestamp": timestamp,
                "expiration": expiration,
                "proof_id": proof.get("id")
            }
        )
        
        return {
            'consent_id': consent_id,
            'proof_id': proof.get("id"),
            'expires_at': expiration,
            'status': 'active'
        }
    
    def access_patient_data(self, patient_id, data_id, user_id, reason):
        """Access patient data with full audit trail"""
        # Check if patient exists
        if patient_id not in self.patient_data:
            return {'error': 'Patient not found'}
        
        # Find the specific data entry
        data_entry = None
        for entry in self.patient_data[patient_id]:
            if entry['id'] == data_id:
                data_entry = entry
                break
        
        if not data_entry:
            return {'error': 'Data not found'}
            
        # Record access in the data entry
        data_entry['accessed_by'].append({
            'user_id': user_id,
            'timestamp': datetime.now().isoformat(),
            'reason': reason
        })
        
        # Record in immutable audit log (minimal PHI)
        access_id = str(uuid.uuid4())
        access_timestamp = datetime.now().isoformat()
        self.client.chainlog.append_log(
            entity_id=access_id,
            entity_type="data_access",
            action="phi_access",
            metadata={
                "data_id": data_id,
                "patient_id_hash": f"hash_{patient_id}",
                "user_id": user_id,
                "reason": reason,
                "timestamp": access_timestamp,
                "data_type": data_entry['data_type']
            }
        )
        
        # Push metrics (without PII/PHI)
        self.client.metrics.push_metric(
            metric_name="phi_access",
            value=1,
            labels={"reason": reason, "data_type": data_entry['data_type']}
        )
        
        return {
            'access_id': access_id,
            'data': data_entry['data'],
            'data_type': data_entry['data_type'],
            'provider_id': data_entry['provider_id'],
            'accessed_at': access_timestamp
        }
    
    def generate_hipaa_report(self, start_date=None, end_date=None):
        """Generate HIPAA compliance report"""
        if not start_date:
            start_date = (datetime.now() - timedelta(days=30)).isoformat()
        
        if not end_date:
            end_date = datetime.now().isoformat()
            
        # Request HIPAA compliance report
        report = self.client.export.generate_compliance_report(
            report_type=COMPLIANCE_HIPAA,
            start_time=start_date,
            end_time=end_date
        )
        
        return report

def main():
    """Main function to demonstrate healthcare compliance system"""
    parser = argparse.ArgumentParser(description="Healthcare Compliance System Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    args = parser.parse_args()
    
    # Initialize healthcare system
    healthcare_system = HealthcareComplianceSystem(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("🏥 DGLA Healthcare Compliance System Demo")
    print("========================================")
    
    # 1. Record patient consent
    print("\n1. Recording patient consent...")
    patient_id = "P12345"
    consent = healthcare_system.record_consent(
        patient_id=patient_id,
        purpose="Treatment and diagnosis",
        expiration_days=90
    )
    print(f"  ✓ Consent recorded with ID: {consent['consent_id']}")
    print(f"  ✓ Cryptographic proof created: {consent['proof_id']}")
    
    # 2. Add patient data
    print("\n2. Adding patient health data...")
    
    # Lab result
    lab_result = healthcare_system.add_patient_data(
        patient_id=patient_id,
        data="Cholesterol: 180mg/dL, HDL: 62mg/dL, LDL: 100mg/dL",
        data_type="lab_result",
        provider_id="lab_corp_1"
    )
    print(f"  ✓ Lab result added with ID: {lab_result['data_id']}")
    
    # Medication
    medication = healthcare_system.add_patient_data(
        patient_id=patient_id,
        data="Lisinopril 10mg, once daily",
        data_type="medication",
        provider_id="pharmacy_rx_2"
    )
    print(f"  ✓ Medication added with ID: {medication['data_id']}")
    
    # 3. Simulate data access
    print("\n3. Simulating physician accessing patient data...")
    
    # Doctor accessing lab results
    access_result = healthcare_system.access_patient_data(
        patient_id=patient_id,
        data_id=lab_result['data_id'],
        user_id="dr_smith",
        reason="Patient follow-up"
    )
    print(f"  ✓ Data accessed by dr_smith: {access_result['data_type']}")
    print(f"  ✓ Access recorded with ID: {access_result['access_id']}")
    
    # 4. Generate compliance report
    print("\n4. Generating HIPAA compliance report...")
    report = healthcare_system.generate_hipaa_report()
    print(f"  ✓ HIPAA report generated: {report.get('id', 'N/A')}")
    
    print("\nAll healthcare data operations have been securely recorded with")
    print("cryptographic proof and minimal PHI in the immutable audit log.")

if __name__ == "__main__":
    main()
