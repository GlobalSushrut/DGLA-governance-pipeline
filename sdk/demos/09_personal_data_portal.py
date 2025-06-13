#!/usr/bin/env python3
"""
Demo 9: Personal Data Control Portal

This application demonstrates a secure portal for individuals to manage
consent and access to their personal data across multiple services.

Features:
- Consent management with cryptographic proof
- Data access tracing
- Revocation of data access
- Transparency reporting
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
from dgla_sdk.constants import FORMAT_JSON, REPORT_GDPR

class PersonalDataPortal:
    """Personal data control portal with consent management"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        # Simple in-memory stores for demo purposes
        self.users = {}
        self.consents = {}
        self.access_logs = {}
        self.data_categories = [
            "identity", "contact", "financial", "health", 
            "location", "biometric", "social", "behavior"
        ]
        self.purposes = [
            "service_provision", "marketing", "analytics", 
            "research", "legal_compliance", "personalization"
        ]
    
    def register_user(self, user_id, name, email):
        """Register a new user of the portal"""
        timestamp = datetime.now().isoformat()
        
        # Create user record
        user = {
            'user_id': user_id,
            'name': name,
            'email': email,
            'registered_at': timestamp,
            'preferences': {
                'data_categories': {cat: False for cat in self.data_categories},
                'purposes': {purpose: False for purpose in self.purposes}
            }
        }
        
        self.users[user_id] = user
        self.consents[user_id] = []
        self.access_logs[user_id] = []
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=user_id,
            entity_type="data_subject",
            action="register",
            metadata={
                'timestamp': timestamp
            }
        )
        
        return {
            'user_id': user_id,
            'registered_at': timestamp
        }
    
    def manage_consent(self, user_id, service_id, data_categories, purposes, is_granted):
        """Grant or revoke consent for data usage"""
        if user_id not in self.users:
            return {'error': 'User not found'}
            
        timestamp = datetime.now().isoformat()
        consent_id = str(uuid.uuid4())
        
        # Validate data categories and purposes
        for cat in data_categories:
            if cat not in self.data_categories:
                return {'error': f'Invalid data category: {cat}'}
                
        for purpose in purposes:
            if purpose not in self.purposes:
                return {'error': f'Invalid purpose: {purpose}'}
        
        # Create consent record
        consent = {
            'consent_id': consent_id,
            'user_id': user_id,
            'service_id': service_id,
            'data_categories': data_categories,
            'purposes': purposes,
            'is_granted': is_granted,
            'timestamp': timestamp
        }
        
        # Add to user's consent records
        self.consents[user_id].append(consent)
        
        # Update user preferences
        if is_granted:
            for cat in data_categories:
                self.users[user_id]['preferences']['data_categories'][cat] = True
            for purpose in purposes:
                self.users[user_id]['preferences']['purposes'][purpose] = True
        
        # Create cryptographic proof of consent
        proof = self.client.verify.create_proof({
            'consent_id': consent_id,
            'user_id': user_id,
            'service_id': service_id,
            'data_categories': data_categories,
            'purposes': purposes,
            'is_granted': is_granted,
            'timestamp': timestamp
        })
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=consent_id,
            entity_type="consent",
            action="grant" if is_granted else "revoke",
            metadata={
                'user_id': user_id,
                'service_id': service_id,
                'data_categories': data_categories,
                'purposes': purposes,
                'proof_id': proof.get('id'),
                'timestamp': timestamp
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="consent_changes",
            value=1,
            labels={
                'action': 'grant' if is_granted else 'revoke',
                'service_id': service_id
            }
        )
        
        return {
            'consent_id': consent_id,
            'user_id': user_id,
            'service_id': service_id,
            'proof_id': proof.get('id'),
            'timestamp': timestamp,
            'status': 'granted' if is_granted else 'revoked'
        }
    
    def log_data_access(self, user_id, service_id, data_categories, purpose, access_type):
        """Log access to user data by a service"""
        if user_id not in self.users:
            return {'error': 'User not found'}
            
        timestamp = datetime.now().isoformat()
        access_id = str(uuid.uuid4())
        
        # Verify consent exists for this access
        has_consent = False
        for consent in self.consents[user_id]:
            if (consent['service_id'] == service_id and 
                consent['is_granted'] and
                purpose in consent['purposes'] and
                all(cat in consent['data_categories'] for cat in data_categories)):
                has_consent = True
                break
        
        # Create access record
        access = {
            'access_id': access_id,
            'user_id': user_id,
            'service_id': service_id,
            'data_categories': data_categories,
            'purpose': purpose,
            'access_type': access_type,
            'has_consent': has_consent,
            'timestamp': timestamp
        }
        
        # Add to access logs
        self.access_logs[user_id].append(access)
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=access_id,
            entity_type="data_access",
            action=access_type,
            metadata={
                'user_id': user_id,
                'service_id': service_id,
                'data_categories': data_categories,
                'purpose': purpose,
                'has_consent': has_consent,
                'timestamp': timestamp
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="data_access",
            value=1,
            labels={
                'service_id': service_id,
                'has_consent': str(has_consent).lower(),
                'access_type': access_type
            }
        )
        
        return {
            'access_id': access_id,
            'has_consent': has_consent,
            'timestamp': timestamp,
            'status': 'authorized' if has_consent else 'unauthorized'
        }
    
    def get_user_access_history(self, user_id):
        """Get history of all data access for a user"""
        if user_id not in self.users:
            return {'error': 'User not found'}
            
        return {
            'user_id': user_id,
            'access_events': self.access_logs[user_id],
            'retrieved_at': datetime.now().isoformat()
        }
    
    def generate_transparency_report(self, user_id):
        """Generate GDPR-style transparency report for user"""
        if user_id not in self.users:
            return {'error': 'User not found'}
            
        timestamp = datetime.now().isoformat()
        report_id = str(uuid.uuid4())
        
        # Compile consent and access information
        current_consents = [c for c in self.consents[user_id] if c['is_granted']]
        services_with_access = {c['service_id'] for c in current_consents}
        
        # Start report generation through DGLA
        self.client.export.generate_compliance_report(
            report_type=REPORT_GDPR,
            entity_id=user_id,
            format=FORMAT_JSON
        )
        
        # In a real system, this would wait for the report to be ready
        
        # Log report generation in immutable log
        self.client.chainlog.append_log(
            entity_id=report_id,
            entity_type="transparency_report",
            action="generate",
            metadata={
                'user_id': user_id,
                'timestamp': timestamp
            }
        )
        
        return {
            'report_id': report_id,
            'user_id': user_id,
            'active_consents': len(current_consents),
            'services_with_access': len(services_with_access),
            'generated_at': timestamp,
            'report_url': f"https://reports.example.com/transparency/{report_id}"
        }

