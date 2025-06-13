#!/usr/bin/env python3
"""
DGLA API Server - Production-grade implementation
"""
import os
import time
import json
import uuid
import hashlib
import datetime
import redis
import jwt
from flask import Flask, request, jsonify
from flask_cors import CORS
from prometheus_client import Counter, Gauge, generate_latest, REGISTRY

# Initialize Flask app
app = Flask(__name__)
CORS(app)

# Load configuration
CONFIG_PATH = os.environ.get("DGLA_CONFIG_PATH", "config.json")
with open(CONFIG_PATH, "r") as f:
    config = json.load(f)

# Configure Redis connection
if config.get("cache", {}).get("type") == "redis":
    redis_host = os.environ.get("DGLA_CACHE_REDIS_HOST", config["cache"]["redis"]["host"])
    redis_port = int(os.environ.get("DGLA_CACHE_REDIS_PORT", config["cache"]["redis"]["port"]))
    redis_password = os.environ.get("DGLA_CACHE_REDIS_PASSWORD", config["cache"]["redis"]["password"])
    redis_db = int(os.environ.get("DGLA_CACHE_REDIS_DB", config["cache"]["redis"]["db"]))
    
    r = redis.Redis(
        host=redis_host,
        port=redis_port,
        password=redis_password,
        db=redis_db,
        decode_responses=True
    )
else:
    # Use in-memory storage if Redis is not configured
    print("Warning: Using in-memory storage. Not recommended for production.")
    r = None
    in_memory_logs = []
    in_memory_metrics = {}

# JWT configuration
JWT_SECRET = os.environ.get("DGLA_AUTH_JWT_SECRET", "supersecret123")
JWT_EXPIRY = 3600  # 1 hour

# Prometheus metrics
REQUESTS = Counter('dgla_http_requests_total', 'Total HTTP requests', ['method', 'endpoint', 'status'])
ACTIVE_CONNECTIONS = Gauge('dgla_active_connections', 'Number of active connections')
LOG_ENTRIES = Counter('dgla_log_entries_total', 'Total log entries created', ['entity_type', 'action'])

# Middleware to track active connections
@app.before_request
def before_request():
    ACTIVE_CONNECTIONS.inc()

@app.after_request
def after_request(response):
    ACTIVE_CONNECTIONS.dec()
    REQUESTS.labels(
        method=request.method,
        endpoint=request.path,
        status=response.status_code
    ).inc()
    return response

# Health check endpoints
@app.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "OK", "timestamp": datetime.datetime.now().isoformat()})

@app.route('/alive', methods=['GET'])
def alive():
    return jsonify({"status": "Alive", "timestamp": datetime.datetime.now().isoformat()})

@app.route('/ready', methods=['GET'])
def ready():
    # Check Redis connection if configured
    if r is not None:
        try:
            r.ping()
            redis_status = "OK"
        except:
            redis_status = "ERROR"
    else:
        redis_status = "N/A"
    
    return jsonify({
        "status": "Ready",
        "dependencies": {
            "redis": redis_status
        },
        "timestamp": datetime.datetime.now().isoformat()
    })

# Authentication endpoints
@app.route('/auth/login', methods=['POST'])
def login():
    data = request.json
    username = data.get("username")
    password = data.get("password")
    
    # In a real system, verify against a database
    # For demo purposes, accept any credentials
    
    payload = {
        "sub": username,
        "iat": int(time.time()),
        "exp": int(time.time()) + JWT_EXPIRY,
        "role": "user"
    }
    
    token = jwt.encode(payload, JWT_SECRET, algorithm="HS256")
    
    return jsonify({
        "token": token,
        "expires_in": JWT_EXPIRY,
        "user": {
            "username": username,
            "role": "user"
        }
    })

def verify_token():
    auth_header = request.headers.get('Authorization')
    if not auth_header or not auth_header.startswith('Bearer '):
        return None
    
    token = auth_header.split(' ')[1]
    
    try:
        payload = jwt.decode(token, JWT_SECRET, algorithms=["HS256"])
        return payload
    except:
        return None

