#!/usr/bin/env python3
"""
Demo 5: Financial Transaction Monitor

This application demonstrates real-time financial transaction monitoring
with fraud detection and regulatory compliance features.

Features:
- Real-time transaction verification
- Pattern-based fraud detection
- Transaction audit logging
- AML/KYC compliance reporting
"""
import os
import sys
import argparse
import time
import random
import uuid
from datetime import datetime, timedelta

# Add parent directory to path to import the SDK
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from dgla_sdk import DGLAClient
from dgla_sdk.constants import FORMAT_JSON

class FinancialTransactionMonitor:
    """Financial transaction monitoring with fraud detection"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        # In a real system, these would be ML models or rule engines
        self.risk_thresholds = {
            "amount": 10000.0,           # Flag high-value transactions
            "velocity": 5,               # Flag many transactions in short time
            "unusual_location": 0.8,     # Probability threshold for location anomaly
            "unusual_merchant": 0.75     # Probability threshold for unusual merchant
        }
    
    def process_transaction(self, transaction_data):
        """Process and evaluate a financial transaction"""
        # Generate transaction ID if not provided
        if "transaction_id" not in transaction_data:
            transaction_data["transaction_id"] = str(uuid.uuid4())
        
        # Add timestamp if not provided
        if "timestamp" not in transaction_data:
            transaction_data["timestamp"] = datetime.now().isoformat()
        
        # Calculate risk score
        risk_score, risk_factors = self._calculate_risk(transaction_data)
        transaction_data["risk_score"] = risk_score
        transaction_data["risk_factors"] = risk_factors
        
        # Determine if transaction should be approved
        approved = risk_score < 0.7  # Example threshold
        transaction_data["approved"] = approved
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=transaction_data["transaction_id"],
            entity_type="financial_transaction",
            action="process",
            metadata={
                "account_id": transaction_data["account_id"],
                "amount": transaction_data["amount"],
                "currency": transaction_data["currency"],
                "merchant": transaction_data["merchant"],
                "risk_score": risk_score,
                "approved": approved,
                "timestamp": transaction_data["timestamp"]
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="transaction_volume",
            value=transaction_data["amount"],
            labels={"currency": transaction_data["currency"]}
        )
        
        if len(risk_factors) > 0:
            self.client.metrics.push_metric(
                metric_name="flagged_transactions",
                value=1,
                labels={"primary_factor": risk_factors[0] if risk_factors else "none"}
            )
        
        return {
            "transaction_id": transaction_data["transaction_id"],
            "approved": approved,
            "risk_score": risk_score,
            "risk_factors": risk_factors,
            "needs_review": risk_score >= 0.5 and risk_score < 0.7,
            "timestamp": transaction_data["timestamp"]
        }
    
    def _calculate_risk(self, transaction):
        """Calculate risk score for a transaction"""
        risk_factors = []
        risk_score = 0.0
        
        # Check for high value
        if transaction["amount"] > self.risk_thresholds["amount"]:
            risk_factors.append("high_value")
            risk_score += 0.4
        
        # Check for unusual location (simplified for demo)
        if "location" in transaction and random.random() > self.risk_thresholds["unusual_location"]:
            risk_factors.append("unusual_location")
            risk_score += 0.3
        
        # Check for unusual merchant (simplified for demo)
        if random.random() > self.risk_thresholds["unusual_merchant"]:
            risk_factors.append("unusual_merchant")
            risk_score += 0.25
        
        # Add some randomness for demo purposes
        risk_score = min(0.95, risk_score + (random.random() * 0.2))
        
        return risk_score, risk_factors
    
    def generate_aml_report(self, account_id, start_date=None, end_date=None):
        """Generate Anti-Money Laundering report"""
        if not start_date:
            start_date = (datetime.now() - timedelta(days=30)).isoformat()
        
        if not end_date:
            end_date = datetime.now().isoformat()
        
        report_id = str(uuid.uuid4())
        
        # Log report generation
        self.client.chainlog.append_log(
            entity_id=report_id,
            entity_type="compliance_report",
            action="aml_report",
            metadata={
                "account_id": account_id,
                "start_date": start_date,
                "end_date": end_date,
                "timestamp": datetime.now().isoformat()
            }
        )
        
        # In a real system, this would query transaction history
        # and generate a comprehensive AML report
        return {
            "report_id": report_id,
            "account_id": account_id,
            "period": {
                "start": start_date,
                "end": end_date
            },
            "report_url": f"https://reports.example.com/aml/{report_id}"
        }

def main():
    """Main function to demonstrate financial transaction monitoring"""
    parser = argparse.ArgumentParser(description="Financial Transaction Monitor Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    args = parser.parse_args()
    
    # Initialize transaction monitor
    monitor = FinancialTransactionMonitor(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("💰 DGLA Financial Transaction Monitor Demo")
    print("=========================================")
    
    # 1. Process some normal transactions
    print("\n1. Processing standard transactions...")
    standard_transactions = [
        {
            "account_id": "acct_12345",
            "amount": 125.50,
            "currency": "USD",
            "merchant": "Grocery Store Inc",
            "location": "New York, NY"
        },
        {
            "account_id": "acct_12345",
            "amount": 49.99,
            "currency": "USD",
            "merchant": "Online Streaming Service",
            "location": "Online"
        }
    ]
    
    for t in standard_transactions:
        result = monitor.process_transaction(t)
        print(f"  ✓ Transaction {result['transaction_id'][:8]}: " +
              f"Risk {result['risk_score']:.2f}, Approved: {result['approved']}")
    
    # 2. Process a high-risk transaction
    print("\n2. Processing high-risk transaction...")
    high_risk = {
        "account_id": "acct_12345",
        "amount": 25000.00,
        "currency": "USD",
        "merchant": "Foreign Exchange Service",
        "location": "Unexpected Location"
    }
    
    high_result = monitor.process_transaction(high_risk)
    print(f"  ✓ Transaction {high_result['transaction_id'][:8]}: " +
          f"Risk {high_result['risk_score']:.2f}, Approved: {high_result['approved']}")
    print(f"  ✓ Risk factors: {', '.join(high_result['risk_factors'])}")
    
    # 3. Generate AML report
    print("\n3. Generating Anti-Money Laundering report...")
    report = monitor.generate_aml_report("acct_12345")
    print(f"  ✓ AML report generated: {report['report_id']}")
    print(f"  ✓ Report URL: {report['report_url']}")
    
    print("\nAll transactions have been recorded in the immutable audit log")
    print("with cryptographic verification and real-time risk assessment.")

if __name__ == "__main__":
    main()
