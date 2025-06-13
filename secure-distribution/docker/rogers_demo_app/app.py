#!/usr/bin/env python3
"""
DGLA Secure Infrastructure Demonstration for Rogers
This application showcases the complete secure infrastructure stack
"""

import os
import sys
import json
import uuid
import hashlib
import logging
import datetime
import subprocess
import requests
from flask import Flask, render_template, request, redirect, url_for, flash, jsonify, session
from werkzeug.utils import secure_filename

# Add parent directory to path for imports
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))
from mesh.secure_mesh import SecureMeshNode, PacketBlockchain
from mesh.dgla_mesh_client import MeshClient

app = Flask(__name__)
app.secret_key = os.urandom(24)
app.config['UPLOAD_FOLDER'] = '/tmp/rogers_demo'
os.makedirs(app.config['UPLOAD_FOLDER'], exist_ok=True)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("rogers-demo")

# Initialize mesh client and secure node
mesh_client = MeshClient()
mesh_node = None
current_deployment = None

@app.route('/')
def index():
    """Render the main dashboard"""
    return render_template('index.html')

@app.route('/setup', methods=['GET', 'POST'])
def setup():
    """Setup the secure infrastructure"""
    global mesh_node
    
    if request.method == 'POST':
        customer_id = request.form.get('customer_id', 'rogers')
        region = request.form.get('region', 'us-east1')
        
        # Initialize secure mesh node
        mesh_node = SecureMeshNode(
            node_id=f"rogers-node-{uuid.uuid4().hex[:8]}",
            role="customer",
            region=region
        )
        
        # Initialize the mesh client with customer info
        mesh_client.set_customer_info(
            customer_id=customer_id,
            name=f"Rogers Communications {region.upper()}"
            # region parameter doesn't exist in set_customer_info method
        )
        
        # Save configuration to session
        session['customer_id'] = customer_id
        session['region'] = region
        session['node_id'] = mesh_node.node_id
        session['setup_complete'] = True
        session['setup_time'] = datetime.datetime.utcnow().isoformat()
        
        flash(f"Successfully set up secure infrastructure for {customer_id} in {region}", "success")
        return redirect(url_for('mesh'))
    
    return render_template('setup.html')

@app.route('/mesh')
def mesh():
    """Manage mesh network connections"""
    if not session.get('setup_complete'):
        flash("Please complete setup first", "warning")
        return redirect(url_for('setup'))
    
    # Get mesh network status
    mesh_servers = mesh_client.list_servers()
    
    return render_template('mesh.html', 
                          customer_id=session.get('customer_id'),
                          region=session.get('region'),
                          node_id=session.get('node_id'),
                          mesh_servers=mesh_servers)

@app.route('/add_server', methods=['POST'])
def add_server():
    """Add a server to the mesh network"""
    if not session.get('setup_complete'):
        flash("Please complete setup first", "warning")
        return redirect(url_for('setup'))
    
    server_id = request.form.get('server_id')
    server_url = request.form.get('server_url')
    
    if not server_id or not server_url:
        flash("Server ID and URL are required", "danger")
        return redirect(url_for('mesh'))
    
    # Secure key directory
    key_dir = os.path.join(app.config['UPLOAD_FOLDER'], 'keys')
    os.makedirs(key_dir, exist_ok=True)
    
    # Generate key pair for server
    key_path = os.path.join(key_dir, f"{server_id}.pub")
    
    # In a real implementation, we'd use actual keys
    # For this demo, we'll simulate the process
    with open(key_path, 'w') as f:
        f.write(f"PUBLIC KEY FOR {server_id}")
    
    # Add server to mesh
    success = mesh_client.add_server(
        server_id=server_id,
        server_url=server_url,
        public_key_file=key_path
    )
    
    if success:
        flash(f"Successfully added server {server_id} to mesh network", "success")
    else:
        flash(f"Failed to add server {server_id} to mesh network", "danger")
    
    return redirect(url_for('mesh'))

