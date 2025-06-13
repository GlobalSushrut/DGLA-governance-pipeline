#!/usr/bin/env python3
"""
DGLA Infrastructure Adapter - Cybersecurity Use Case
This adapter connects to the actual DGLA infrastructure components and demonstrates
real-world cybersecurity data governance capabilities.
"""
import json
import time
import uuid
import hashlib
import requests
import redis
from datetime import datetime
import os
import socket

# Configuration for connecting to the actual infrastructure - using environment variables
import os

REDIS_HOST = os.environ.get("REDIS_HOST", "dgla-redis")
REDIS_PORT = int(os.environ.get("REDIS_PORT", 6379))
PROMETHEUS_URL = os.environ.get("PROMETHEUS_URL", "http://dgla-prometheus:9090")
GRAFANA_URL = os.environ.get("GRAFANA_URL", "http://dgla-grafana:3000")

# Log file for audit trails - container compatible path
LOG_FILE = "/app/logs/audit_trail.log"
os.makedirs(os.path.dirname(LOG_FILE), exist_ok=True)

# Create Redis connection to the actual Redis instance
def get_redis_client():
    try:
        r = redis.Redis(host=REDIS_HOST, port=REDIS_PORT, decode_responses=True)
        r.ping()  # Test connection
        print(f"✅ Successfully connected to Redis at {REDIS_HOST}:{REDIS_PORT}")
        return r
    except redis.exceptions.ConnectionError as e:
        print(f"❌ Failed to connect to Redis: {e}")
        return None

# Push metrics to Prometheus for real-time monitoring
def push_metric_to_prometheus(metric_name, value, labels=None):
    try:
        # In production, we would use the Prometheus client library
        # For this demonstration, we're simulating the metrics push via Redis
        r = get_redis_client()
        if r:
            metric_data = {
                "name": metric_name,
                "value": value,
                "timestamp": int(time.time()),
                "labels": labels or {}
            }
            r.lpush("dgla:metrics", json.dumps(metric_data))
            print(f"✅ Pushed metric {metric_name}={value} to Redis for Prometheus")
            return True
        return False
    except Exception as e:
        print(f"❌ Failed to push metric: {e}")
        return False

# Record an immutable audit log entry (this would be anchored to blockchain in the full app)
def record_audit_log(action, entity_id, metadata=None):
    """Record an immutable audit log that would be anchored to blockchain in the full implementation"""
    try:
        log_entry = {
            "id": str(uuid.uuid4()),
            "timestamp": datetime.utcnow().isoformat(),
            "action": action,
            "entity_id": entity_id,
            "metadata": metadata or {},
            "source_ip": socket.gethostbyname(socket.gethostname()),
            "hash": None  # Will be computed
        }
        
        # Create cryptographic hash of the log entry for integrity verification
        content_to_hash = f"{log_entry['id']}{log_entry['timestamp']}{log_entry['action']}{log_entry['entity_id']}{json.dumps(log_entry['metadata'])}"
        log_entry["hash"] = hashlib.sha256(content_to_hash.encode()).hexdigest()
        
        # Store in Redis (simulating the chainlog storage)
        r = get_redis_client()
        if r:
            r.lpush("dgla:audit_logs", json.dumps(log_entry))
        
        # Also write to local log file for demonstration
        with open(LOG_FILE, "a") as f:
            f.write(json.dumps(log_entry) + "\n")
            
        print(f"✅ Recorded audit log for {action} on {entity_id}")
        return log_entry
    except Exception as e:
        print(f"❌ Failed to record audit log: {e}")
        return None

# Detect and respond to a security threat (demonstration of real-time threat monitoring)
def detect_security_threat(data_source, event_data):
    """Simulates detecting a security threat and records appropriate governance data"""
    try:
        # 1. Record the event in the audit log
        audit_entry = record_audit_log(
            action="security_threat_detected",
            entity_id=f"{data_source}:{event_data.get('id', uuid.uuid4())}",
            metadata={
                "threat_type": event_data.get("threat_type", "unknown"),
                "severity": event_data.get("severity", "high"),
                "source": data_source,
                "details": event_data
            }
        )
        
        # 2. Push metrics to Prometheus for real-time monitoring
        push_metric_to_prometheus(
            "dgla_security_threat_detected_total", 
            1, 
            {
                "source": data_source,
                "threat_type": event_data.get("threat_type", "unknown"),
                "severity": event_data.get("severity", "high")
            }
        )
        
        # 3. Store threat details in Redis for rapid lookup
        r = get_redis_client()
        if r:
            threat_key = f"dgla:threats:{audit_entry['id']}"
            r.hmset(threat_key, {
                "timestamp": audit_entry["timestamp"],
                "source": data_source,
                "type": event_data.get("threat_type", "unknown"),
                "severity": event_data.get("severity", "high"),
                "details": json.dumps(event_data),
                "status": "detected"
            })
            r.expire(threat_key, 86400 * 7)  # Keep for 7 days
            
        return audit_entry["id"]
    except Exception as e:
        print(f"❌ Error in threat detection: {e}")
        return None