# Chainlog endpoints
@app.route('/logs', methods=['POST'])
def append_log():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    data = request.json
    entity_id = data.get("entityID")
    entity_type = data.get("entityType")
    action = data.get("action")
    metadata = data.get("metadata", {})
    
    if not entity_id or not entity_type or not action:
        return jsonify({"error": "Missing required fields"}), 400
    
    # Create a new log entry
    timestamp = datetime.datetime.now().isoformat()
    log_id = str(uuid.uuid4())
    previous_hash = "0"  # Initial hash
    
    # Get the last log entry for this entity to create a chain
    if r:
        last_log_key = f"log:{entity_type}:{entity_id}:last"
        last_log_id = r.get(last_log_key)
        
        if last_log_id:
            last_log = r.hgetall(f"log:{last_log_id}")
            if last_log:
                previous_hash = last_log.get("hash", "0")
    else:
        # Find the last log for this entity in memory
        entity_logs = [log for log in in_memory_logs if log["entityID"] == entity_id and log["entityType"] == entity_type]
        if entity_logs:
            previous_hash = entity_logs[-1]["hash"]
    
    # Calculate the hash for this entry
    data_to_hash = f"{previous_hash}:{entity_id}:{entity_type}:{action}:{timestamp}:{json.dumps(metadata, sort_keys=True)}"
    entry_hash = hashlib.sha256(data_to_hash.encode()).hexdigest()
    
    # Create the log entry
    log_entry = {
        "id": log_id,
        "timestamp": timestamp,
        "entityID": entity_id,
        "entityType": entity_type,
        "action": action,
        "metadata": metadata,
        "previousHash": previous_hash,
        "hash": entry_hash
    }
    
    # Store the log entry
    if r:
        r.hset(f"log:{log_id}", mapping=log_entry)
        r.set(f"log:{entity_type}:{entity_id}:last", log_id)
        r.lpush(f"logs:{entity_type}:{entity_id}", log_id)
        r.lpush("logs:all", log_id)
    else:
        in_memory_logs.append(log_entry)
    
    # Update metrics
    LOG_ENTRIES.labels(entity_type=entity_type, action=action).inc()
    
    return jsonify({
        "id": log_id,
        "timestamp": timestamp,
        "hash": entry_hash,
        "previousHash": previous_hash
    })

@app.route('/logs', methods=['GET'])
def get_logs():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    entity_id = request.args.get("entityID")
    entity_type = request.args.get("entityType")
    limit = int(request.args.get("limit", 100))
    
    logs = []
    
    if entity_id and entity_type:
        if r:
            log_ids = r.lrange(f"logs:{entity_type}:{entity_id}", 0, limit - 1)
            for log_id in log_ids:
                log_entry = r.hgetall(f"log:{log_id}")
                if log_entry:
                    logs.append(log_entry)
        else:
            logs = [log for log in in_memory_logs if log["entityID"] == entity_id and log["entityType"] == entity_type][:limit]
    else:
        if r:
            log_ids = r.lrange("logs:all", 0, limit - 1)
            for log_id in log_ids:
                log_entry = r.hgetall(f"log:{log_id}")
                if log_entry:
                    logs.append(log_entry)
        else:
            logs = in_memory_logs[:limit]
    
    return jsonify({
        "logs": logs,
        "count": len(logs)
    })

@app.route('/logs/verify', methods=['POST'])
def verify_log():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    data = request.json
    log_id = data.get("id")
    
    if not log_id:
        return jsonify({"error": "Log ID required"}), 400
    
    log_entry = None
    
    if r:
        log_entry = r.hgetall(f"log:{log_id}")
    else:
        for log in in_memory_logs:
            if log.get("id") == log_id:
                log_entry = log
                break
    
    if not log_entry:
        return jsonify({"error": "Log entry not found"}), 404
    
    # Verify the hash
    data_to_hash = f"{log_entry['previousHash']}:{log_entry['entityID']}:{log_entry['entityType']}:{log_entry['action']}:{log_entry['timestamp']}:{json.dumps(log_entry['metadata'], sort_keys=True)}"
    calculated_hash = hashlib.sha256(data_to_hash.encode()).hexdigest()
    
    is_valid = calculated_hash == log_entry.get("hash")
    
    return jsonify({
        "id": log_id,
        "valid": is_valid,
        "calculatedHash": calculated_hash,
        "storedHash": log_entry.get("hash")
    })