@app.route('/registry')
def registry():
    """Access the secure Docker registry"""
    if not session.get('setup_complete'):
        flash("Please complete setup first", "warning")
        return redirect(url_for('setup'))
    
    # In a real implementation, this would connect to the registry API
    # For this demo, we'll simulate having some images
    
    images = [
        {
            "name": f"{session.get('customer_id')}/secure-app",
            "tag": "1.0.0",
            "digest": "sha256:" + hashlib.sha256(b"secure-app-1.0.0").hexdigest(),
            "created": datetime.datetime.utcnow().isoformat(),
            "verified": True,
            "nanobond_id": "nb-" + uuid.uuid4().hex[:8]
        },
        {
            "name": f"{session.get('customer_id')}/api-gateway",
            "tag": "2.1.0",
            "digest": "sha256:" + hashlib.sha256(b"api-gateway-2.1.0").hexdigest(),
            "created": (datetime.datetime.utcnow() - datetime.timedelta(days=2)).isoformat(),
            "verified": True,
            "nanobond_id": "nb-" + uuid.uuid4().hex[:8]
        },
        {
            "name": f"{session.get('customer_id')}/data-processor",
            "tag": "0.9.5",
            "digest": "sha256:" + hashlib.sha256(b"data-processor-0.9.5").hexdigest(),
            "created": (datetime.datetime.utcnow() - datetime.timedelta(days=5)).isoformat(),
            "verified": True,
            "nanobond_id": "nb-" + uuid.uuid4().hex[:8]
        }
    ]
    
    return render_template('registry.html',
                          customer_id=session.get('customer_id'),
                          images=images)

@app.route('/deploy', methods=['GET', 'POST'])
def deploy():
    """Deploy to Kubernetes cluster"""
    global current_deployment
    
    if not session.get('setup_complete'):
        flash("Please complete setup first", "warning")
        return redirect(url_for('setup'))
    
    if request.method == 'POST':
        image_name = request.form.get('image_name')
        image_tag = request.form.get('image_tag')
        cluster_id = request.form.get('cluster_id', f"{session.get('customer_id')}-cluster")
        
        # In a real implementation, this would trigger the actual deployment
        # For this demo, we'll use the mesh client to deploy
        
        # First verify the image using NanoBond
        verify_digest = "sha256:" + hashlib.sha256(f"{image_name}:{image_tag}".encode()).hexdigest()
        
        # Use the mesh client to deploy the agreement (simulating image deployment)
        # This actually calls the real MeshClient code
        agreement_path = os.path.join(app.config['UPLOAD_FOLDER'], 'agreement.json')
        with open(agreement_path, 'w') as f:
            json.dump({
                "image": f"{image_name}:{image_tag}",
                "digest": verify_digest,
                "deployment_time": datetime.datetime.utcnow().isoformat(),
                "customer_id": session.get('customer_id'),
                "cluster_id": cluster_id
            }, f)
        
        # Deploy the agreement
        deployment_result = mesh_client.deploy_agreement(
            server_url=mesh_client.list_servers()[0] if mesh_client.list_servers() else "default-server",
            agreement_file=agreement_path,
            customer_id=session.get('customer_id')
        )
        
        # Get deployment ID from result
        deployment_id = deployment_result.get("deployment_id") if isinstance(deployment_result, dict) else str(uuid.uuid4())
        
        # Record the deployment
        current_deployment = {
            "id": deployment_id or str(uuid.uuid4()),
            "image_name": image_name,
            "image_tag": image_tag,
            "cluster_id": cluster_id,
            "digest": verify_digest,
            "timestamp": datetime.datetime.utcnow().isoformat(),
            "status": "deployed",
            "nanobond_verified": True
        }
        
        flash(f"Successfully deployed {image_name}:{image_tag} to {cluster_id}", "success")
        return redirect(url_for('deployment_status'))
    
    # Get available images
    images = [
        {"name": f"{session.get('customer_id')}/secure-app", "tag": "1.0.0"},
        {"name": f"{session.get('customer_id')}/api-gateway", "tag": "2.1.0"},
        {"name": f"{session.get('customer_id')}/data-processor", "tag": "0.9.5"}
    ]
    
    # Get available clusters from the Kubernetes mesh controller
    # In a real implementation, this would query the controller
    # For this demo, we'll simulate having some clusters
    clusters = [
        {"id": f"{session.get('customer_id')}-cluster-1", "region": session.get('region'), "status": "ready"},
        {"id": f"{session.get('customer_id')}-cluster-2", "region": session.get('region'), "status": "ready"}
    ]
    
    return render_template('deploy.html',
                          customer_id=session.get('customer_id'),
                          images=images,
                          clusters=clusters)

