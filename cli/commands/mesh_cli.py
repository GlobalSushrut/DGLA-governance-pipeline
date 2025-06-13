#!/usr/bin/env python3
"""
DGLA Mesh Network CLI Module
Integrates secure mesh networking capabilities into the DGLA CLI
"""

import os
import sys
import json
import yaml
import logging
from datetime import datetime

# Add parent directory to path for local imports
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '../../')))

class MeshCLI:
    """Mesh Network CLI Integration"""
    
    def __init__(self, base_dir):
        """Initialize Mesh CLI"""
        self.base_dir = base_dir
        self.config_dir = os.path.join(base_dir, "config")
        self.mesh_dir = os.path.join(base_dir, "secure-distribution", "mesh")
        
        # Set up logging
        self.logger = logging.getLogger("dgla-mesh-cli")
        
    def setup_parser(self, subparsers):
        """Add mesh network commands to CLI parser"""
        
        mesh_parser = subparsers.add_parser('mesh', help='Secure mesh network operations')
        mesh_subparsers = mesh_parser.add_subparsers(dest='mesh_command')
        
        # Init mesh network
        init_parser = mesh_subparsers.add_parser('init', help='Initialize secure mesh network')
        init_parser.add_argument('--region', required=True, help='Region for this node')
        init_parser.add_argument('--role', choices=['client', 'server', 'controller'], 
                               required=True, help='Role of this node')
        init_parser.add_argument('--node-id', help='Custom node ID (generated if not provided)')
        
        # Connect to mesh network
        connect_parser = mesh_subparsers.add_parser('connect', help='Connect to mesh network server')
        connect_parser.add_argument('--server', required=True, help='Server URL to connect to')
        connect_parser.add_argument('--server-id', required=True, help='Server ID')
        connect_parser.add_argument('--key-file', required=True, help='Server public key file')
        
        # Deploy via mesh network
        deploy_parser = mesh_subparsers.add_parser('deploy', help='Deploy via secure mesh network')
        deploy_parser.add_argument('--server', required=True, help='Server URL to deploy to')
        deploy_parser.add_argument('--resource-file', help='Resource file to deploy')
        deploy_parser.add_argument('--resource-dir', help='Resource directory to deploy')
        deploy_parser.add_argument('--customer-id', help='Customer ID')
        deploy_parser.add_argument('--customer-name', help='Customer name')
        
        # Register customer cluster
        cluster_parser = mesh_subparsers.add_parser('register-cluster', 
                                                  help='Register customer Kubernetes cluster')
        cluster_parser.add_argument('--server', required=True, help='Server URL')
        cluster_parser.add_argument('--name', required=True, help='Cluster name')
        cluster_parser.add_argument('--customer-id', required=True, help='Customer ID')
        cluster_parser.add_argument('--region', required=True, help='Region for cluster')
        cluster_parser.add_argument('--nodes', type=int, default=3, help='Number of nodes')
        
        # List mesh connections
        list_parser = mesh_subparsers.add_parser('list-connections', 
                                               help='List mesh network connections')
        
        return mesh_parser
        
    def handle_command(self, args):
        """Handle mesh network CLI commands"""
        if not hasattr(args, 'mesh_command') or args.mesh_command is None:
            self.logger.error("No mesh command specified")
            return False
        
        # Import mesh functionality here to avoid circular imports
        sys.path.insert(0, os.path.abspath(self.mesh_dir))
        
        try:
            # Import mesh client
            from secure_distribution.mesh.dgla_mesh_client import MeshClient
        except ImportError:
            self.logger.error("Mesh client not found. Make sure secure-distribution/mesh is set up.")
            return False
        
        # Initialize mesh network
        if args.mesh_command == 'init':
            return self.init_mesh(args)
        
        # Connect to mesh network server
        elif args.mesh_command == 'connect':
            return self.connect_to_server(args)
        
        # Deploy via mesh network
        elif args.mesh_command == 'deploy':
            return self.deploy_via_mesh(args)
        
        # Register customer cluster
        elif args.mesh_command == 'register-cluster':
            return self.register_customer_cluster(args)
        
        # List mesh connections
        elif args.mesh_command == 'list-connections':
            return self.list_mesh_connections(args)
        
        else:
            self.logger.error(f"Unknown mesh command: {args.mesh_command}")
            return False
            
    def init_mesh(self, args):
        """Initialize secure mesh network"""
        self.logger.info(f"Initializing mesh network node with role: {args.role}")
        
        try:
            # Import here to avoid circular imports
            from secure_distribution.mesh.dgla_mesh_client import MeshClient
            
            # Create mesh client
            node_id = args.node_id or f"dgla-{args.role}-{os.getpid()}"
            client = MeshClient(client_id=node_id, region=args.region)
            
            print(f"Initialized mesh node:")
            print(f"  Node ID: {client.client_id}")
            print(f"  Role: {args.role}")
            print(f"  Region: {args.region}")
            print(f"  Key directory: {client.key_dir}")
            print(f"  Config directory: {client.config_dir}")
            
            return True
        except Exception as e:
            self.logger.error(f"Error initializing mesh: {str(e)}")
            return False

    def connect_to_server(self, args):
        """Connect to a mesh server"""
        self.logger.info(f"Connecting to mesh server: {args.server}")
        
        try:
            from secure_distribution.mesh.dgla_mesh_client import MeshClient
            client = MeshClient()
            success = client.add_server(args.server, args.server_id, args.key_file)
            
            if success:
                print(f"Successfully connected to server: {args.server}")
                return True
            else:
                return False
        except Exception as e:
            self.logger.error(f"Error connecting to server: {str(e)}")
            return False

    def deploy_via_mesh(self, args):
        """Deploy resources via mesh network"""
        if not args.resource_file and not args.resource_dir:
            self.logger.error("Either --resource-file or --resource-dir must be specified")
            return False
        
        self.logger.info(f"Deploying to mesh server: {args.server}")
        
        try:
            from secure_distribution.mesh.dgla_mesh_client import MeshClient
            client = MeshClient()
            
            if args.resource_file:
                result = client.deploy_agreement(
                    args.server, 
                    args.resource_file,
                    args.customer_id,
                    args.customer_name
                )
            else:
                result = client.deploy_logic(
                    args.server,
                    args.resource_dir,
                    args.customer_id,
                    args.customer_name
                )
            
            if result:
                print("Deployment successful!")
                print(json.dumps(result, indent=2))
                return True
            else:
                return False
        except Exception as e:
            self.logger.error(f"Error during deployment: {str(e)}")
            return False

    def register_customer_cluster(self, args):
        """Register a dedicated customer Kubernetes cluster"""
        self.logger.info(f"Registering customer cluster: {args.name} for {args.customer_id}")
        
        # This would interact with the mesh controller to create a new cluster
        # For demonstration, we'll show the concept
        print(f"Initiating registration for customer cluster:")
        print(f"  Customer: {args.customer_id}")
        print(f"  Cluster name: {args.name}")
        print(f"  Region: {args.region}")
        print(f"  Nodes: {args.nodes}")
        print("")
        print("In a production deployment, this would:")
        print("1. Create a dedicated Kubernetes cluster for the customer")
        print("2. Configure service mesh networking between clusters")
        print("3. Set up secure communication channels")
        print("4. Register the cluster in the main deployment ledger")
        print("")
        print("The customer would have complete control over their")
        print("dedicated cluster while still connected to the mesh network.")
        
        return True

    def list_mesh_connections(self, args):
        """List current mesh network connections"""
        try:
            from secure_distribution.mesh.dgla_mesh_client import MeshClient
            client = MeshClient()
            servers = client.list_servers()
            
            print("Current mesh network connections:")
            print(json.dumps(servers, indent=2))
            return True
        except Exception as e:
            self.logger.error(f"Error listing connections: {str(e)}")
            return False
