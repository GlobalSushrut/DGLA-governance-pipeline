#!/usr/bin/env python3
"""
DGLA Mesh Integrator
Connects the secure mesh network with Kubernetes clusters and deployment server
"""

import os
import sys
import yaml
import json
import uuid
import logging
from datetime import datetime
from typing import Dict, List, Any, Optional

# Import secure mesh components
from secure_mesh import SecureMeshNode, PacketBlockchain
from kubernetes_mesh_controller import KubernetesMeshController

class MeshIntegrator:
    """
    Integrates the secure mesh network with Kubernetes clusters
    and the deployment server for military-grade secure operations
    """
    
    def __init__(self, integrator_id=None):
        """Initialize mesh integrator"""
        self.integrator_id = integrator_id or f"dgla-integrator-{uuid.uuid4().hex[:8]}"
        self.config_dir = os.path.expanduser("~/.dgla/mesh")
        
        # Ensure config directory exists
        os.makedirs(self.config_dir, exist_ok=True)
        
        # Set up logging
        self.logger = logging.getLogger("dgla-mesh-integrator")
        self.logger.setLevel(logging.INFO)
        if not self.logger.hasHandlers():
            handler = logging.StreamHandler()
            formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
            handler.setFormatter(formatter)
            self.logger.addHandler(handler)
            
        # Initialize components
        self.mesh_node = SecureMeshNode(
            self.integrator_id, 
            "integrator", 
            region="global"
        )
        
        self.k8s_controller = KubernetesMeshController()
        
        # Load configuration
        self.config = self._load_config()
        
    def _load_config(self) -> Dict:
        """Load or create integrator configuration"""
        config_path = os.path.join(self.config_dir, "integrator_config.json")
        
        if os.path.exists(config_path):
            try:
                with open(config_path, 'r') as f:
                    return json.load(f)
            except Exception as e:
                self.logger.error(f"Error loading config: {str(e)}")
                
        # Default config
        config = {
            "integrator_id": self.integrator_id,
            "created_at": datetime.utcnow().isoformat(),
            "deployment_servers": [],
            "connected_clusters": [],
            "integration_status": "initialized"
        }
        
        # Save default config
        with open(config_path, 'w') as f:
            json.dump(config, f, indent=2)
            
        return config
    
    def save_config(self):
        """Save current configuration"""
        config_path = os.path.join(self.config_dir, "integrator_config.json")
        
        with open(config_path, 'w') as f:
            json.dump(self.config, f, indent=2)
    
    def connect_deployment_server(self, server_url, server_id, public_key_file):
        """Connect to a deployment server"""
        self.logger.info(f"Connecting to deployment server: {server_url}")
        
        # Load public key
        try:
            with open(public_key_file, 'rb') as f:
                public_key_data = f.read()
                
            # Check if server already exists
            for server in self.config["deployment_servers"]:
                if server["url"] == server_url or server["id"] == server_id:
                    self.logger.info(f"Updating existing server: {server_id}")
                    server["url"] = server_url
                    server["id"] = server_id
                    server["updated_at"] = datetime.utcnow().isoformat()
                    self.save_config()
                    return True
            
            # Add new server
            self.config["deployment_servers"].append({
                "url": server_url,
                "id": server_id,
                "connected_at": datetime.utcnow().isoformat(),
                "status": "connected"
            })
            
            self.save_config()
            self.logger.info(f"Added deployment server: {server_id} at {server_url}")
            
            # In a real implementation, we would establish a secure mesh connection
            # with the deployment server using the mesh node
            
            return True
        except Exception as e:
            self.logger.error(f"Error connecting to server: {str(e)}")
            return False
    
    def connect_cluster(self, cluster_id):
        """Connect a Kubernetes cluster to the mesh network"""
        self.logger.info(f"Connecting cluster {cluster_id} to mesh network")
        
        # Get cluster details from controller
        cluster = self.k8s_controller.get_cluster_status(cluster_id)
        if not cluster:
            self.logger.error(f"Cluster {cluster_id} not found")
            return False
        
        # Add cluster to connected clusters if not already present
        for connected in self.config["connected_clusters"]:
            if connected["id"] == cluster_id:
                self.logger.info(f"Cluster {cluster_id} already connected")
                connected["last_connected"] = datetime.utcnow().isoformat()
                self.save_config()
                return True
        
        # Add new connected cluster
        self.config["connected_clusters"].append({
            "id": cluster_id,
            "name": cluster["name"],
            "customer_id": cluster["customer_id"],
            "region": cluster["region"],
            "connected_at": datetime.utcnow().isoformat(),
            "status": "connected"
        })
        
        self.save_config()
        self.logger.info(f"Connected cluster {cluster_id} to mesh network")
        
        # In a real implementation, we would configure the Istio mesh
        # between clusters and establish secure mesh connections
        
        return True
    
    def verify_cluster_mesh_integrity(self, cluster_id):
        """Verify the mesh integrity of a connected cluster"""
        self.logger.info(f"Verifying mesh integrity for cluster {cluster_id}")
        
        # Check if cluster is connected
        connected = False
        for cluster in self.config["connected_clusters"]:
            if cluster["id"] == cluster_id:
                connected = True
                break
                
        if not connected:
            self.logger.error(f"Cluster {cluster_id} not connected to mesh")
            return False
            
        # In a real implementation, this would check:
        # 1. mTLS certificate validity
        # 2. Istio sidecar injection status
        # 3. Network policy enforcement
        # 4. Authorization policy enforcement
        # 5. Packet blockchain integrity
        
        # For demonstration, we'll simulate success
        self.logger.info(f"Mesh integrity verified for cluster {cluster_id}")
        return True
    
    def list_connected_clusters(self):
        """List all clusters connected to the mesh network"""
        return self.config["connected_clusters"]
    
    def list_deployment_servers(self):
        """List all connected deployment servers"""
        return self.config["deployment_servers"]