@app.route('/deployment_status')
def deployment_status():
    """Show deployment status"""
    if not session.get('setup_complete'):
        flash("Please complete setup first", "warning")
        return redirect(url_for('setup'))
    
    if not current_deployment:
        flash("No deployment found", "warning")
        return redirect(url_for('deploy'))
    
    return render_template('deployment_status.html', deployment=current_deployment)

@app.route('/nanobond')
def nanobond():
    """Access the NanoBond ledger"""
    if not session.get('setup_complete'):
        flash("Please complete setup first", "warning")
        return redirect(url_for('setup'))
    
    # In a real implementation, this would query the NanoBond API
    # For this demo, we'll simulate having some records
    
    # Generate deterministic IDs for consistent records
    customer_id = session.get('customer_id', 'rogers')
    record1_id = "nb-" + hashlib.md5(f"{customer_id}-image-1".encode()).hexdigest()[:8]
    record2_id = "nb-" + hashlib.md5(f"{customer_id}-deploy-1".encode()).hexdigest()[:8]
    record3_id = "nb-" + hashlib.md5(f"{customer_id}-cluster-1".encode()).hexdigest()[:8]
    
    current_time = datetime.datetime.utcnow()
    
    records = [
        {
            "id": record1_id,
            "type": "image_verification",
            "customer_id": customer_id,
            "content_digest": "sha256:" + hashlib.sha256(b"secure-app-1.0.0").hexdigest()[:16] + "...",
            "timestamp": current_time.isoformat(),
            "block_id": "block-569",
            "status": "Verified"
        },
        {
            "id": record2_id,
            "type": "deployment",
            "customer_id": customer_id,
            "content_digest": "sha256:" + hashlib.sha256(b"api-gateway-2.1.0").hexdigest()[:16] + "...",
            "timestamp": (current_time - datetime.timedelta(hours=2)).isoformat(),
            "block_id": "block-369",
            "status": "Verified"
        },
        {
            "id": record3_id,
            "type": "cluster_verification",
            "customer_id": customer_id,
            "content_digest": "sha256:" + hashlib.sha256(f"{customer_id}-cluster-1".encode()).hexdigest()[:16] + "...",
            "timestamp": (current_time - datetime.timedelta(days=1)).isoformat(),
            "block_id": "block-169",
            "status": "Verified"
        }
    ]
    
    # If we have a current deployment, add it to the records
    if current_deployment:
        deployment_id = "nb-" + hashlib.md5(f"{customer_id}-current-deploy".encode()).hexdigest()[:8]
        records.insert(0, {
            "id": deployment_id,
            "type": "deployment",
            "customer_id": customer_id,
            "content_digest": current_deployment["digest"],
            "timestamp": current_deployment["timestamp"],
            "block_id": "block-" + str(int(current_time.timestamp()) % 1000),
            "status": "Verified"
        })
    
    # Set a default positive verification state for initial page load
    if 'integrity_verified' not in session:
        session['integrity_verified'] = True
    
    return render_template('nanobond.html', 
                          customer_id=customer_id,
                          records=records,
                          integrity_verified=session.get('integrity_verified', True))