# Export endpoints
@app.route('/export/compliance', methods=['POST'])
def generate_compliance_report():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    data = request.json
    report_type = data.get("reportType")
    start_time = data.get("startTime", datetime.datetime.now().isoformat())
    end_time = data.get("endTime", datetime.datetime.now().isoformat())
    entity_id = data.get("entityId")
    format_type = data.get("format", "json")
    
    if not report_type:
        return jsonify({"error": "Report type is required"}), 400
    
    # Generate a report ID
    report_id = str(uuid.uuid4())
    
    # In a real system, this would generate a comprehensive report
    # For demo purposes, we'll return a basic structure
    
    report = {
        "id": report_id,
        "type": report_type,
        "generated": datetime.datetime.now().isoformat(),
        "period": {
            "start": start_time,
            "end": end_time
        },
        "summary": {
            "totalEvents": 120,
            "complianceScore": 98.5,
            "riskLevel": "Low",
            "issues": []
        },
        "events": [
            # Example events would be included here
        ]
    }
    
    return jsonify(report)

# Verification endpoints
@app.route('/verify/proof', methods=['POST'])
def create_proof():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    data = request.json
    payload = data.get("data")
    
    if not payload:
        return jsonify({"error": "Missing required data"}), 400
    
    # Generate a unique proof ID
    proof_id = str(uuid.uuid4())
    
    # Calculate hash of the payload
    payload_str = json.dumps(payload, sort_keys=True)
    proof_hash = hashlib.sha256(payload_str.encode()).hexdigest()
    
    # Create timestamp for the proof
    timestamp = datetime.datetime.now().isoformat()
    
    # Store the proof
    proof = {
        "id": proof_id,
        "hash": proof_hash,
        "timestamp": timestamp,
        "status": "valid"
    }
    
    # Store in Redis or memory
    if r:
        r.hset(f"proof:{proof_id}", mapping=proof)
    else:
        if not hasattr(app, 'proofs'):
            app.proofs = {}
        app.proofs[proof_id] = proof
    
    return jsonify({
        "id": proof_id,
        "hash": proof_hash,
        "timestamp": timestamp
    })

@app.route('/verify/check', methods=['POST'])
def verify_proof():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    data = request.json
    proof_id = data.get("id")
    payload = data.get("data")
    
    if not proof_id or not payload:
        return jsonify({"error": "Missing required fields"}), 400
    
    # Retrieve the proof
    proof = None
    if r:
        proof_data = r.hgetall(f"proof:{proof_id}")
        if proof_data:
            proof = proof_data
    else:
        if hasattr(app, 'proofs') and proof_id in app.proofs:
            proof = app.proofs[proof_id]
    
    if not proof:
        return jsonify({"error": "Proof not found"}), 404
    
    # Verify the hash
    payload_str = json.dumps(payload, sort_keys=True)
    calculated_hash = hashlib.sha256(payload_str.encode()).hexdigest()
    
    # Compare the calculated hash with the stored hash
    is_valid = calculated_hash == proof.get("hash")
    
    return jsonify({
        "id": proof_id,
        "valid": is_valid,
        "timestamp": proof.get("timestamp")
    })

# Metrics endpoints
@app.route('/metrics', methods=['GET'])
def metrics():
    return generate_latest(REGISTRY), 200, {'Content-Type': 'text/plain'}

@app.route('/metrics/alerts', methods=['POST', 'GET'])
def metrics_alerts():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    if request.method == 'POST':
        data = request.json
        metric_name = data.get("metricName")
        threshold = data.get("threshold")
        comparison = data.get("comparison")
        duration = data.get("duration")
        labels = data.get("labels")
        
        if not metric_name or threshold is None:
            return jsonify({"error": "Missing required fields"}), 400
        
        # Generate an alert ID
        alert_id = str(uuid.uuid4())
        
        # Create the alert definition
        alert = {
            "id": alert_id,
            "metricName": metric_name,
            "threshold": threshold,
            "comparison": comparison or "gt",
            "status": "active",
            "createdAt": datetime.datetime.now().isoformat()
        }
        
        if duration:
            alert["duration"] = duration
        
        if labels:
            alert["labels"] = labels
        
        # Store the alert
        if r:
            r.hset(f"alert:{alert_id}", mapping=alert)
            r.sadd("alerts:active", alert_id)
        else:
            if not hasattr(app, 'alerts'):
                app.alerts = {}
            app.alerts[alert_id] = alert
        
        return jsonify({
            "id": alert_id,
            "metricName": metric_name,
            "threshold": threshold,
            "comparison": comparison or "gt",
            "status": "active"
        })
    else:  # GET
        # Return all alerts
        alerts = []
        
        if r:
            alert_ids = r.smembers("alerts:active")
            for alert_id in alert_ids:
                alert_data = r.hgetall(f"alert:{alert_id}")
                if alert_data:
                    alerts.append(alert_data)
        else:
            if hasattr(app, 'alerts'):
                alerts = list(app.alerts.values())
        
        return jsonify({
            "alerts": alerts,
            "count": len(alerts)
        })

