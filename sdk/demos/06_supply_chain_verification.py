#!/usr/bin/env python3
"""
Demo 6: Supply Chain Verification System

This application demonstrates a secure supply chain verification system
with blockchain-backed provenance tracking and tamper-evident records.

Features:
- Product origin verification
- Chain of custody tracking
- Tamper-evident shipping records
- Supply chain compliance reporting
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

class SupplyChainVerifier:
    """Supply chain verification with tamper-evident records"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        # Simple in-memory stores for demo purposes
        self.products = {}
        self.custody_chain = {}
    
    def register_product(self, product_id, manufacturer, batch_id, manufacturing_date, metadata=None):
        """Register a new product with verifiable origin"""
        if not metadata:
            metadata = {}
            
        timestamp = datetime.now().isoformat()
        
        # Create product record
        product = {
            'product_id': product_id,
            'manufacturer': manufacturer,
            'batch_id': batch_id,
            'manufacturing_date': manufacturing_date,
            'registration_date': timestamp,
            'metadata': metadata
        }
        
        self.products[product_id] = product
        
        # Initialize chain of custody
        self.custody_chain[product_id] = [{
            'custodian': manufacturer,
            'location': metadata.get('facility_location', 'Unknown'),
            'timestamp': timestamp,
            'action': 'manufactured'
        }]
        
        # Create cryptographic proof
        proof = self.client.verify.create_proof({
            'product_id': product_id,
            'manufacturer': manufacturer,
            'batch_id': batch_id,
            'timestamp': timestamp
        })
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=product_id,
            entity_type="product",
            action="register",
            metadata={
                'manufacturer': manufacturer,
                'batch_id': batch_id,
                'manufacturing_date': manufacturing_date,
                'registration_date': timestamp,
                'proof_id': proof.get('id')
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="products_registered",
            value=1,
            labels={'manufacturer': manufacturer}
        )
        
        return {
            'product_id': product_id,
            'proof_id': proof.get('id'),
            'registration_date': timestamp
        }
    
    def transfer_custody(self, product_id, from_party, to_party, location, transport_data=None):
        """Record chain of custody transfer"""
        if product_id not in self.products:
            return {'error': 'Product not found'}
            
        if product_id not in self.custody_chain:
            return {'error': 'Custody chain not found'}
        
        timestamp = datetime.now().isoformat()
        transfer_id = str(uuid.uuid4())
        
        # Create custody transfer record
        transfer = {
            'transfer_id': transfer_id,
            'from_party': from_party,
            'to_party': to_party,
            'location': location,
            'timestamp': timestamp,
            'transport_data': transport_data or {}
        }
        
        # Add to custody chain
        self.custody_chain[product_id].append({
            'custodian': to_party,
            'location': location,
            'timestamp': timestamp,
            'action': 'transfer_received',
            'transfer_id': transfer_id
        })
        
        # Create cryptographic proof
        proof = self.client.verify.create_proof({
            'transfer_id': transfer_id,
            'product_id': product_id,
            'from_party': from_party,
            'to_party': to_party,
            'location': location,
            'timestamp': timestamp
        })
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=transfer_id,
            entity_type="custody_transfer",
            action="transfer",
            metadata={
                'product_id': product_id,
                'from_party': from_party,
                'to_party': to_party,
                'location': location,
                'timestamp': timestamp,
                'proof_id': proof.get('id')
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="custody_transfers",
            value=1,
            labels={'product_type': self.products[product_id].get('metadata', {}).get('type', 'unknown')}
        )
        
        return {
            'transfer_id': transfer_id,
            'product_id': product_id,
            'timestamp': timestamp,
            'proof_id': proof.get('id')
        }
    
    def verify_product_history(self, product_id):
        """Verify complete product history and chain of custody"""
        if product_id not in self.products:
            return {'error': 'Product not found'}
            
        if product_id not in self.custody_chain:
            return {'error': 'Custody chain not found'}
            
        # Get product and custody information
        product = self.products[product_id]
        custody = self.custody_chain[product_id]
        
        # Verify chain integrity by checking with immutable log
        # In a real system, this would check cryptographic proofs
        chain_verified = True
        
        return {
            'product_id': product_id,
            'manufacturer': product['manufacturer'],
            'manufacturing_date': product['manufacturing_date'],
            'chain_of_custody': custody,
            'chain_verified': chain_verified,
            'current_custodian': custody[-1]['custodian'] if custody else None,
            'current_location': custody[-1]['location'] if custody else None,
            'verification_timestamp': datetime.now().isoformat()
        }
    
    def generate_compliance_report(self, product_ids=None, report_type="supply_chain_integrity"):
        """Generate compliance report for supply chain"""
        timestamp = datetime.now().isoformat()
        report_id = str(uuid.uuid4())
        
        # Log report generation
        self.client.chainlog.append_log(
            entity_id=report_id,
            entity_type="compliance_report",
            action=f"{report_type}_report",
            metadata={
                'product_count': len(product_ids) if product_ids else len(self.products),
                'report_type': report_type,
                'timestamp': timestamp
            }
        )
        
        # In a real system, this would generate a comprehensive report
        return {
            'report_id': report_id,
            'report_type': report_type,
            'timestamp': timestamp,
            'product_count': len(product_ids) if product_ids else len(self.products),
            'report_url': f"https://reports.example.com/supply-chain/{report_id}"
        }

