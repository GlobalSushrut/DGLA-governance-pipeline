#!/usr/bin/env python3
"""
Demo 7: IoT Security Monitoring Platform

This application demonstrates secure IoT device management with
real-time monitoring, firmware verification, and anomaly detection.

Features:
- IoT device registration with secure identities
- Firmware integrity verification
- Behavioral anomaly detection
- Secure update verification
"""
import os
import sys
import argparse
import time
import uuid
import json
import random
from datetime import datetime, timedelta

# Add parent directory to path to import the SDK
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from dgla_sdk import DGLAClient
from dgla_sdk.constants import HASH_SHA256

class IoTSecurityPlatform:
    """IoT security and management platform with advanced monitoring"""
    
    def __init__(self, api_url, api_key=None):
        """Initialize with DGLA client"""
        self.client = DGLAClient(base_url=api_url, api_key=api_key)
        # Simple in-memory stores for demo purposes
        self.devices = {}
        self.firmware = {}
        self.baselines = {}  # For anomaly detection
    
    def register_device(self, device_id, device_type, firmware_version, location=None):
        """Register a new IoT device with secure identity"""
        timestamp = datetime.now().isoformat()
        
        # Create device identity
        device = {
            'device_id': device_id,
            'device_type': device_type,
            'firmware_version': firmware_version,
            'location': location,
            'registration_date': timestamp,
            'last_seen': timestamp,
            'status': 'active'
        }
        
        self.devices[device_id] = device
        
        # Create baseline for this device type if not exists
        if device_type not in self.baselines:
            self.baselines[device_type] = {
                'avg_signal_strength': random.uniform(-70, -50),
                'avg_battery_level': random.uniform(90, 100),
                'avg_temperature': random.uniform(35, 45),
                'avg_uptime': random.uniform(99.5, 99.9),
                'packet_size_range': [20, 200],
                'typical_endpoints': ['mqtt-broker', 'ntp-server', 'update-server']
            }
        
        # Create cryptographic proof of device identity
        proof = self.client.verify.create_proof({
            'device_id': device_id,
            'device_type': device_type,
            'firmware_version': firmware_version,
            'timestamp': timestamp
        })
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=device_id,
            entity_type="iot_device",
            action="register",
            metadata={
                'device_type': device_type,
                'firmware_version': firmware_version,
                'location': location,
                'timestamp': timestamp,
                'proof_id': proof.get('id')
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="iot_devices_registered",
            value=1,
            labels={'device_type': device_type}
        )
        
        return {
            'device_id': device_id,
            'identity_proof': proof.get('id'),
            'registration_date': timestamp
        }
    
    def register_firmware(self, firmware_id, version, device_type, firmware_hash):
        """Register firmware with cryptographic verification"""
        timestamp = datetime.now().isoformat()
        
        # Store firmware information
        firmware = {
            'firmware_id': firmware_id,
            'version': version,
            'device_type': device_type,
            'hash': firmware_hash,
            'registration_date': timestamp
        }
        
        self.firmware[firmware_id] = firmware
        
        # Create cryptographic proof
        proof = self.client.verify.create_proof({
            'firmware_id': firmware_id,
            'version': version,
            'device_type': device_type,
            'hash': firmware_hash,
            'timestamp': timestamp
        })
        
        # Record in immutable audit log
        self.client.chainlog.append_log(
            entity_id=firmware_id,
            entity_type="firmware",
            action="register",
            metadata={
                'version': version,
                'device_type': device_type,
                'hash': firmware_hash,
                'timestamp': timestamp,
                'proof_id': proof.get('id')
            }
        )
        
        return {
            'firmware_id': firmware_id,
            'proof_id': proof.get('id'),
            'registration_date': timestamp
        }
    
    def verify_firmware(self, device_id, firmware_id, reported_hash):
        """Verify firmware integrity on a device"""
        if device_id not in self.devices:
            return {'error': 'Device not found'}
            
        if firmware_id not in self.firmware:
            return {'error': 'Firmware not found'}
        
        # Get device and firmware info
        device = self.devices[device_id]
        firmware = self.firmware[firmware_id]
        
        # Verify the hash matches
        is_verified = reported_hash == firmware['hash']
        verification_id = str(uuid.uuid4())
        timestamp = datetime.now().isoformat()
        
        # Record verification in immutable log
        self.client.chainlog.append_log(
            entity_id=verification_id,
            entity_type="firmware_verification",
            action="verify",
            metadata={
                'device_id': device_id,
                'firmware_id': firmware_id,
                'expected_hash': firmware['hash'],
                'reported_hash': reported_hash,
                'verified': is_verified,
                'timestamp': timestamp
            }
        )
        
        # Push metrics
        self.client.metrics.push_metric(
            metric_name="firmware_verifications",
            value=1,
            labels={
                'device_type': device['device_type'],
                'verified': str(is_verified).lower()
            }
        )
        
        return {
            'verification_id': verification_id,
            'device_id': device_id,
            'firmware_id': firmware_id,
            'verified': is_verified,
            'timestamp': timestamp
        }
    
    def process_telemetry(self, device_id, telemetry_data):
        """Process device telemetry and check for anomalies"""
        if device_id not in self.devices:
            return {'error': 'Device not found'}
            
        timestamp = datetime.now().isoformat()
        telemetry_id = str(uuid.uuid4())
        
        # Update device last seen
        device = self.devices[device_id]
        device['last_seen'] = timestamp
        
        # Get baseline for this device type
        baseline = self.baselines.get(device['device_type'], {})
        
        # Check for anomalies (simplified for demo)
        anomalies = []
        
        # Temperature anomaly detection
        if 'temperature' in telemetry_data:
            if abs(telemetry_data['temperature'] - baseline.get('avg_temperature', 40)) > 15:
                anomalies.append('temperature_anomaly')
        
        # Connection anomaly detection
        if 'connected_to' in telemetry_data:
            if telemetry_data['connected_to'] not in baseline.get('typical_endpoints', []):
                anomalies.append('connection_anomaly')
        
        # Battery anomaly detection
        if 'battery_level' in telemetry_data:
            if telemetry_data['battery_level'] < baseline.get('avg_battery_level', 90) * 0.5:
                anomalies.append('battery_anomaly')
        
        # Record telemetry in log if there are anomalies
        if anomalies:
            self.client.chainlog.append_log(
                entity_id=telemetry_id,
                entity_type="device_telemetry",
                action="anomaly_detected",
                metadata={
                    'device_id': device_id,
                    'device_type': device['device_type'],
                    'anomalies': anomalies,
                    'telemetry': telemetry_data,
                    'timestamp': timestamp
                }
            )
            
            # Push metrics for anomalies
            for anomaly in anomalies:
                self.client.metrics.push_metric(
                    metric_name="iot_anomalies",
                    value=1,
                    labels={
                        'device_type': device['device_type'],
                        'anomaly_type': anomaly
                    }
                )
        
        return {
            'telemetry_id': telemetry_id,
            'device_id': device_id,
            'timestamp': timestamp,
            'anomalies': anomalies,
            'status': 'alert' if anomalies else 'normal'
        }