def main():
    """Main function to demonstrate personal data portal"""
    parser = argparse.ArgumentParser(description="Personal Data Control Portal Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    args = parser.parse_args()
    
    # Initialize data portal
    portal = PersonalDataPortal(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("🔐 DGLA Personal Data Control Portal Demo")
    print("========================================")
    
    # 1. Register a user
    print("\n1. Registering new user...")
    user_id = "user123456"
    user = portal.register_user(
        user_id=user_id,
        name="Jane Citizen",
        email="jane@example.com"
    )
    print(f"  ✓ User registered: {user_id}")
    
    # 2. User grants consent to services
    print("\n2. Managing consent for data usage...")
    
    # Social media app consent
    social_consent = portal.manage_consent(
        user_id=user_id,
        service_id="social_media_app",
        data_categories=["identity", "contact", "social", "behavior"],
        purposes=["service_provision", "personalization"],
        is_granted=True
    )
    print(f"  ✓ Consent granted to social media app")
    print(f"  ✓ Consent proof: {social_consent['proof_id']}")
    
    # Fitness app consent
    fitness_consent = portal.manage_consent(
        user_id=user_id,
        service_id="fitness_app",
        data_categories=["identity", "health", "location"],
        purposes=["service_provision", "analytics"],
        is_granted=True
    )
    print(f"  ✓ Consent granted to fitness app")
    
    # Marketing consent (revoked)
    marketing_consent = portal.manage_consent(
        user_id=user_id,
        service_id="marketing_service",
        data_categories=["contact", "behavior"],
        purposes=["marketing"],
        is_granted=False
    )
    print(f"  ✓ Consent explicitly denied to marketing service")
    
    # 3. Log data access events
    print("\n3. Recording data access by services...")
    
    # Authorized access
    social_access = portal.log_data_access(
        user_id=user_id,
        service_id="social_media_app",
        data_categories=["social", "behavior"],
        purpose="personalization",
        access_type="read"
    )
    print(f"  ✓ Social media app access: {social_access['status']}")
    
    # Another authorized access
    fitness_access = portal.log_data_access(
        user_id=user_id,
        service_id="fitness_app",
        data_categories=["health", "location"],
        purpose="service_provision",
        access_type="read"
    )
    print(f"  ✓ Fitness app access: {fitness_access['status']}")
    
    # Unauthorized access attempt
    marketing_access = portal.log_data_access(
        user_id=user_id,
        service_id="marketing_service",
        data_categories=["contact"],
        purpose="marketing",
        access_type="read"
    )
    print(f"  ✓ Marketing service access attempt: {marketing_access['status']}")
    
    # 4. Get user's access history
    print("\n4. Retrieving user's data access history...")
    history = portal.get_user_access_history(user_id)
    print(f"  ✓ Retrieved {len(history['access_events'])} access events")
    authorized = sum(1 for event in history['access_events'] if event['has_consent'])
    unauthorized = sum(1 for event in history['access_events'] if not event['has_consent'])
    print(f"  ✓ Authorized accesses: {authorized}")
    print(f"  ✓ Unauthorized access attempts: {unauthorized}")
    
    # 5. Generate transparency report
    print("\n5. Generating GDPR transparency report...")
    report = portal.generate_transparency_report(user_id)
    print(f"  ✓ Transparency report generated: {report['report_id']}")
    print(f"  ✓ Active consents: {report['active_consents']}")
    print(f"  ✓ Services with access: {report['services_with_access']}")
    print(f"  ✓ Report URL: {report['report_url']}")
    
    print("\nAll consent actions and data access events have been")
    print("cryptographically verified and recorded in the immutable audit log.")

if __name__ == "__main__":
    main()