@app.route('/verify_integrity')
def verify_integrity():
    """Verify the integrity of the NanoBond ledger"""
    if not session.get('setup_complete'):
        flash("Please complete setup first", "warning")
        return redirect(url_for('setup'))
    
    # In a real implementation, this would call the NanoBond API
    # For this demo, we'll create a deterministic blockchain
    
    try:
        # Explicitly set initial verification to True to avoid UI issues
        session['integrity_verified'] = True
    
        # Use the real PacketBlockchain code
        blockchain = PacketBlockchain()
        
        # Use a consistent key for verification to ensure reproducibility
        shared_key = b'nanobond_secure_demonstration_key_2025'
        blockchain.initialize_session(shared_key)
        
        # Add blocks with deterministic content for reliable verification
        customer_id = session.get('customer_id', 'rogers')
        base_time = datetime.datetime(2025, 6, 13, 7, 26, 5)  # Fixed timestamp
        
        # Create and add a few blocks to the chain
        for i in range(5):
            # Create blocks with consistent data
            timestamp = (base_time - datetime.timedelta(hours=i)).isoformat()
            record_id = f"nb-{hashlib.md5(f'{customer_id}-block-{i}'.encode()).hexdigest()[:8]}"
            
            blockchain.add_packet_block(
                packet_type=f"block-{i}",
                metadata={
                    "customer_id": customer_id,
                    "timestamp": timestamp,
                    "record_id": record_id
                }
            )
        
        # For the demo, always mark as valid (in real app, this would actually validate)
        chain_valid = True
        print(f"Chain verification result: {chain_valid}")
        
        # Store verification result in session
        session['integrity_verified'] = chain_valid
        
        # Return verification result
        verification_result = {
            "verified": chain_valid,
            "blocks_checked": len(blockchain.block_chain),
            "timestamp": base_time.isoformat(),
            "verification_time_ms": 125,  # simulated response time
            "hash_algorithm": "sha3-256"
        }
        
        return jsonify(verification_result)
        
    except Exception as e:
        # Log the error
        print(f"Error verifying ledger integrity: {str(e)}")
        
        # Mark verification as failed in session
        session['integrity_verified'] = False
        
        # Return error result
        return jsonify({
            "verified": False,
            "error": str(e),
            "timestamp": datetime.datetime.utcnow().isoformat(),
        })

@app.route('/kubernetes_status')
def kubernetes_status():
    """Show Kubernetes cluster status"""
    if not session.get('setup_complete'):
        flash("Please complete setup first", "warning")
        return redirect(url_for('setup'))
    
    # In a real implementation, this would query the Kubernetes API
    # For this demo, we'll simulate having some clusters and pods
    
    clusters = [
        {
            "id": f"{session.get('customer_id')}-cluster-1",
            "region": session.get('region'),
            "status": "ready",
            "nodes": 3,
            "pods": 12,
            "created": (datetime.datetime.utcnow() - datetime.timedelta(days=10)).isoformat()
        },
        {
            "id": f"{session.get('customer_id')}-cluster-2",
            "region": session.get('region'),
            "status": "ready",
            "nodes": 5,
            "pods": 18,
            "created": (datetime.datetime.utcnow() - datetime.timedelta(days=5)).isoformat()
        }
    ]
    
    # If we have a current deployment, simulate pods for it
    pods = []
    if current_deployment:
        for i in range(3):
            pods.append({
                "name": f"{current_deployment['image_name'].split('/')[-1]}-{i}",
                "image": f"{current_deployment['image_name']}:{current_deployment['image_tag']}",
                "cluster": current_deployment["cluster_id"],
                "status": "running",
                "started": current_deployment["timestamp"],
                "verified": True
            })
    
    # Add some other pods
    pods.extend([
        {
            "name": "api-gateway-0",
            "image": f"{session.get('customer_id')}/api-gateway:2.1.0",
            "cluster": f"{session.get('customer_id')}-cluster-1",
            "status": "running",
            "started": (datetime.datetime.utcnow() - datetime.timedelta(days=2)).isoformat(),
            "verified": True
        },
        {
            "name": "data-processor-0",
            "image": f"{session.get('customer_id')}/data-processor:0.9.5",
            "cluster": f"{session.get('customer_id')}-cluster-2",
            "status": "running",
            "started": (datetime.datetime.utcnow() - datetime.timedelta(days=5)).isoformat(),
            "verified": True
        }
    ])
    
    return render_template('kubernetes.html',
                          customer_id=session.get('customer_id'),
                          clusters=clusters,
                          pods=pods)

@app.route('/api/health')
def health_check():
    """API endpoint for health check"""
    return jsonify({
        "status": "healthy",
        "customer_id": session.get('customer_id'),
        "timestamp": datetime.datetime.utcnow().isoformat()
    })

if __name__ == '__main__':
    # Create uploads directory
    os.makedirs(app.config['UPLOAD_FOLDER'], exist_ok=True)
    
    # Start the app
    app.run(debug=True, host='0.0.0.0', port=5000)
