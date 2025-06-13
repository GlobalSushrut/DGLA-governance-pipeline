# DGLA Troubleshooting Guide

## Connection Issues

### API Server Connectivity

**Problem**: Unable to connect to API server
- Check Kubernetes pod status: `kubectl get pods -n dgla`
- Verify API server logs: `kubectl logs <api-pod-name> -n dgla`
- Confirm NodePort accessibility: `curl http://<minikube-ip>:30081/system/status`

**Solution**:
```bash
# Restart API server if needed
kubectl rollout restart deployment api-server -n dgla

# Verify API server is running
kubectl get pods -n dgla | grep api-server
```

### Redis Backend Connectivity

**Problem**: API server can't connect to Redis
- Check Redis pod: `kubectl get pods -n dgla | grep redis`
- Verify Redis logs: `kubectl logs <redis-pod-name> -n dgla`
- Check Redis service: `kubectl get svc -n dgla | grep redis`

**Solution**:
```bash
# Check Redis connection with port-forward
kubectl port-forward svc/redis -n dgla 6379:6379
redis-cli -h localhost -a $(kubectl get secret redis -n dgla -o jsonpath="{.data.password}" | base64 -d)
```

## Authentication Issues

### Failed Login Attempts

**Problem**: Unable to authenticate with SDK
- Verify credentials are correct
- Check API server logs for authentication failures
- Confirm JWT secret is properly set in ConfigMap

**Solution**:
```python
# Reset credentials in SDK client
from dgla_sdk import DGLAClient
client = DGLAClient(
    api_url="http://minikube-ip:30081",
    username="admin",
    password="admin_password"
)
```

### Token Expiration

**Problem**: Authentication token expired
- SDK will automatically refresh tokens when possible
- Check token expiration in API server logs

**Solution**:
```python
# Force token refresh
client.auth.refresh_token()
```

## Audit Log Issues

### Chain Verification Failures

**Problem**: Audit chain verification failing
- Check for Redis data corruption
- Verify logs for tampering attempts
- Confirm appropriate access controls

**Solution**:
```python
# Verify specific entity chain
verification = client.chainlog.verify_chain(entity_id, entity_type)
print(f"Chain valid: {verification['verified']}")
if not verification['verified']:
    print(f"Error at position: {verification['error_position']}")
```

### Missing Logs

**Problem**: Expected logs not appearing
- Confirm logs were correctly appended
- Check for logging circuit breakers
- Verify search parameters are correct

**Solution**:
```python
# Search with broader parameters
logs = client.chainlog.get_logs(
    entity_id=entity_id,
    start_time=start_time - 86400,  # Check 24h before
    end_time=end_time + 86400       # Check 24h after
)
```

## Compliance Reports

### Report Generation Failures

**Problem**: Unable to generate compliance reports
- Verify entity exists with valid logs
- Check report type is supported
- Confirm date range contains data

**Solution**:
```python
# Test with known good parameters
report = client.export.generate_compliance_report(
    report_type="REPORT_GENERAL",
    start_time=int(time.time() - 604800),  # Last 7 days
    end_time=int(time.time()),
    format="pdf"
)
```

### Empty Reports

**Problem**: Reports generate but contain no data
- Verify logs exist for the entity
- Check date range is correct
- Confirm entity has required metadata

**Solution**:
```python
# Verify logs exist first
logs = client.chainlog.get_logs(entity_id=entity_id)
if logs:
    print(f"Found {len(logs)} logs, proceed with report")
else:
    print("No logs found for this entity")
```

## Kubernetes Deployment Issues

### Pod Crash Loop

**Problem**: Pods in CrashLoopBackOff
- Check pod logs: `kubectl logs <pod-name> -n dgla`
- Describe pod: `kubectl describe pod <pod-name> -n dgla`
- Verify ConfigMap and Secret values

**Solution**:
```bash
# View logs of crashing pod
kubectl logs <pod-name> -n dgla --previous

# Fix ConfigMap if needed
kubectl edit configmap api-config -n dgla
kubectl rollout restart deployment api-server -n dgla
```

### Volume Mount Issues

**Problem**: PersistentVolumeClaim issues
- Check PVC status: `kubectl get pvc -n dgla`
- Verify underlying storage
- Check pod events for mount errors

**Solution**:
```bash
# Recreate PVC if necessary
kubectl delete pvc redis-data -n dgla
kubectl apply -f redis-pvc.yaml
```

## Demo Execution Issues

### Demo Script Errors

**Problem**: `run_all_demos.sh` fails
- Check Minikube IP detection
- Verify API server endpoint
- Examine individual demo errors

**Solution**:
```bash
# Run with explicit IP
export MINIKUBE_IP=$(minikube ip)
export DGLA_API="http://$MINIKUBE_IP:30081"
./run_all_demos.sh
```

### Individual Demo Failures

**Problem**: Specific demo fails
- Examine Python error messages
- Check API server logs during execution
- Verify demo prerequisites

**Solution**:
```bash
# Run specific demo with debug output
PYTHONPATH=.. python -m demos.advanced.11_quantum_resistant_zk_authentication --verbose
```

## Performance Issues

### Slow API Responses

**Problem**: API calls taking too long
- Check Redis performance
- Verify API server resources
- Look for long-running transactions

**Solution**:
```bash
# Check API server resource usage
kubectl top pod -n dgla
```

### High Memory Usage

**Problem**: Memory consumption issues
- Check for Redis memory pressure
- Verify API server memory limits
- Look for memory leaks in logs

**Solution**:
```bash
# Adjust memory limits if needed
kubectl edit deployment api-server -n dgla
# Increase limits.memory and requests.memory
```

## Verification Issues

### Proof Creation Failures

**Problem**: Unable to create cryptographic proofs
- Check input data format
- Verify cryptographic libraries
- Examine detailed error messages

**Solution**:
```python
# Test simple verification
proof = client.verify.create_proof({"test": "data"})
print(f"Proof created with ID: {proof['id']}")
```

### Validation Failures

**Problem**: Proof validation failing
- Check proof ID is correct
- Verify proof hasn't been tampered with
- Confirm cryptographic parameters

**Solution**:
```python
# Validate with detailed output
validation = client.verify.validate_proof(proof_id, verbose=True)
print(f"Validation details: {validation}")
```

## Recovery Procedures

### Redis Data Recovery

**Problem**: Redis data corruption
- Use Redis persistence files
- Restore from backup if available
- Rebuild from application state

**Solution**:
```bash
# Check Redis persistence files
kubectl exec -it <redis-pod> -n dgla -- ls -l /data
```

### System Reset

**Problem**: Need to reset entire system
- Back up important data first
- Delete and recreate namespace
- Redeploy all components

**Solution**:
```bash
# Complete system reset
kubectl delete namespace dgla
kubectl create namespace dgla
# Reapply all deployment YAML files
```

## Contact Support

For issues not resolved by this guide, contact:

- Email: dgla-support@example.com
- Slack: #dgla-support
- File bug report: [GitHub Issues](https://github.com/example/dgla/issues)