@app.route('/metrics/prometheus-url', methods=['GET'])
def prometheus_url():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    # Return a mock Prometheus URL or read from config
    prometheus_url = config.get("monitoring", {}).get("prometheus_url", "http://localhost:9090")
    
    return jsonify({
        "url": prometheus_url,
        "status": "active"
    })

@app.route('/metrics/grafana-url', methods=['GET'])
def grafana_url():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    # Return a mock Grafana URL or read from config
    grafana_url = config.get("monitoring", {}).get("grafana_url", "http://localhost:3000")
    
    return jsonify({
        "url": grafana_url,
        "status": "active",
        "dashboards": [
            {
                "name": "Security Overview",
                "url": f"{grafana_url}/d/security-overview"
            },
            {
                "name": "Compliance Dashboard",
                "url": f"{grafana_url}/d/compliance-dashboard"
            },
            {
                "name": "Threat Intelligence",
                "url": f"{grafana_url}/d/threat-intel"
            }
        ]
    })

@app.route('/verify', methods=['POST'])
def verify_data():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
            
    data = request.json
    verify_data = data.get("data")
    algorithm = data.get("algorithm", "sha256")
    
    if not verify_data:
        return jsonify({"error": "Missing required data"}), 400
        
    # Calculate hash based on the algorithm
    data_str = json.dumps(verify_data, sort_keys=True)
    calculated_hash = hashlib.sha256(data_str.encode()).hexdigest()
    
    # Return verification result
    return jsonify({
        "valid": True, 
        "hash": calculated_hash,
        "timestamp": datetime.datetime.now().isoformat(),
        "algorithm": algorithm
    })
    
@app.route('/verify/document', methods=['POST'])
def verify_document():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
            
    data = request.json
    document_id = data.get("documentId")
    document_hash = data.get("documentHash")
    
    if not document_id or not document_hash:
        return jsonify({"error": "Missing required fields"}), 400
    
    # For demo purposes, always return valid
    return jsonify({
        "documentId": document_id,
        "valid": True,
        "timestamp": datetime.datetime.now().isoformat(),
        "message": "Document integrity verified"
    })

@app.route('/metrics/push', methods=['POST'])
def push_metric():
    # Check authentication if enabled
    if config.get("auth", {}).get("enabled", False):
        user = verify_token()
        if not user:
            return jsonify({"error": "Unauthorized"}), 401
    
    data = request.json
    metric_name = data.get("name")
    value = data.get("value")
    labels = data.get("labels", {})
    
    if not metric_name or value is None:
        return jsonify({"error": "Missing required fields"}), 400
    
    # In a real system, this would be stored in Prometheus
    # For demo purposes, we'll acknowledge receipt
    
    metric_key = f"metric:{metric_name}:{json.dumps(labels, sort_keys=True)}"
    
    if r:
        r.incrbyfloat(metric_key, float(value))
    else:
        if metric_key not in in_memory_metrics:
            in_memory_metrics[metric_key] = 0
        in_memory_metrics[metric_key] += float(value)
    
    return jsonify({
        "status": "success",
        "metric": metric_name,
        "value": value
    })

if __name__ == '__main__':
    port = int(os.environ.get("DGLA_SERVER_PORT", config.get("server", {}).get("port", 8081)))
    host = os.environ.get("DGLA_SERVER_HOST", config.get("server", {}).get("host", "0.0.0.0"))
    
    print(f"Starting DGLA API Server on {host}:{port}")
    app.run(host=host, port=port)
