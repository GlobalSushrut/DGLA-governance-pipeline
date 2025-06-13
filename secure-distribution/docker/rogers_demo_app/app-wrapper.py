#!/usr/bin/env python3
"""
Rogers Demo App Wrapper
This wrapper sets up the environment for the Rogers Demo app
"""

import os
import sys
import importlib.util

# Add necessary paths
sys.path.insert(0, '/app')
sys.path.insert(0, '/app/secure-distribution')

# Create symlinks between the modules to fix imports
def setup_mesh_module():
    """Setup the mesh module for proper imports"""
    mesh_dir = '/app/mesh'
    secure_mesh_path = '/app/secure-distribution/mesh/secure_mesh.py'
    
    # Make sure mesh directory exists
    os.makedirs(mesh_dir, exist_ok=True)
    
    # Create an __init__.py file
    with open(os.path.join(mesh_dir, '__init__.py'), 'w') as f:
        pass
    
    # Copy secure_mesh.py to the mesh directory
    if os.path.exists(secure_mesh_path):
        with open(secure_mesh_path, 'r') as src:
            secure_mesh_content = src.read()
        
        with open(os.path.join(mesh_dir, 'secure_mesh.py'), 'w') as dest:
            dest.write(secure_mesh_content)
            
    # Copy dgla_mesh_client.py to the mesh directory
    dgla_client_path = '/app/secure-distribution/mesh/dgla_mesh_client.py'
    if os.path.exists(dgla_client_path):
        with open(dgla_client_path, 'r') as src:
            dgla_client_content = src.read()
        
        # Modify the import to use relative import
        dgla_client_content = dgla_client_content.replace(
            'from secure_mesh import SecureMeshNode', 
            'from .secure_mesh import SecureMeshNode'
        )
        
        with open(os.path.join(mesh_dir, 'dgla_mesh_client.py'), 'w') as dest:
            dest.write(dgla_client_content)
            
    print("Mesh module setup complete")

# Run the setup
setup_mesh_module()

# Import and run the app
app_path = '/app/secure-distribution/docker/rogers_demo_app/app.py'

if os.path.exists(app_path):
    # Import app.py as a module
    spec = importlib.util.spec_from_file_location('app', app_path)
    app_module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(app_module)
    
    # If the app contains the 'app' Flask object, then run it
    if hasattr(app_module, 'app') and hasattr(app_module.app, 'run'):
        print("Starting Rogers Demo App...")
        app_module.app.run(debug=True, host='0.0.0.0', port=5000)
    else:
        print("Error: app.py doesn't contain a Flask app object")
else:
    print(f"Error: Could not find {app_path}")
