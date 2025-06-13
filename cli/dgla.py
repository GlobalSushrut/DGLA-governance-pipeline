#!/usr/bin/env python3
"""DGLA Command Line Interface - Complete System Management"""
import os
import sys
import yaml
import argparse
import subprocess
from commands.rogers_5g import Rogers5GCLI
from commands.consumer import ConsumerCLI
from commands.mesh_network import setup_mesh_parser, handle_mesh_command
from datetime import datetime

# Import Rogers 5G module
sys.path.append(os.path.dirname(os.path.realpath(__file__)))
from commands.rogers_5g import setup_parser as setup_rogers_5g_parser

VERSION = "1.0.0"

def run_cmd(cmd, check=True):
    """Run shell command and return output"""
    return subprocess.run(cmd, shell=True, check=check, text=True, 
                         stdout=subprocess.PIPE, stderr=subprocess.PIPE)

def log(msg, level="INFO"):
    """Log message with timestamp"""
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{timestamp}] {level}: {msg}")

class DglaCLI:
    def __init__(self):
        self.base_dir = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
        self.config_dir = os.path.join(self.base_dir, "config")
        self.infra_dir = os.path.join(self.base_dir, "infrastructure")
        
        # Ensure config directory exists
        os.makedirs(self.config_dir, exist_ok=True)
        
        # Initialize CLI modules
        self.rogers_5g = Rogers5GCLI(self.base_dir)
        self.consumer = ConsumerCLI(self.base_dir)
        
        # Initialize mesh network module
        self.mesh = MeshCLI(self.base_dir)
        
        # Load default configuration
        self.config = self._load_or_create_config()

    def _load_or_create_config(self):
        """Load or create default configuration"""
        config_file = os.path.join(self.config_dir, "dgla-config.yaml")
        
        if os.path.exists(config_file):
            with open(config_file, 'r') as f:
                return yaml.safe_load(f)
        
        # Default config
        config = {
            "version": VERSION,
            "environment": "development",
            "kubernetes": {
                "namespace": "dgla",
                "context": "",
            },
            "mongodb": {
                "replicas": 3,
                "merkle_enabled": True,
            },
            "cdn": {
                "enabled": True,
                "regions": ["us-east", "eu-west", "asia-east"]
            },
            "sla": {
                "enabled": True,
                "default_tier": "silver"
            },
            "vendors": []
        }
        
        # Save default config
        with open(config_file, 'w') as f:
            yaml.dump(config, f)
        
        return config

    def init(self, args):
        """Initialize DGLA environment"""
        log("Initializing DGLA environment...")
        
        # Check if kubectl is available
        if run_cmd("which kubectl", check=False).returncode != 0:
            log("kubectl not found. Please install Kubernetes CLI first.", "ERROR")
            sys.exit(1)
            
        # Check if current k8s context
        result = run_cmd("kubectl config current-context", check=False)
        if result.returncode != 0:
            log("No Kubernetes context found. Please configure kubectl first.", "ERROR")
            sys.exit(1)
            
        self.config["kubernetes"]["context"] = result.stdout.strip()
        
        # Set kubernetes namespace
        if args.namespace:
            self.config["kubernetes"]["namespace"] = args.namespace
            
        # Create namespace if it doesn't exist
        run_cmd(f"kubectl create namespace {self.config['kubernetes']['namespace']} --dry-run=client -o yaml | kubectl apply -f -")
            
        # Set environment
        if args.environment:
            self.config["environment"] = args.environment
            
        # Save updated config
        with open(os.path.join(self.config_dir, "dgla-config.yaml"), 'w') as f:
            yaml.dump(self.config, f)
            
        log(f"DGLA initialized with namespace: {self.config['kubernetes']['namespace']}, environment: {self.config['environment']}")
        log(f"Run 'dgla deploy' to deploy the system components")

    def deploy(self, args):
        """Deploy DGLA components"""
        components = args.components.split(",") if args.components else ["all"]
        
        if "all" in components:
            components = ["mongodb", "cdn", "monitoring", "node-management", "alerting", "sla", "api"]
            
        namespace = self.config["kubernetes"]["namespace"]
            
        for component in components:
            log(f"Deploying {component}...")
            
            if component == "mongodb":
                run_cmd(f"kubectl apply -f {self.infra_dir}/db/mongodb-secrets.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/db/mongodb-statefulset.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/db/mongodb-service.yaml -n {namespace}")
                
                if self.config["mongodb"]["merkle_enabled"]:
                    run_cmd(f"kubectl apply -f {self.infra_dir}/db/mongodb-merkle-implementation.yaml -n {namespace}")
                    run_cmd(f"kubectl apply -f {self.infra_dir}/db/client-db-connector.yaml -n {namespace}")
            
            elif component == "cdn" and self.config["cdn"]["enabled"]:
                run_cmd(f"kubectl apply -f {self.infra_dir}/cdn/cdn-deployment.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/cdn/cdn-service.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/cdn/cdn-config.yaml -n {namespace}")
                
            elif component == "monitoring":
                run_cmd(f"kubectl apply -f {self.infra_dir}/monitoring/prometheus-deployment.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/monitoring/prometheus-config.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/monitoring/grafana-deployment.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/monitoring/grafana-config.yaml -n {namespace}")
                
            elif component == "node-management":
                run_cmd(f"kubectl apply -f {self.infra_dir}/node-management/node-manager-secret.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/node-management/node-manager-daemonset.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/node-management/control-service-deployment.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/node-management/control-service.yaml -n {namespace}")
                
            elif component == "alerting":
                run_cmd(f"kubectl apply -f {self.infra_dir}/alerting/alertmanager-deployment.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/alerting/alertmanager-config.yaml -n {namespace}")
                
            elif component == "sla" and self.config["sla"]["enabled"]:
                run_cmd(f"kubectl apply -f {self.infra_dir}/alerting/sla-service-deployment.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/alerting/sla-service.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/alerting/sla-secrets.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/alerting/sla-config.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/alerting/sla-operator-rbac.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/alerting/sla-operator-deployment.yaml -n {namespace}")
                run_cmd(f"kubectl apply -f {self.infra_dir}/alerting/custom-sla-config.yaml -n {namespace}")
                
            elif component == "api":
                run_cmd(f"kubectl apply -f {self.infra_dir}/complete-infrastructure.yaml -n {namespace}")
                
        log("Deployment completed. Run 'dgla status' to check component status.")

    def create_sla(self, args):
        """Create a custom SLA definition"""
        sla_dir = os.path.join(self.config_dir, "sla")
        os.makedirs(sla_dir, exist_ok=True)
        
        # Basic SLA template
        sla = {
            "apiVersion": "dgla.io/v1",
            "kind": "SLADefinition",
            "metadata": {
                "name": args.name,
                "namespace": args.namespace or self.config["kubernetes"]["namespace"]
            },
            "spec": {
                "customerConfig": {
                    "enabled": True,
                    "customerName": args.customer,
                    "customerID": args.id or f"{args.name}-{datetime.now().strftime('%Y%m%d')}",
                },
                "serviceLevelTemplate": {
                    "name": args.tier or self.config["sla"]["default_tier"],
                }
            }
        }
        
        # Add custom metrics if specified
        if args.metrics:
            metrics = []
            for metric in args.metrics.split(","):
                name, value = metric.split("=")
                metrics.append({
                    "name": name,
                    "threshold": value if not value.isdigit() else int(value),
                })
            
            sla["spec"]["serviceLevelTemplate"]["customMetrics"] = metrics
        
        # Write SLA definition to file
        sla_file = os.path.join(sla_dir, f"{args.name}-sla.yaml")
        with open(sla_file, 'w') as f:
            yaml.dump(sla, f)
            
        log(f"SLA definition created at {sla_file}")
        
        # Apply SLA if requested
        if args.apply:
            run_cmd(f"kubectl apply -f {sla_file}")
            log(f"SLA '{args.name}' applied to cluster")

    def add_vendor(self, args):
        """Add a vendor container to DGLA"""
        vendor = {
            "name": args.name,
            "image": args.image,
            "version": args.version or "latest",
            "description": args.description or f"{args.name} integration for DGLA",
            "enabled": True
        }
        
        # Add vendor to config
        if not any(v["name"] == args.name for v in self.config["vendors"]):
            self.config["vendors"].append(vendor)
        else:
            # Update existing vendor
            for i, v in enumerate(self.config["vendors"]):
                if v["name"] == args.name:
                    self.config["vendors"][i] = vendor
        
        # Save updated config
        with open(os.path.join(self.config_dir, "dgla-config.yaml"), 'w') as f:
            yaml.dump(self.config, f)
            
        # Create vendor deployment
        vendor_file = os.path.join(self.config_dir, "vendors", f"{args.name}.yaml")
        os.makedirs(os.path.dirname(vendor_file), exist_ok=True)
        
        deployment = {
            "apiVersion": "apps/v1",
            "kind": "Deployment",
            "metadata": {
                "name": f"dgla-vendor-{args.name}",
                "namespace": self.config["kubernetes"]["namespace"]
            },
            "spec": {
                "replicas": args.replicas or 1,
                "selector": {
                    "matchLabels": {
                        "app": f"dgla-vendor-{args.name}"
                    }
                },
                "template": {
                    "metadata": {
                        "labels": {
                            "app": f"dgla-vendor-{args.name}"
                        }
                    },
                    "spec": {
                        "containers": [{
                            "name": args.name,
                            "image": f"{args.image}:{args.version or 'latest'}",
                            "env": [
                                {
                                    "name": "DGLA_API",
                                    "value": "http://dgla-api-service:8080" 
                                },
                                {
                                    "name": "MONGODB_URI",
                                    "value": "mongodb://dgla-mongodb:27017/vendor"
                                }
                            ]
                        }]
                    }
                }
            }
        }
        
        with open(vendor_file, 'w') as f:
            yaml.dump(deployment, f)
            
        log(f"Vendor {args.name} added to DGLA")
        
        # Deploy vendor if requested
        if args.deploy:
            run_cmd(f"kubectl apply -f {vendor_file}")
            log(f"Vendor {args.name} deployed to cluster")

    def status(self, args):
        """Check status of DGLA components"""
        namespace = self.config["kubernetes"]["namespace"]
        
        log("Checking DGLA component status...")
        run_cmd(f"kubectl get pods -n {namespace}")
        
        # Check services
        log("\nServices:")
        run_cmd(f"kubectl get svc -n {namespace}")
        
        # Check specific components if requested
        if args.component:
            log(f"\nDetailed status for {args.component}:")
            run_cmd(f"kubectl describe deployment dgla-{args.component} -n {namespace}")

    def test(self, args):
        """Run tests against DGLA deployment"""
        namespace = self.config["kubernetes"]["namespace"]
        
        log("Testing DGLA deployment...")
        
        # API health check
        log("Checking API health...")
        run_cmd("kubectl port-forward svc/dgla-api-service 8080:8080 -n " + 
               f"{namespace} > /dev/null 2>&1 & sleep 2 && " +
               "curl -s http://localhost:8080/health && kill %1")
        
        # MongoDB check
        log("\nChecking MongoDB connection...")
        run_cmd(f"kubectl exec deploy/dgla-api -n {namespace} -- " +
               "python -c 'import pymongo; print(\"Connected\" if pymongo.MongoClient(\"mongodb://dgla-mongodb:27017\").server_info() else \"Failed\")'")
        
        # Run vendor tests if available
        if args.vendor:
            vendor = next((v for v in self.config["vendors"] if v["name"] == args.vendor), None)
            if vendor:
                log(f"\nRunning tests for vendor {args.vendor}...")
                run_cmd(f"kubectl exec deploy/dgla-vendor-{args.vendor} -n {namespace} -- /bin/sh -c './run_tests.sh'")
            else:
                log(f"Vendor {args.vendor} not found", "ERROR")
                
        log("Tests completed")

    def parse_args(self):
        parser = argparse.ArgumentParser(
            description="DGLA Command Line Interface",
            formatter_class=argparse.RawDescriptionHelpFormatter,
        )
        
        subparsers = parser.add_subparsers(dest="command")
        
        # Define the init subparser
        init_parser = subparsers.add_parser("init", help="Initialize DGLA environment")
        init_parser.add_argument("--namespace", default="dgla", help="Kubernetes namespace")
        init_parser.add_argument("--environment", default="dev", help="Environment (dev, test, prod)")
        
        # Deploy command
        deploy_parser = subparsers.add_parser("deploy", help="Deploy DGLA components")
        deploy_parser.add_argument("--components", help="Comma-separated components to deploy")
        
        # Create SLA command
        sla_parser = subparsers.add_parser("create-sla", help="Create SLA definition")
        sla_parser.add_argument("name", help="SLA name")
        sla_parser.add_argument("--customer", required=True, help="Customer name")
        sla_parser.add_argument("--id", help="Customer ID")
        sla_parser.add_argument("--namespace", help="Kubernetes namespace")
        sla_parser.add_argument("--tier", choices=["platinum", "gold", "silver", "custom"], 
                              help="Service tier")
        sla_parser.add_argument("--metrics", help="Custom metrics (format: name=value,name2=value2)")
        sla_parser.add_argument("--apply", action="store_true", help="Apply SLA to cluster")
        
        # Add vendor command
        vendor_parser = subparsers.add_parser("add-vendor", help="Add vendor container")
        vendor_parser.add_argument("name", help="Vendor name")
        vendor_parser.add_argument("--image", required=True, help="Container image")
        vendor_parser.add_argument("--version", help="Container version/tag")
        vendor_parser.add_argument("--description", help="Vendor description")
        vendor_parser.add_argument("--replicas", type=int, help="Number of replicas")
        vendor_parser.add_argument("--deploy", action="store_true", help="Deploy vendor container")
        
        # Status command
        status_parser = subparsers.add_parser("status", help="Check component status")
        status_parser.add_argument("--component", help="Specific component to check")
        
        # Test command
        test_parser = subparsers.add_parser("test", help="Run tests")
        test_parser.add_argument("--vendor", help="Test specific vendor integration")
        
        # Rogers 5G integration
        setup_rogers_5g_parser(subparsers)
        
        # Add Consumer-friendly CLI command parser
        self.consumer.setup_parser(subparsers)
        
        # Add Secure Mesh Network parser
        setup_mesh_parser(subparsers)
        
        # Parse args
        args = parser.parse_args()
        
        return args

    def main(self):
        """Main CLI entry point"""
        args = self.parse_args()
        
        if args.command == "init":
            return self.init(args)
        elif args.command == "deploy":
            return self.deploy(args)
        elif args.command == "create-sla":
            return self.create_sla(args)
        elif args.command == "add-vendor":
            return self.add_vendor(args)
        elif args.command == "status":
            return self.status(args)
        elif args.command == "test":
            return self.test(args)
        elif args.command == "rogers-5g":
            # Handle Rogers 5G subcommand
            return self.rogers_5g.handle_command(args)
        elif args.command == "consumer":
            # Handle consumer-friendly commands
            return self.consumer.handle_command(args)
        elif args.command == "mesh":
            # Handle secure mesh network command
            return handle_mesh_command(args)
        elif args.command is None:
            parser = argparse.ArgumentParser()
            parser.print_help()
            return 1
        else:
            print(f"Unknown command: {args.command}")
            return 1

if __name__ == "__main__":
    DglaCLI().main()
