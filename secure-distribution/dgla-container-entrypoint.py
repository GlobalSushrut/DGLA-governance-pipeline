#!/usr/bin/env python3
"""
DGLA Secure Container Entrypoint
Provides integrity verification and secure execution of the DGLA SDK
Following blockchain-like security model for immutable infrastructure
"""

import os
import sys
import json
import hmac
import hashlib
import argparse
import subprocess
import toml
from datetime import datetime
from pathlib import Path
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import padding, rsa
from cryptography.hazmat.primitives.serialization import load_pem_public_key
from cryptography.exceptions import InvalidSignature

# Constants
VERSION = "1.0.0"
MANIFEST_PATH = "/app/keys/manifest.json"
PUBLIC_KEY_PATH = "/app/keys/dgla_public.pem"
LOCK_FILE_PATH = "/app/.dgla/lock.toml"

class DglaSecureContainer:
    """Manages the secure execution environment for DGLA SDK"""
    
    def __init__(self):
        """Initialize secure container environment"""
        self.sdk_path = "/app"
        self.user_extensions_path = "/app/user-extensions"
        self.deployments_path = "/app/deployments"
        self.config_path = "/app/.dgla"
        
        # Ensure required directories exist
        os.makedirs(self.config_path, exist_ok=True)
        os.makedirs(self.user_extensions_path, exist_ok=True)
        os.makedirs(self.deployments_path, exist_ok=True)
        
        # Load manifest if it exists
        self.manifest = self._load_manifest()
        
        # Debug mode
        self.debug = os.environ.get('DGLA_DEBUG', '').lower() == 'true'
    
    def _load_manifest(self):
        """Load the signed manifest containing file hashes"""
        try:
            if os.path.exists(MANIFEST_PATH):
                with open(MANIFEST_PATH, 'r') as f:
                    return json.load(f)
            return {
                "version": VERSION,
                "created": datetime.utcnow().isoformat(),
                "files": {},
                "signature": None
            }
        except Exception as e:
            print(f"Error loading manifest: {e}")
            return None
    
    def _log(self, message, level="INFO"):
        """Log message with timestamp"""
        if self.debug or level != "DEBUG":
            timestamp = datetime.utcnow().isoformat()
            print(f"[{timestamp}] {level}: {message}")
    
    def verify_integrity(self, strict=True):
        """
        Verify the integrity of core SDK files
        Uses cryptographic verification similar to blockchain validators
        """
        self._log("Verifying SDK integrity...")
        
        if not self.manifest:
            if strict:
                self._log("Manifest not found or invalid", "ERROR")
                return False
            return True
            
        try:
            # Load public key
            if not os.path.exists(PUBLIC_KEY_PATH):
                if strict:
                    self._log("Public key not found", "ERROR")
                    return False
                return True
                
            with open(PUBLIC_KEY_PATH, 'rb') as key_file:
                public_key = load_pem_public_key(key_file.read())
                
            # Verify manifest signature
            if 'signature' in self.manifest and self.manifest['signature']:
                # Extract contents to verify and signature
                manifest_copy = self.manifest.copy()
                signature_bytes = bytes.fromhex(manifest_copy.pop('signature'))
                manifest_content = json.dumps(manifest_copy, sort_keys=True).encode('utf-8')
                
                try:
                    # Verify signature
                    public_key.verify(
                        signature_bytes,
                        manifest_content,
                        padding.PSS(
                            mgf=padding.MGF1(hashes.SHA256()),
                            salt_length=padding.PSS.MAX_LENGTH
                        ),
                        hashes.SHA256()
                    )
                    self._log("Manifest signature verified", "INFO")
                except InvalidSignature:
                    self._log("Invalid manifest signature", "ERROR")
                    return False
                    
            # Verify core files integrity
            if 'files' in self.manifest:
                for file_path, expected_hash in self.manifest['files'].items():
                    full_path = os.path.join(self.sdk_path, file_path)
                    if os.path.exists(full_path) and os.path.isfile(full_path):
                        sha256_hash = hashlib.sha256()
                        with open(full_path, "rb") as f:
                            # Read file in chunks
                            for byte_block in iter(lambda: f.read(4096), b""):
                                sha256_hash.update(byte_block)
                        actual_hash = sha256_hash.hexdigest()
                        
                        if actual_hash != expected_hash:
                            self._log(f"File integrity check failed: {file_path}", "ERROR")
                            if strict:
                                return False
                    elif strict:
                        self._log(f"Required file not found: {file_path}", "ERROR")
                        return False
            
            self._log("SDK integrity verification completed successfully", "INFO")
            return True
            
        except Exception as e:
            self._log(f"Error during integrity verification: {e}", "ERROR")
            if strict:
                return False
            return True
    
    def load_lock_file(self):
        """Load deployment lock file (similar to blockchain state)"""
        try:
            if os.path.exists(LOCK_FILE_PATH):
                return toml.load(LOCK_FILE_PATH)
            return {
                "version": VERSION,
                "last_updated": datetime.utcnow().isoformat(),
                "deployments": {},
                "signed_configs": {}
            }
        except Exception as e:
            self._log(f"Error loading lock file: {e}", "ERROR")
            return None
            
    def update_lock_file(self, lock_data):
        """Update the lock file with new deployment information"""
        try:
            # Update last_updated timestamp
            lock_data["last_updated"] = datetime.utcnow().isoformat()
            
            # Write lock file
            with open(LOCK_FILE_PATH, 'w') as f:
                toml.dump(lock_data, f)
            
            # Set permissions to read-only
            os.chmod(LOCK_FILE_PATH, 0o400)  # Read-only
            
            return True
        except Exception as e:
            self._log(f"Error updating lock file: {e}", "ERROR")
            return False
    
    def execute_cli(self, args):
        """Execute DGLA CLI commands in the secure container"""
        if not self.verify_integrity(strict=False):
            self._log("SDK integrity check failed, proceeding with caution", "WARNING")
        
        # Delegate to the actual CLI
        cli_path = os.path.join(self.sdk_path, "cli", "dgla.py")
        cmd = [sys.executable, cli_path] + args
        
        self._log(f"Executing command: {' '.join(cmd)}", "DEBUG")
        return subprocess.call(cmd)
        
    def main(self):
        """Main entrypoint for the secure container"""
        # Parse command line arguments
        parser = argparse.ArgumentParser(
            description="DGLA Secure Container"
        )
        parser.add_argument("--check-only", action="store_true", 
                           help="Only check integrity without running commands")
        
        # Parse known args to handle both container flags and pass the rest to the CLI
        container_args, cli_args = parser.parse_known_args()
        
        # Check integrity
        if not self.verify_integrity(strict=False):
            self._log("SDK integrity verification failed", "WARNING")
            if container_args.check_only:
                return 1
                
        if container_args.check_only:
            self._log("Integrity check completed")
            return 0
            
        # Execute CLI with remaining arguments
        return self.execute_cli(cli_args)
        
if __name__ == "__main__":
    container = DglaSecureContainer()
    sys.exit(container.main())