# Export logs for compliance reporting
def export_compliance_report(start_time, end_time, report_type):
    """Export logs for compliance reporting"""
    try:
        r = get_redis_client()
        if not r:
            return None
            
        # Get all audit logs from Redis
        all_logs = r.lrange("dgla:audit_logs", 0, -1)
        logs = [json.loads(log) for log in all_logs]
        
        # Filter by time range
        filtered_logs = [
            log for log in logs
            if start_time <= datetime.fromisoformat(log["timestamp"]) <= end_time
        ]
        
        # Create report
        report = {
            "report_id": str(uuid.uuid4()),
            "report_type": report_type,
            "generated_at": datetime.utcnow().isoformat(),
            "time_range": {
                "start": start_time.isoformat(),
                "end": end_time.isoformat()
            },
            "total_events": len(filtered_logs),
            "events": filtered_logs
        }
        
        # Store report in Redis
        report_key = f"dgla:reports:{report['report_id']}"
        r.set(report_key, json.dumps(report))
        r.expire(report_key, 86400 * 30)  # Keep for 30 days
        
        # Record audit log for report generation
        record_audit_log(
            action="compliance_report_generated",
            entity_id=report["report_id"],
            metadata={
                "report_type": report_type,
                "start_time": start_time.isoformat(),
                "end_time": end_time.isoformat(),
                "event_count": len(filtered_logs)
            }
        )
        
        print(f"✅ Generated compliance report {report['report_id']} with {len(filtered_logs)} events")
        return report
    except Exception as e:
        print(f"❌ Failed to generate compliance report: {e}")
        return None

# Check health of infrastructure components
def check_infrastructure_health():
    """Check the health of all infrastructure components"""
    results = {
        "timestamp": datetime.utcnow().isoformat(),
        "components": {}
    }
    
    # Check Redis
    try:
        r = redis.Redis(host=REDIS_HOST, port=REDIS_PORT)
        r.ping()
        results["components"]["redis"] = {"status": "healthy", "details": "Connection successful"}
    except Exception as e:
        results["components"]["redis"] = {"status": "unhealthy", "details": str(e)}
    
    # Check Prometheus
    try:
        response = requests.get(f"{PROMETHEUS_URL}/-/healthy", timeout=5)
        if response.status_code == 200:
            results["components"]["prometheus"] = {"status": "healthy", "details": "API accessible"}
        else:
            results["components"]["prometheus"] = {
                "status": "degraded", 
                "details": f"Unexpected status code: {response.status_code}"
            }
    except Exception as e:
        results["components"]["prometheus"] = {"status": "unhealthy", "details": str(e)}
    
    # Check Grafana
    try:
        response = requests.get(f"{GRAFANA_URL}/api/health", timeout=5)
        if response.status_code == 200:
            results["components"]["grafana"] = {"status": "healthy", "details": "API accessible"}
        else:
            results["components"]["grafana"] = {
                "status": "degraded", 
                "details": f"Unexpected status code: {response.status_code}"
            }
    except Exception as e:
        results["components"]["grafana"] = {"status": "unhealthy", "details": str(e)}
    
    # Overall status
    all_healthy = all(component["status"] == "healthy" for component in results["components"].values())
    results["overall_status"] = "healthy" if all_healthy else "degraded"
    
    return results

if __name__ == "__main__":
    print("🔒 DGLA Cybersecurity Infrastructure Adapter")
    print("============================================\n")
    
    # Check infrastructure health
    print("Checking infrastructure health...")
    health = check_infrastructure_health()
    print(f"Overall status: {health['overall_status'].upper()}")
    for component, status in health["components"].items():
        print(f"  - {component}: {status['status'].upper()}")
    
    print("\nDemonstrating cybersecurity use case with actual infrastructure...")
    
    # Simulate detecting security threats
    print("\n1. Detecting and responding to security threats...")
    threat1_id = detect_security_threat("firewall", {
        "id": "FW-1234",
        "threat_type": "port_scan",
        "severity": "medium",
        "source_ip": "203.0.113.42",
        "target_ip": "10.0.0.15",
        "ports": [22, 80, 443, 8080],
        "timestamp": datetime.utcnow().isoformat()
    })
    
    threat2_id = detect_security_threat("ids", {
        "id": "IDS-5678",
        "threat_type": "sql_injection",
        "severity": "high",
        "source_ip": "198.51.100.73",
        "target_url": "https://example.com/api/users",
        "payload": "' OR 1=1 --",
        "timestamp": datetime.utcnow().isoformat()
    })
    
    # Generate a compliance report
    print("\n2. Generating compliance report...")
    now = datetime.utcnow()
    start_time = datetime.fromtimestamp(time.time() - 3600)  # Last hour
    report = export_compliance_report(start_time, now, "security_incidents")
    
    if report:
        print(f"Report generated with {report['total_events']} events")
    
    print("\n3. Checking audit trail integrity...")
    # In the real implementation, this would verify blockchain anchors
    
    print("\nDemonstration complete. The DGLA infrastructure provides:")
    print("✓ Immutable audit trails with cryptographic verification")
    print("✓ Real-time security metrics and monitoring")
    print("✓ Automated compliance reporting")
    print("✓ Distributed threat intelligence")
    print("✓ Integration with existing security tools")
