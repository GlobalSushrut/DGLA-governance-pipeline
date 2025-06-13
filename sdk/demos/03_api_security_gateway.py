#!/usr/bin/env python3
"""
Demo 3: API Security Gateway

This application demonstrates a secure API gateway that uses
zero-trust principles and continuous verification.

Features:
- Request authentication and authorization
- Rate limiting with quota enforcement
- API usage tracking and anomaly detection
- Immutable access audit trail
"""
import os
import sys
import argparse
import time
import random
import uuid
import json
from datetime import datetime

# Add parent directory to path to import the SDK
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from dgla_sdk import DGLAClient

class APISecurity:
    """API security gateway with zero-trust verification"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        self.api_keys = {}  # Simple in-memory API key store
        self.quotas = {}    # API usage quotas
    
    def register_api_key(self, client_id, permissions=None, quota_limit=100):
        """Register a new API key with permissions and quota"""
        api_key = str(uuid.uuid4())
        
        # Store API key details
        self.api_keys[api_key] = {
            'client_id': client_id,
            'permissions': permissions or [],
            'created_at': datetime.now().isoformat(),
            'last_verified': None
        }
        
        # Set quota for this key
        self.quotas[api_key] = {
            'limit': quota_limit,
            'usage': 0,
            'reset_at': None  # In a real system, this would be time-based
        }
        
        # Log key creation in immutable audit trail
        self.client.chainlog.append_log(
            entity_id=api_key,
            entity_type="api_key",
            action="create",
            metadata={
                "client_id": client_id,
                "permissions": permissions,
                "quota_limit": quota_limit
            }
        )
        
        return {
            'api_key': api_key,
            'client_id': client_id,
            'permissions': permissions,
            'quota_limit': quota_limit
        }
    
    def verify_request(self, api_key, endpoint, method="GET", payload=None):
        """Verify an API request using zero-trust principles"""
        verification_id = str(uuid.uuid4())
        request_time = datetime.now().isoformat()
        
        # Check if API key exists
        if api_key not in self.api_keys:
            # Log failed verification
            self.client.chainlog.append_log(
                entity_id=verification_id,
                entity_type="api_verification",
                action="failed",
                metadata={
                    "reason": "invalid_api_key",
                    "endpoint": endpoint,
                    "method": method
                }
            )
            
            # Update security metrics
            self.client.metrics.push_metric(
                metric_name="api_auth_failures",
                value=1,
                labels={"reason": "invalid_key", "endpoint": endpoint}
            )
            
            return {
                'status': 'denied',
                'reason': 'Invalid API key',
                'verification_id': verification_id
            }
        
        # Get API key details
        key_details = self.api_keys[api_key]
        quota = self.quotas[api_key]
        
        # Update verification timestamp
        key_details['last_verified'] = request_time
        
        # Check quota
        if quota['usage'] >= quota['limit']:
            # Log quota exceeded
            self.client.chainlog.append_log(
                entity_id=verification_id,
                entity_type="api_verification",
                action="failed",
                metadata={
                    "reason": "quota_exceeded",
                    "client_id": key_details['client_id'],
                    "endpoint": endpoint,
                    "method": method,
                    "quota_usage": quota['usage'],
                    "quota_limit": quota['limit']
                }
            )
            
            # Update metrics
            self.client.metrics.push_metric(
                metric_name="api_quota_exceeded",
                value=1,
                labels={"client_id": key_details['client_id']}
            )
            
            return {
                'status': 'denied',
                'reason': 'Quota exceeded',
                'verification_id': verification_id
            }
        
        # Check permissions (simplified for demo)
        endpoint_permission = f"{method.lower()}:{endpoint}"
        if endpoint_permission not in key_details['permissions'] and "*" not in key_details['permissions']:
            # Log permission denied
            self.client.chainlog.append_log(
                entity_id=verification_id,
                entity_type="api_verification",
                action="failed",
                metadata={
                    "reason": "permission_denied",
                    "client_id": key_details['client_id'],
                    "endpoint": endpoint,
                    "method": method,
                    "required_permission": endpoint_permission
                }
            )
            
            # Update metrics
            self.client.metrics.push_metric(
                metric_name="api_permission_denied",
                value=1,
                labels={"client_id": key_details['client_id'], "endpoint": endpoint}
            )
            
            return {
                'status': 'denied',
                'reason': 'Permission denied',
                'verification_id': verification_id
            }
        
        # Request is valid, update quota usage
        quota['usage'] += 1
        
        # Create verifiable proof of verification
        proof = self.client.verify.create_proof({
            "client_id": key_details['client_id'],
            "api_key_hash": api_key[:8],  # Only use part of key for proof
            "endpoint": endpoint,
            "method": method,
            "timestamp": request_time,
            "verification_id": verification_id
        })
        
        # Log successful verification
        self.client.chainlog.append_log(
            entity_id=verification_id,
            entity_type="api_verification",
            action="success",
            metadata={
                "client_id": key_details['client_id'],
                "endpoint": endpoint,
                "method": method,
                "quota_usage": quota['usage'],
                "quota_limit": quota['limit'],
                "proof_id": proof.get("id")
            }
        )
        
        # Update metrics
        self.client.metrics.push_metric(
            metric_name="api_requests",
            value=1,
            labels={"client_id": key_details['client_id'], "endpoint": endpoint, "method": method}
        )
        
        return {
            'status': 'approved',
            'client_id': key_details['client_id'],
            'verification_id': verification_id,
            'proof_id': proof.get("id"),
            'quota': {
                'used': quota['usage'],
                'limit': quota['limit']
            }
        }

def main():
    """Main function to demonstrate API security gateway"""
    parser = argparse.ArgumentParser(description="API Security Gateway Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    args = parser.parse_args()
    
    # Initialize API security
    api_security = APISecurity(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("🔒 DGLA API Security Gateway Demo")
    print("=================================")
    
    # 1. Register two API keys for different clients
    print("\n1. Registering API keys for clients...")
    
    # Client with full access
    admin_registration = api_security.register_api_key(
        client_id="admin-service",
        permissions=["*"],  # Wildcard for all permissions
        quota_limit=1000
    )
    admin_key = admin_registration['api_key']
    print(f"  ✓ Registered admin API key: {admin_key[:8]}...")
    
    # Client with limited access
    app_registration = api_security.register_api_key(
        client_id="mobile-app",
        permissions=[
            "get:users",
            "get:products",
            "post:orders"
        ],
        quota_limit=50
    )
    app_key = app_registration['api_key']
    print(f"  ✓ Registered app API key: {app_key[:8]}...")
    
    # 2. Verify valid requests
    print("\n2. Verifying valid API requests...")
    
    # Admin accessing users
    admin_result = api_security.verify_request(
        api_key=admin_key,
        endpoint="users",
        method="GET"
    )
    print(f"  ✓ Admin accessing users: {admin_result['status']}")
    
    # App accessing users
    app_result = api_security.verify_request(
        api_key=app_key,
        endpoint="users",
        method="GET"
    )
    print(f"  ✓ App accessing users: {app_result['status']}")
    
    # 3. Verify invalid permission
    print("\n3. Testing permission enforcement...")
    
    # App trying to access admin endpoint
    invalid_result = api_security.verify_request(
        api_key=app_key,
        endpoint="admin/settings",
        method="POST"
    )
    print(f"  ✓ App accessing admin endpoint: {invalid_result['status']} - {invalid_result['reason']}")
    
    # 4. Quota management
    print("\n4. Testing quota enforcement...")
    
    # Simulate app reaching quota limit
    print("  ✓ Simulating app reaching quota limit...")
    api_security.quotas[app_key]['usage'] = app_registration['quota_limit']
    
    # App trying one more request
    quota_result = api_security.verify_request(
        api_key=app_key,
        endpoint="products",
        method="GET"
    )
    print(f"  ✓ App request after quota exceeded: {quota_result['status']} - {quota_result['reason']}")
    
    print("\nAll API requests have been verified with cryptographic proof and recorded in the immutable audit log.")

if __name__ == "__main__":
    main()
