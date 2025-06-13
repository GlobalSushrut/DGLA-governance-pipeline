#!/usr/bin/env python3
"""
DGLA Blockchain-like Deployment Server - Core Module
Handles secure immutable deployments with cryptographic verification
"""

import os
import json
import uuid
import hashlib
import logging
import yaml
import toml
from datetime import datetime
from flask import Flask, request, jsonify

# Constants
VERSION = "1.0.0"
DEPLOYMENTS_DIR = "/var/lib/dgla/deployments"
LEDGER_PATH = "/var/lib/dgla/ledger.toml"

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("dgla-server")

class DglaSecureChain:
    """Manages blockchain-like immutable deployment ledger"""
    
    def __init__(self, ledger_path=LEDGER_PATH):
        self.ledger_path = ledger_path
        self.ledger = self._load_ledger()
    
    def _load_ledger(self):
        """Load the deployment ledger"""
        if os.path.exists(self.ledger_path):
            return toml.load(self.ledger_path)
            
        # Initialize new ledger
        return {
            "version": VERSION,
            "chain_id": str(uuid.uuid4()),
            "created": datetime.utcnow().isoformat(),
            "last_block_hash": "",
            "blocks": []
        }
    
    def _save_ledger(self):
        """Save the ledger to disk"""
        os.makedirs(os.path.dirname(self.ledger_path), exist_ok=True)
        with open(self.ledger_path, 'w') as f:
            toml.dump(self.ledger, f)
    
    def add_block(self, action, data):
        """Add a new block to the chain"""
        # Get previous hash
        prev_hash = self.ledger["last_block_hash"]
        if not prev_hash and self.ledger["blocks"]:
            prev_hash = self.ledger["blocks"][-1]["hash"]
        
        # Create block
        timestamp = datetime.utcnow().isoformat()
        block = {
            "index": len(self.ledger["blocks"]),
            "timestamp": timestamp,
            "action": action,
            "data": data,
            "previous_hash": prev_hash or "0"
        }
        
        # Calculate block hash
        block_json = json.dumps(block, sort_keys=True)
        block_hash = hashlib.sha256(block_json.encode()).hexdigest()
        block["hash"] = block_hash
        
        # Add to chain
        self.ledger["blocks"].append(block)
        self.ledger["last_block_hash"] = block_hash
        self.ledger["last_updated"] = timestamp
        
        # Save to disk
        self._save_ledger()
        
        return block
    
    def verify_chain(self):
        """Verify the integrity of the blockchain"""
        if not self.ledger["blocks"]:
            return True
            
        for i, block in enumerate(self.ledger["blocks"]):
            # Skip genesis block
            if i == 0:
                continue
                
            # Check hash links
            if block["previous_hash"] != self.ledger["blocks"][i-1]["hash"]:
                logger.error(f"Invalid chain at block {i}: hash mismatch")
                return False
                
            # Verify block hash
            block_copy = block.copy()
            stored_hash = block_copy.pop("hash")
            block_json = json.dumps(block_copy, sort_keys=True)
            calculated_hash = hashlib.sha256(block_json.encode()).hexdigest()
            
            if calculated_hash != stored_hash:
                logger.error(f"Invalid block hash at index {i}")
                return False
                
        return True


class DeploymentManager:
    """Manages secure deployments to Kubernetes"""
    
    def __init__(self, chain, base_dir=DEPLOYMENTS_DIR):
        self.chain = chain
        self.base_dir = base_dir
        os.makedirs(self.base_dir, exist_ok=True)
    
    def create_deployment(self, customer_id, resources):
        """Create a new deployment"""
        # Generate deployment ID
        deployment_id = str(uuid.uuid4())
        timestamp = datetime.utcnow().isoformat()
        
        # Create deployment directory
        deploy_dir = os.path.join(self.base_dir, deployment_id)
        os.makedirs(deploy_dir, exist_ok=True)
        
        # Save resources
        resources_file = os.path.join(deploy_dir, "resources.yaml")
        with open(resources_file, 'w') as f:
            yaml.dump(resources, f)
        
        # Record deployment metadata
        metadata = {
            "id": deployment_id,
            "customer_id": customer_id,
            "timestamp": timestamp,
            "resource_count": len(resources) if isinstance(resources, list) else 1,
            "status": "created"
        }
        
        metadata_file = os.path.join(deploy_dir, "metadata.json")
        with open(metadata_file, 'w') as f:
            json.dump(metadata, f, indent=2)
        
        # Add to blockchain
        self.chain.add_block("deployment_created", metadata)
        
        return metadata


# Initialize Flask application
def create_app():
    """Create and configure Flask application"""
    app = Flask(__name__)
    
    # Initialize components
    chain = DglaSecureChain()
    deployment_mgr = DeploymentManager(chain)
    
    # Store in app context
    app.chain = chain
    app.deployment_mgr = deployment_mgr
    
    # Routes
    @app.route('/api/v1/health', methods=['GET'])
    def health_check():
        """Health check endpoint"""
        return jsonify({
            "status": "running",
            "version": VERSION,
            "chain_verified": chain.verify_chain()
        })
    
    @app.route('/api/v1/deployments', methods=['POST'])
    def create_deployment():
        """Create a new deployment"""
        if not request.is_json:
            return jsonify({"error": "Request must be JSON"}), 400
        
        data = request.json
        if 'customer_id' not in data or 'resources' not in data:
            return jsonify({"error": "Missing required fields"}), 400
        
        # Create deployment
        result = deployment_mgr.create_deployment(
            data['customer_id'],
            data['resources']
        )
        
        return jsonify(result), 201
    
    @app.route('/api/v1/verify', methods=['GET'])
    def verify_chain():
        """Verify the blockchain integrity"""
        is_valid = chain.verify_chain()
        return jsonify({
            "valid": is_valid,
            "block_count": len(chain.ledger["blocks"])
        })
    
    return app

# Application entry point
if __name__ == "__main__":
    app = create_app()
    app.run(host="0.0.0.0", port=8080)