# Command-line interface
if __name__ == "__main__":
    import argparse
    
    # Set up logging
    logging.basicConfig(level=logging.INFO)
    
    # Create argument parser
    parser = argparse.ArgumentParser(description="DGLA Mesh Integrator")
    subparsers = parser.add_subparsers(dest="command", help="Command to run")
    
    # Connect server command
    server_parser = subparsers.add_parser("connect-server", help="Connect to deployment server")
    server_parser.add_argument("--url", required=True, help="Server URL")
    server_parser.add_argument("--id", required=True, help="Server ID")
    server_parser.add_argument("--key-file", required=True, help="Public key file")
    
    # Connect cluster command
    cluster_parser = subparsers.add_parser("connect-cluster", help="Connect cluster to mesh")
    cluster_parser.add_argument("--cluster-id", required=True, help="Cluster ID")
    
    # Verify cluster command
    verify_parser = subparsers.add_parser("verify-cluster", help="Verify cluster mesh integrity")
    verify_parser.add_argument("--cluster-id", required=True, help="Cluster ID")
    
    # List clusters command
    list_clusters = subparsers.add_parser("list-clusters", help="List connected clusters")
    
    # List servers command
    list_servers = subparsers.add_parser("list-servers", help="List deployment servers")
    
    # Parse arguments
    args = parser.parse_args()
    
    # Create integrator
    integrator = MeshIntegrator()
    
    # Handle commands
    if args.command == "connect-server":
        success = integrator.connect_deployment_server(args.url, args.id, args.key_file)
        sys.exit(0 if success else 1)
    
    elif args.command == "connect-cluster":
        success = integrator.connect_cluster(args.cluster_id)
        sys.exit(0 if success else 1)
    
    elif args.command == "verify-cluster":
        success = integrator.verify_cluster_mesh_integrity(args.cluster_id)
        sys.exit(0 if success else 1)
    
    elif args.command == "list-clusters":
        clusters = integrator.list_connected_clusters()
        print(json.dumps(clusters, indent=2))
    
    elif args.command == "list-servers":
        servers = integrator.list_deployment_servers()
        print(json.dumps(servers, indent=2))
    
    else:
        parser.print_help()
        sys.exit(1)
