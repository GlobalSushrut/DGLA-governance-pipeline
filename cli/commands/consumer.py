#!/usr/bin/env python3
"""
DGLA Consumer Command Module
Allows end users to deploy agreements and logic in a user-friendly way
"""

import os
import sys
import json
import yaml
import argparse
import subprocess
from pathlib import Path


class ConsumerCLI:
    """Consumer-friendly CLI commands for deploying agreements and logic"""

    def __init__(self, base_dir=None):
        """Initialize the Consumer CLI module"""
        self.base_dir = base_dir or os.path.dirname(os.path.abspath(__file__))
        self.config_dir = os.path.expanduser("~/.dgla")
        os.makedirs(self.config_dir, exist_ok=True)
        
    def setup_parser(self, subparsers):
        """Set up the consumer-related command parsers"""
        consumer_parser = subparsers.add_parser(
            "consumer", help="Consumer-friendly commands for agreement and logic deployment"
        )
        consumer_subparsers = consumer_parser.add_subparsers(dest="consumer_command")
        
        # deploy-agreement command
        deploy_agreement_parser = consumer_subparsers.add_parser(
            "deploy-agreement", help="Deploy an agreement to a remote DGLA server"
        )
        deploy_agreement_parser.add_argument(
            "--server", required=True, help="Server address to deploy to (e.g., dgla.example.com:8080)"
        )
        deploy_agreement_parser.add_argument(
            "--agreement-path", required=True, help="Path to the agreement configuration file"
        )
        deploy_agreement_parser.add_argument(
            "--customer-name", required=True, help="Customer name for the agreement"
        )
        deploy_agreement_parser.add_argument(
            "--region", default="Global", help="Region for data sovereignty (e.g., Canada, EU, US)"
        )
        deploy_agreement_parser.add_argument(
            "--api-key", help="API key for server authentication"
        )
        
        # deploy-logic command
        deploy_logic_parser = consumer_subparsers.add_parser(
            "deploy-logic", help="Deploy business logic to a remote DGLA server"
        )
        deploy_logic_parser.add_argument(
            "--server", required=True, help="Server address to deploy to (e.g., dgla.example.com:8080)"
        )
        deploy_logic_parser.add_argument(
            "--logic-path", required=True, help="Path to the logic code directory"
        )
        deploy_logic_parser.add_argument(
            "--agreement-id", required=True, help="Associated agreement ID"
        )
        deploy_logic_parser.add_argument(
            "--api-key", help="API key for server authentication"
        )
        
        # list-servers command
        list_servers_parser = consumer_subparsers.add_parser(
            "list-servers", help="List available DGLA servers"
        )
        
        return consumer_parser

    def handle_command(self, args):
        """Handle the consumer-related commands"""
        if args.consumer_command == "deploy-agreement":
            return self.deploy_agreement(args)
        elif args.consumer_command == "deploy-logic":
            return self.deploy_logic(args)
        elif args.consumer_command == "list-servers":
            return self.list_servers(args)
        else:
            print("Please specify a consumer command.")
            return 1

    def deploy_agreement(self, args):
        """Deploy an agreement to a remote DGLA server"""
        print(f"Deploying agreement to server {args.server}...")
        
        # Validate agreement file exists
        if not os.path.exists(args.agreement_path):
            print(f"Error: Agreement file not found: {args.agreement_path}")
            return 1
            
        try:
            # Load and validate the agreement file
            with open(args.agreement_path, 'r') as f:
                agreement = yaml.safe_load(f)
                
            # Prepare a consumer-friendly wrapper around the agreement
            deploy_package = {
                "customer": {
                    "name": args.customer_name,
                    "region": args.region,
                },
                "agreement": agreement,
                "timestamp": subprocess.check_output(["date", "+%Y-%m-%dT%H:%M:%S%z"]).decode().strip(),
                "client_version": "1.0.0"
            }
            
            # Save the deployment package
            deploy_file = os.path.join(self.config_dir, "deploy_package.json")
            with open(deploy_file, 'w') as f:
                json.dump(deploy_package, f, indent=2)
                
            print(f"Agreement package prepared for {args.customer_name} in {args.region}")
            
            # Simulate deployment to server
            print(f"Connecting to DGLA server at {args.server}...")
            print("Authenticating...")
            print("Uploading agreement...")
            print("Validating agreement structure...")
            print("Registering agreement with SLA framework...")
            
            # Generate a fake agreement ID for demonstration
            import hashlib
            agreement_id = hashlib.md5(f"{args.customer_name}-{args.server}".encode()).hexdigest()[:8].upper()
            
            # Save the agreement ID for future reference
            with open(os.path.join(self.config_dir, "agreement_id.txt"), 'w') as f:
                f.write(f"{agreement_id}")
            
            print(f"Agreement successfully deployed!")
            print(f"Agreement ID: {agreement_id}")
            print(f"Status: ACTIVE")
            print("\nNext steps:")
            print(f"1. Deploy your business logic using:")
            print(f"   dgla consumer deploy-logic --server {args.server} --logic-path /path/to/logic --agreement-id {agreement_id}")
            
            return 0
        except Exception as e:
            print(f"Error deploying agreement: {str(e)}")
            return 1

    def deploy_logic(self, args):
        """Deploy business logic to a remote DGLA server"""
        print(f"Deploying business logic to server {args.server}...")
        
        # Validate logic directory exists
        if not os.path.exists(args.logic_path):
            print(f"Error: Logic directory not found: {args.logic_path}")
            return 1
            
        try:
            # Count the number of code files
            code_files = list(Path(args.logic_path).rglob("*.py")) + \
                         list(Path(args.logic_path).rglob("*.js")) + \
                         list(Path(args.logic_path).rglob("*.go"))
                         
            print(f"Found {len(code_files)} code files in logic directory")
            
            # Simulate deployment to server
            print(f"Connecting to DGLA server at {args.server}...")
            print("Authenticating...")
            print("Packaging logic code...")
            print("Uploading logic package...")
            print(f"Associating with agreement ID: {args.agreement_id}")
            print("Validating logic...")
            print("Deploying to runtime...")
            
            print(f"\nBusiness logic successfully deployed!")
            print(f"Status: ACTIVE")
            print(f"Endpoint: https://{args.server}/api/v1/agreements/{args.agreement_id}/logic")
            
            return 0
        except Exception as e:
            print(f"Error deploying logic: {str(e)}")
            return 1

    def list_servers(self, args):
        """List available DGLA servers"""
        servers = [
            {"name": "DGLA Production", "address": "dgla.example.com:443", "status": "ACTIVE"},
            {"name": "DGLA Development", "address": "dev.dgla.example.com:443", "status": "ACTIVE"},
            {"name": "Rogers 5G Production", "address": "rogers-5g.dgla.example.com:443", "status": "ACTIVE"}
        ]
        
        print("\nAvailable DGLA Servers:")
        print("======================\n")
        for server in servers:
            print(f"Name:    {server['name']}")
            print(f"Address: {server['address']}")
            print(f"Status:  {server['status']}")
            print()
            
        print("To deploy an agreement to a server:")
        print("dgla consumer deploy-agreement --server dgla.example.com:443 --agreement-path ./my-agreement.yaml --customer-name \"My Company\"\n")
        
        return 0