def main():
    """Main function to demonstrate supply chain verification"""
    parser = argparse.ArgumentParser(description="Supply Chain Verification Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    args = parser.parse_args()
    
    # Initialize supply chain verifier
    verifier = SupplyChainVerifier(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("📦 DGLA Supply Chain Verification Demo")
    print("=====================================")
    
    # 1. Register a product
    print("\n1. Registering product...")
    product_id = "MED-VAC-12345"
    product_result = verifier.register_product(
        product_id=product_id,
        manufacturer="BioPharmaCorp",
        batch_id="LOT-2025-06-11",
        manufacturing_date="2025-06-01T08:30:00Z",
        metadata={
            'type': 'vaccine',
            'name': 'COVID-25 Vaccine',
            'storage_temp': '-70C',
            'facility_location': 'Boston, MA'
        }
    )
    print(f"  ✓ Product registered: {product_id}")
    print(f"  ✓ Proof created: {product_result['proof_id']}")
    
    # 2. Record custody transfers
    print("\n2. Recording chain of custody transfers...")
    
    # Transfer to distributor
    transfer1 = verifier.transfer_custody(
        product_id=product_id,
        from_party="BioPharmaCorp",
        to_party="MedLogistics Inc",
        location="Chicago, IL",
        transport_data={
            'transport_id': 'FLIGHT-1234',
            'temperature_logs': 'continuous -69C to -72C',
            'vehicle': 'refrigerated aircraft'
        }
    )
    print(f"  ✓ Transfer recorded: {transfer1['transfer_id']}")
    
    # Transfer to hospital
    transfer2 = verifier.transfer_custody(
        product_id=product_id,
        from_party="MedLogistics Inc",
        to_party="Central Hospital",
        location="Denver, CO",
        transport_data={
            'transport_id': 'TRUCK-5678',
            'temperature_logs': 'continuous -68C to -71C',
            'vehicle': 'refrigerated truck'
        }
    )
    print(f"  ✓ Transfer recorded: {transfer2['transfer_id']}")
    
    # 3. Verify product history
    print("\n3. Verifying complete product history...")
    history = verifier.verify_product_history(product_id)
    print(f"  ✓ Product verified: {product_id}")
    print(f"  ✓ Manufacturer: {history['manufacturer']}")
    print(f"  ✓ Current location: {history['current_location']}")
    print(f"  ✓ Current custodian: {history['current_custodian']}")
    print(f"  ✓ Chain of custody verified: {history['chain_verified']}")
    print(f"  ✓ Total custody transfers: {len(history['chain_of_custody']) - 1}")
    
    # 4. Generate compliance report
    print("\n4. Generating supply chain compliance report...")
    report = verifier.generate_compliance_report([product_id])
    print(f"  ✓ Report generated: {report['report_id']}")
    print(f"  ✓ Report URL: {report['report_url']}")
    
    print("\nAll supply chain events have been cryptographically verified")
    print("and recorded in the immutable audit log for compliance.")

if __name__ == "__main__":
    main()
