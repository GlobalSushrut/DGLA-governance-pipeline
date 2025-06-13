#!/usr/bin/env python3
"""
Demo 2: Security Monitoring Dashboard

This application demonstrates real-time security monitoring with
custom metrics, alerts, and visualization integration.

Features:
- Real-time security event monitoring
- Custom metrics for security incidents
- Automated alerts for threshold breaches
- Grafana dashboard integration
"""
import os
import sys
import argparse
import time
import random
from datetime import datetime
import json

# Add parent directory to path to import the SDK
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from dgla_sdk import DGLAClient
from dgla_sdk.constants import COMPARISON_GT, COMPARISON_LT

class SecurityMonitoringDashboard:
    """Real-time security monitoring with metrics and alerts"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        self.initialize_alerts()
    
    def initialize_alerts(self):
        """Set up initial alert thresholds"""
        # Alert when login failures exceed 5 in monitoring period
        self.client.metrics.create_alert(
            metric_name="login_failures",
            threshold=5.0,
            comparison=COMPARISON_GT,
            duration="5m"
        )
        
        # Alert on suspicious API calls
        self.client.metrics.create_alert(
            metric_name="suspicious_api_calls",
            threshold=3.0,
            comparison=COMPARISON_GT,
            duration="10m"
        )
        
        # Alert on abnormal data access patterns
        self.client.metrics.create_alert(
            metric_name="abnormal_data_access",
            threshold=1.0,
            comparison=COMPARISON_GT,
            duration="1m"
        )
    
    def simulate_login_failures(self, count=1, username=None):
        """Simulate login failures for monitoring"""
        # Generate random username if none provided
        if not username:
            username = f"user{random.randint(1000, 9999)}"
        
        # Log each failure
        for _ in range(count):
            # Record in immutable audit log
            self.client.chainlog.append_log(
                entity_id=username,
                entity_type="auth",
                action="login_failure",
                metadata={
                    "ip_address": f"192.168.1.{random.randint(2, 254)}",
                    "timestamp": datetime.now().isoformat(),
                    "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
                }
            )
            
            # Push metric
            self.client.metrics.push_metric(
                metric_name="login_failures",
                value=1,
                labels={"username": username}
            )
    
    def simulate_suspicious_api_call(self, api_name, source_ip=None):
        """Simulate suspicious API call detection"""
        if not source_ip:
            source_ip = f"203.0.{random.randint(1, 255)}.{random.randint(1, 255)}"
        
        incident_id = f"api-incident-{int(time.time())}"
        
        # Log suspicious activity
        self.client.chainlog.append_log(
            entity_id=incident_id,
            entity_type="security_incident",
            action="suspicious_api_call",
            metadata={
                "api_name": api_name,
                "source_ip": source_ip,
                "timestamp": datetime.now().isoformat(),
                "risk_score": random.uniform(0.7, 0.95)
            }
        )
        
        # Push metric
        self.client.metrics.push_metric(
            metric_name="suspicious_api_calls",
            value=1,
            labels={"api": api_name, "source_ip": source_ip}
        )
        
        return incident_id
    
    def simulate_abnormal_data_access(self, data_id, user_id, severity="high"):
        """Simulate detection of abnormal data access pattern"""
        incident_id = f"access-incident-{int(time.time())}"
        
        # Determine risk score based on severity
        risk_scores = {
            "low": random.uniform(0.5, 0.7),
            "medium": random.uniform(0.7, 0.85),
            "high": random.uniform(0.85, 0.98)
        }
        risk_score = risk_scores.get(severity, 0.75)
        
        # Log the incident
        self.client.chainlog.append_log(
            entity_id=incident_id,
            entity_type="security_incident",
            action="abnormal_data_access",
            metadata={
                "data_id": data_id,
                "user_id": user_id,
                "timestamp": datetime.now().isoformat(),
                "risk_score": risk_score,
                "severity": severity
            }
        )
        
        # Push metric
        self.client.metrics.push_metric(
            metric_name="abnormal_data_access",
            value=1,
            labels={"data_id": data_id, "severity": severity}
        )
        
        return incident_id
    
    def get_grafana_dashboard_url(self):
        """Get URL for Grafana security dashboard"""
        base_info = self.client.metrics.get_grafana_url()
        
        # In a real implementation, this would return a specific dashboard URL
        # For demo purposes, we'll construct a simulated one
        if "url" in base_info:
            return f"{base_info['url']}/d/security/dgla-security-monitoring-dashboard"
        return None

def main():
    """Main function to demonstrate security monitoring"""
    parser = argparse.ArgumentParser(description="Security Monitoring Dashboard Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    parser.add_argument("--scenarios", default="all", help="Comma-separated list of scenarios to run")
    args = parser.parse_args()
    
    # Initialize security dashboard
    security_dashboard = SecurityMonitoringDashboard(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("🔐 DGLA Security Monitoring Dashboard Demo")
    print("=========================================")
    
    # Determine which scenarios to run
    scenarios = args.scenarios.lower().split(',') if args.scenarios != "all" else ["logins", "api", "access"]
    
    if "logins" in scenarios:
        # 1. Simulate login failure pattern
        print("\n1. Simulating login failure pattern...")
        security_dashboard.simulate_login_failures(count=3, username="admin")
        time.sleep(1)
        security_dashboard.simulate_login_failures(count=4, username="admin")
        print("✓ Login failure pattern simulation complete.")
    
    if "api" in scenarios:
        # 2. Simulate suspicious API calls
        print("\n2. Simulating suspicious API calls...")
        apis = ["getUserData", "transferFunds", "updateCredentials"]
        for api in apis:
            incident_id = security_dashboard.simulate_suspicious_api_call(api)
            print(f"  ✓ Recorded suspicious API call to {api} (Incident: {incident_id})")
            time.sleep(0.5)
    
    if "access" in scenarios:
        # 3. Simulate abnormal data access
        print("\n3. Simulating abnormal data access patterns...")
        data_types = [
            ("customer_financial_data", "user123", "high"),
            ("hrdb_salary_records", "admin", "medium"),
            ("system_credentials", "sysuser", "high")
        ]
        
        for data_id, user_id, severity in data_types:
            incident_id = security_dashboard.simulate_abnormal_data_access(data_id, user_id, severity)
            print(f"  ✓ Recorded abnormal access to {data_id} by {user_id} (Severity: {severity}, Incident: {incident_id})")
            time.sleep(0.5)
    
    # Get Grafana dashboard URL
    dashboard_url = security_dashboard.get_grafana_dashboard_url()
    print(f"\nAll security events have been logged. View the dashboard at: {dashboard_url}")

if __name__ == "__main__":
    main()