def main():
    """Main function to demonstrate IoT security platform"""
    parser = argparse.ArgumentParser(description="IoT Security Monitoring Demo")
    parser.add_argument("--api-url", default="http://localhost:8080", help="DGLA API URL")
    parser.add_argument("--api-key", default=None, help="DGLA API key")
    args = parser.parse_args()
    
    # Initialize IoT security platform
    iot_platform = IoTSecurityPlatform(api_url=args.api_url, api_key=args.api_key)
    
    # Demo workflow
    print("🔌 DGLA IoT Security Monitoring Demo")
    print("===================================")
    
    # 1. Register IoT devices
    print("\n1. Registering IoT devices...")
    devices = [
        {"id": "thermostat-0a1b2c", "type": "smart_thermostat", "firmware": "v2.1.0", "location": "Building A"},
        {"id": "camera-3d4e5f", "type": "security_camera", "firmware": "v1.5.2", "location": "Building A"},
        {"id": "gateway-6g7h8i", "type": "edge_gateway", "firmware": "v3.0.1", "location": "Building A"}
    ]
    
    for device in devices:
        result = iot_platform.register_device(
            device_id=device["id"],
            device_type=device["type"],
            firmware_version=device["firmware"],
            location=device["location"]
        )
        print(f"  ✓ Device registered: {device['id']}")
    
    # 2. Register and verify firmware
    print("\n2. Registering and verifying firmware...")
    
    # Register firmware
    firmware_id = "thermostat-fw-v2.1.0"
    firmware_hash = "8a7b6c5d4e3f2g1h0i9j8k7l6m5n4o3p2q1r"
    
    firmware = iot_platform.register_firmware(
        firmware_id=firmware_id,
        version="v2.1.0",
        device_type="smart_thermostat",
        firmware_hash=firmware_hash
    )
    print(f"  ✓ Firmware registered: {firmware_id}")
    
    # Verify firmware on device
    verification = iot_platform.verify_firmware(
        device_id="thermostat-0a1b2c",
        firmware_id=firmware_id,
        reported_hash=firmware_hash  # Matching hash
    )
    print(f"  ✓ Firmware verification: {'Passed' if verification['verified'] else 'Failed'}")
    
    # Try with incorrect hash
    bad_verification = iot_platform.verify_firmware(
        device_id="thermostat-0a1b2c",
        firmware_id=firmware_id,
        reported_hash="tampered_hash_value"  # Non-matching hash
    )
    print(f"  ✓ Tampered firmware verification: {'Passed' if bad_verification['verified'] else 'Failed (as expected)'}")
    
    # 3. Process device telemetry with anomaly detection
    print("\n3. Processing device telemetry with anomaly detection...")
    
    # Normal telemetry
    normal_telemetry = {
        "temperature": 42,
        "battery_level": 95,
        "signal_strength": -65,
        "connected_to": "mqtt-broker"
    }
    
    normal_result = iot_platform.process_telemetry(
        device_id="thermostat-0a1b2c",
        telemetry_data=normal_telemetry
    )
    print(f"  ✓ Normal telemetry status: {normal_result['status']}")
    
    # Anomalous telemetry
    anomalous_telemetry = {
        "temperature": 85,  # Very high temperature
        "battery_level": 25,  # Unusually low battery
        "signal_strength": -95,  # Poor signal
        "connected_to": "unknown-server"  # Suspicious connection
    }
    
    anomaly_result = iot_platform.process_telemetry(
        device_id="thermostat-0a1b2c",
        telemetry_data=anomalous_telemetry
    )
    print(f"  ✓ Anomalous telemetry status: {anomaly_result['status']}")
    if anomaly_result['anomalies']:
        print(f"  ✓ Detected anomalies: {', '.join(anomaly_result['anomalies'])}")
    
    print("\nAll IoT device activities and security events have been")
    print("cryptographically verified and recorded in the immutable audit log.")

if __name__ == "__main__":
    main()
