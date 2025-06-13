#!/usr/bin/env python3
"""
Demo 1: Secure Document Manager

This application demonstrates a document management system with
tamper-evident audit logs and cryptographic verification.

Features:
- Document upload with automatic hash verification
- Immutable audit trail for all document activities
- Cryptographic verification of document integrity
- Compliance reporting for document access
"""
import os
import sys
import hashlib
import argparse
from datetime import datetime, timedelta
import json

# Add parent directory to path to import the SDK
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from dgla_sdk import DGLAClient
from dgla_sdk.constants import COMPLIANCE_GDPR, FORMAT_JSON

class SecureDocumentManager:
    """Secure document management with tamper-evident audit trails"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        self.documents = {}  # Simple in-memory document store for demo
    
    def upload_document(self, doc_id, doc_content, metadata=None):
        """Upload document with cryptographic verification"""
        # Calculate document hash
        doc_hash = hashlib.sha256(doc_content.encode()).hexdigest()
        
        # Store document with metadata
        self.documents[doc_id] = {
            'content': doc_content,
            'hash': doc_hash,
            'metadata': metadata or {},
            'uploaded_at': datetime.now().isoformat()
        }
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=doc_id,
            entity_type="document",
            action="upload",
            metadata={
                "hash": doc_hash,
                "size": len(doc_content),
                **(metadata or {})
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="document_uploads",
            value=1,
            labels={"doc_type": metadata.get("doc_type", "unknown") if metadata else "unknown"}
        )
        
        return {
            'id': doc_id,
            'hash': doc_hash,
            'status': 'uploaded',
            'size': len(doc_content)
        }
    
    def view_document(self, doc_id, user_id):
        """View document with audit logging"""
        if doc_id not in self.documents:
            return {'error': 'Document not found'}
        
        # Record access in immutable audit log
        self.client.chainlog.append_log(
            entity_id=doc_id,
            entity_type="document",
            action="view",
            metadata={
                "user_id": user_id,
                "timestamp": datetime.now().isoformat()
            }
        )
        
        # Verify document integrity
        doc = self.documents[doc_id]
        current_hash = hashlib.sha256(doc['content'].encode()).hexdigest()
        integrity_verified = current_hash == doc['hash']
        
        if not integrity_verified:
            # Security incident - document tampering detected!
            self.client.chainlog.append_log(
                entity_id=doc_id,
                entity_type="security_incident",
                action="document_tampering_detected",
                metadata={
                    "original_hash": doc['hash'],
                    "current_hash": current_hash,
                    "user_id": user_id
                }
            )
            
            # Push security incident metric
            self.client.metrics.push_metric(
                metric_name="security_incidents",
                value=1,
                labels={"type": "document_tampering"}
            )
            
            return {
                'error': 'Document integrity violation',
                'details': 'The document has been modified since upload'
            }
        
        return {
            'id': doc_id,
            'content': doc['content'],
            'metadata': doc['metadata'],
            'integrity_verified': integrity_verified
        }
    
    def generate_compliance_report(self, start_date=None, end_date=None):
        """Generate compliance report for document access"""
        if not start_date:
            start_date = (datetime.now() - timedelta(days=30)).isoformat()
        
        if not end_date:
            end_date = datetime.now().isoformat()
            
        # Request GDPR compliance report
        report = self.client.export.generate_compliance_report(
            report_type=COMPLIANCE_GDPR,
            start_time=start_date,
            end_time=end_date
        )
        
        return report

def main():
    """Main function to demonstrate the secure document manager"""
    parser = argparse.ArgumentParser(description="Secure Document Manager Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    parser.add_argument("--user-id", default="demo_user", help="User ID for the demo")
    args = parser.parse_args()
    
    # Initialize document manager
    doc_manager = SecureDocumentManager(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("📄 DGLA Secure Document Manager Demo")
    print("=====================================")
    
    # 1. Upload a document
    print("\nUploading document...")
    doc_id = "contract_2025_06_11"
    doc_content = "This is a confidential contract between ACME Corp and Cyberdyne Systems."
    doc_metadata = {
        "doc_type": "contract",
        "classification": "confidential",
        "department": "legal"
    }
    
    upload_result = doc_manager.upload_document(doc_id, doc_content, doc_metadata)
    print(f"Upload result: {json.dumps(upload_result, indent=2)}")
    
    # 2. View the document
    print("\nViewing document...")
    view_result = doc_manager.view_document(doc_id, args.user_id)
    print(f"View result: {json.dumps(view_result, indent=2)}")
    
    # 3. Generate compliance report
    print("\nGenerating GDPR compliance report...")
    report_result = doc_manager.generate_compliance_report()
    print(f"Report generated: {json.dumps(report_result, indent=2)}")
    
    print("\nDemo completed. All actions have been recorded in the immutable audit log.")

if __name__ == "__main__":
    main()
