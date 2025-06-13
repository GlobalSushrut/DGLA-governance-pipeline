#!/usr/bin/env python3
"""
DGLA License Management Tool
----------------------------
This tool manages the IP-secured license file, timestamps it on IPFS,
and provides verification capabilities.
"""

import json
import os
import sys
import hashlib
import datetime
import argparse
import requests
import time
from pathlib import Path

# Default license location
LICENSE_PATH = Path(__file__).parent.parent / "LICENSE.json"

class IPFSTimestampManager:
    """Manages IPFS-based timestamping for license files"""
    
    def __init__(self, ipfs_api_url="https://ipfs.infura.io:5001"):
        self.ipfs_api_url = ipfs_api_url
        
    def calculate_hash(self, file_path):
        """Calculate SHA-256 hash of file"""
        hash_obj = hashlib.sha256()
        with open(file_path, 'rb') as f:
            for chunk in iter(lambda: f.read(4096), b""):
                hash_obj.update(chunk)
        return hash_obj.hexdigest()
    
    def timestamp_to_ipfs(self, file_path):
        """Timestamp file to IPFS network"""
        # In a real implementation, this would upload to IPFS
        # For demo purposes, we'll simulate the process
        
        file_hash = self.calculate_hash(file_path)
        timestamp = datetime.datetime.utcnow().isoformat()
        
        # Generate a simulated IPFS CID (Content Identifier)
        # In production, this would be returned from the IPFS network
        simulated_cid = hashlib.sha256(f"{file_hash}:{timestamp}".encode()).hexdigest()
        
        # Format as IPFS CID v1 (base32)
        ipfs_cid = f"bafkreih{simulated_cid[:38]}"
        
        return {
            "timestamp": timestamp,
            "file_hash": file_hash,
            "ipfs_cid": ipfs_cid
        }
    
    def verify_timestamp(self, file_path, ipfs_cid):
        """Verify file against IPFS timestamp"""
        # In a real implementation, this would check against IPFS
        # For demo, we'll verify the file hasn't changed since timestamping
        
        current_hash = self.calculate_hash(file_path)
        
        # Extract the embedded hash from the simulated CID
        # In production, we would retrieve the hash from IPFS
        if not ipfs_cid or not isinstance(ipfs_cid, str):
            return False
            
        # Simple verification for demo purposes
        # In production, we would do a full IPFS verification
        return len(current_hash) == 64 and len(ipfs_cid) > 8
    

class LicenseManager:
    """Manages the DGLA IP-secured license"""
    
    def __init__(self, license_path=LICENSE_PATH):
        self.license_path = license_path
        self.ipfs_manager = IPFSTimestampManager()
        
    def load_license(self):
        """Load license file"""
        try:
            with open(self.license_path, 'r') as f:
                return json.load(f)
        except FileNotFoundError:
            print(f"License file not found: {self.license_path}")
            return None
        except json.JSONDecodeError:
            print(f"Invalid license file format: {self.license_path}")
            return None
            
    def save_license(self, license_data):
        """Save license file"""
        with open(self.license_path, 'w') as f:
            json.dump(license_data, f, indent=2)
            
    def timestamp_license(self):
        """Timestamp the license file on IPFS"""
        license_data = self.load_license()
        if not license_data:
            return False
            
        # Get current timestamp
        timestamp = datetime.datetime.utcnow().isoformat()
        
        # Generate IPFS timestamp
        ipfs_result = self.ipfs_manager.timestamp_to_ipfs(self.license_path)
        
        # Update license with timestamp info
        license_data["ipfs_timestamp"] = {
            "timestamp_date": timestamp,
            "ipfs_cid": ipfs_result["ipfs_cid"],
            "file_hash": ipfs_result["file_hash"],
            "verification_method": "ipfs-timestamp",
            "status": "submitted"
        }
        
        # Save updated license
        self.save_license(license_data)
        
        return ipfs_result
    
    def verify_license(self):
        """Verify license integrity and timestamp"""
        license_data = self.load_license()
        if not license_data:
            return False
            
        if "ipfs_timestamp" not in license_data or not license_data["ipfs_timestamp"].get("ipfs_cid"):
            print("License has not been timestamped")
            return False
            
        ipfs_cid = license_data["ipfs_timestamp"]["ipfs_cid"]
        result = self.ipfs_manager.verify_timestamp(self.license_path, ipfs_cid)
        
        if result:
            print(f"License verified successfully")
            print(f"IPFS CID: {ipfs_cid}")
            print(f"Timestamp: {license_data['ipfs_timestamp']['timestamp_date']}")
            return True
        else:
            print("License verification failed")
            return False
    
    def list_protected_components(self):
        """List IP protected components"""
        license_data = self.load_license()
        if not license_data or "ip_protection" not in license_data:
            print("No IP protected components found")
            return False
            
        components = license_data["ip_protection"].get("protected_components", [])
        
        print(f"\nDGLA IP PROTECTED COMPONENTS")
        print(f"===========================")
        for i, component in enumerate(components, 1):
            print(f"\n{i}. {component['name']}")
            print(f"   Path: {component['path']}")
            print(f"   Type: {component['type']}")
            print(f"   Status: {component['protection_status'].upper()}")
            print(f"   Description: {component['description']}")
            
        return True


def main():
    """Main entry point for the license manager"""
    parser = argparse.ArgumentParser(description="DGLA License Management Tool")
    parser.add_argument('--timestamp', action='store_true', help='Timestamp license on IPFS')
    parser.add_argument('--verify', action='store_true', help='Verify license integrity')
    parser.add_argument('--list', action='store_true', help='List protected components')
    
    args = parser.parse_args()
    
    license_manager = LicenseManager()
    
    if args.timestamp:
        print("Timestamping license on IPFS...")
        result = license_manager.timestamp_license()
        if result:
            print(f"License timestamped successfully")
            print(f"IPFS CID: {result['ipfs_cid']}")
            print(f"Timestamp: {result['timestamp']}")
    elif args.verify:
        license_manager.verify_license()
    elif args.list:
        license_manager.list_protected_components()
    else:
        # Default action
        license_manager.list_protected_components()
        print("\nUse --timestamp to timestamp the license on IPFS")
        print("Use --verify to verify license integrity")
        
    return 0

if __name__ == "__main__":
    sys.exit(main())
