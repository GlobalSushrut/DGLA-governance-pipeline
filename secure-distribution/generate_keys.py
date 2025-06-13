#!/usr/bin/env python3
"""
DGLA Security Key Generator
Creates cryptographic key pairs for secure SDK distribution and verification
"""

import os
import json
import hashlib
import argparse
from datetime import datetime
from pathlib import Path
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa, padding
from cryptography.hazmat.backends import default_backend

def generate_key_pair(key_size=4096):
    """Generate an RSA key pair for signing and verification"""
    private_key = rsa.generate_private_key(
        public_exponent=65537,
        key_size=key_size,
        backend=default_backend()
    )
    
    public_key = private_key.public_key()
    
    return private_key, public_key

def save_keys(private_key, public_key, output_dir):
    """Save the keys to PEM files"""
    # Ensure output directory exists
    os.makedirs(output_dir, exist_ok=True)
    
    # Serialize private key with encryption
    private_pem = private_key.private_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PrivateFormat.PKCS8,
        encryption_algorithm=serialization.BestAvailableEncryption(b'strongpassword')  # Replace with actual password
    )
    
    # Serialize public key
    public_pem = public_key.public_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
    
    # Write keys to files
    with open(os.path.join(output_dir, "dgla_private.pem"), 'wb') as f:
        f.write(private_pem)
    
    with open(os.path.join(output_dir, "dgla_public.pem"), 'wb') as f:
        f.write(public_pem)
    
    print(f"Keys saved to {output_dir}")
    
    # Set secure permissions for private key
    os.chmod(os.path.join(output_dir, "dgla_private.pem"), 0o400)  # Read-only for owner

def generate_manifest(base_dir, private_key, output_file):
    """Generate a manifest of file hashes and sign it with the private key"""
    manifest = {
        "version": "1.0.0",
        "created": datetime.utcnow().isoformat(),
        "files": {}
    }
    
    # Calculate hashes for core files (cli and infrastructure)
    for directory in ["cli", "infrastructure"]:
        dir_path = os.path.join(base_dir, directory)
        if os.path.exists(dir_path):
            for root, _, files in os.walk(dir_path):
                for file in files:
                    if file.endswith(('.py', '.yaml', '.json', '.sh')):
                        file_path = os.path.join(root, file)
                        rel_path = os.path.relpath(file_path, base_dir)
                        
                        sha256_hash = hashlib.sha256()
                        with open(file_path, "rb") as f:
                            # Read file in chunks
                            for byte_block in iter(lambda: f.read(4096), b""):
                                sha256_hash.update(byte_block)
                        manifest["files"][rel_path] = sha256_hash.hexdigest()
                        
    # Sign the manifest
    manifest_copy = manifest.copy()
    manifest_content = json.dumps(manifest_copy, sort_keys=True).encode('utf-8')
    
    signature = private_key.sign(
        manifest_content,
        padding.PSS(
            mgf=padding.MGF1(hashes.SHA256()),
            salt_length=padding.PSS.MAX_LENGTH
        ),
        hashes.SHA256()
    )
    
    manifest["signature"] = signature.hex()
    
    # Write manifest to file
    with open(output_file, 'w') as f:
        json.dump(manifest, f, indent=2)
    
    print(f"Manifest generated with {len(manifest['files'])} files and saved to {output_file}")

def main():
    """Main function"""
    parser = argparse.ArgumentParser(description="Generate cryptographic keys for DGLA SDK")
    parser.add_argument("--key-size", type=int, default=4096, help="RSA key size (default: 4096)")
    parser.add_argument("--output-dir", default="./keys", help="Output directory for keys")
    parser.add_argument("--base-dir", default="..", help="Base directory for manifest generation")
    parser.add_argument("--manifest-output", default="./keys/manifest.json", help="Output file for manifest")
    
    args = parser.parse_args()
    
    print("Generating RSA key pair...")
    private_key, public_key = generate_key_pair(args.key_size)
    
    print("Saving keys...")
    save_keys(private_key, public_key, args.output_dir)
    
    print("Generating manifest...")
    generate_manifest(args.base_dir, private_key, args.manifest_output)
    
    print("Done!")

if __name__ == "__main__":
    main()
