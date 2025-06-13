#!/bin/bash
# Build script for creating the secure DGLA SDK package
# This creates an immutable, cryptographically signed container
# that can be distributed securely

set -e

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
BASE_DIR="$(dirname "$SCRIPT_DIR")"
BUILD_VERSION="1.0.0"
REGISTRY="${REGISTRY:-ghcr.io/dgla}"

echo "===== DGLA Secure SDK Builder ====="
echo "Version: $BUILD_VERSION"
echo "Base directory: $BASE_DIR"

# Check if requirements.txt exists, if not create it
if [ ! -f "$BASE_DIR/requirements.txt" ]; then
    echo "Creating requirements.txt..."
    cat > "$BASE_DIR/requirements.txt" << EOL
pyyaml>=6.0
cryptography>=37.0.0
kubernetes>=24.2.0
requests>=2.28.0
pymongo>=4.1.0
EOL
fi

# Generate cryptographic keys and manifest
echo "Generating cryptographic keys and manifest..."
cd "$SCRIPT_DIR"
python3 generate_keys.py --base-dir "$BASE_DIR" --output-dir "$SCRIPT_DIR/keys" --manifest-output "$SCRIPT_DIR/keys/manifest.json"

# Build the Docker container
echo "Building secure container..."
docker build -t dgla-secure-sdk:$BUILD_VERSION -f "$SCRIPT_DIR/Dockerfile" "$BASE_DIR"

# Tag for registry if specified
if [ -n "$REGISTRY" ]; then
    echo "Tagging for registry: $REGISTRY"
    docker tag dgla-secure-sdk:$BUILD_VERSION $REGISTRY/dgla-secure-sdk:$BUILD_VERSION
    docker tag dgla-secure-sdk:$BUILD_VERSION $REGISTRY/dgla-secure-sdk:latest

    echo "To push to registry:"
    echo "docker push $REGISTRY/dgla-secure-sdk:$BUILD_VERSION"
    echo "docker push $REGISTRY/dgla-secure-sdk:latest"
fi

# Generate client installer
echo "Generating client installer..."
cat > "$SCRIPT_DIR/dgla-client-installer.sh" << 'EOL'
#!/bin/bash
# DGLA Secure SDK Client Installer
# This script installs the DGLA client wrapper that connects to the secure container

set -e

# Installation directory
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.dgla"

# Create directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"

# Create client wrapper
cat > "$INSTALL_DIR/dgla" << 'EOF'
#!/bin/bash
# DGLA Secure Client Wrapper

# Configuration
DGLA_IMAGE="${DGLA_IMAGE:-ghcr.io/dgla/dgla-secure-sdk:latest}"
CONFIG_DIR="${HOME}/.dgla"
USER_EXTENSIONS="${CONFIG_DIR}/extensions"
DEPLOYMENTS_DIR="${CONFIG_DIR}/deployments"

# Create directories if they don't exist
mkdir -p "$CONFIG_DIR"
mkdir -p "$USER_EXTENSIONS"
mkdir -p "$DEPLOYMENTS_DIR"

# Pull latest image if requested
if [ "$1" = "update" ]; then
    echo "Updating DGLA secure SDK..."
    docker pull "$DGLA_IMAGE"
    exit $?
fi

# Run the container with proper mounts
docker run --rm -it \
    -v "$CONFIG_DIR:/app/.dgla" \
    -v "$USER_EXTENSIONS:/app/user-extensions" \
    -v "$DEPLOYMENTS_DIR:/app/deployments" \
    -v "${HOME}/.kube:/root/.kube" \
    "$DGLA_IMAGE" "$@"
EOF

chmod +x "$INSTALL_DIR/dgla"

echo "DGLA Secure SDK installed successfully!"
echo "You can now use the 'dgla' command."
echo ""
echo "Example usage:"
echo "  dgla consumer deploy-agreement --server example.com:8080 --agreement-path ./my-agreement.yaml --customer-name \"My Company\""
echo ""
echo "To update the SDK:"
echo "  dgla update"
EOL

chmod +x "$SCRIPT_DIR/dgla-client-installer.sh"

# Create an init script for CI/CD pipeline
cat > "$SCRIPT_DIR/ci-pipeline.yml" << 'EOL'
# DGLA Secure SDK CI/CD Pipeline
# For GitHub Actions

name: DGLA Secure SDK Build

on:
  push:
    branches: [ main ]
    tags: [ 'v*' ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
    - uses: actions/checkout@v3
      
    - name: Set up Python
      uses: actions/setup-python@v4
      with:
        python-version: '3.9'
        
    - name: Install dependencies
      run: |
        python -m pip install --upgrade pip
        pip install cryptography pynacl toml
        
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v2
      
    - name: Login to GitHub Container Registry
      if: github.event_name != 'pull_request'
      uses: docker/login-action@v2
      with:
        registry: ghcr.io
        username: ${{ github.actor }}
        password: ${{ secrets.GITHUB_TOKEN }}
        
    - name: Generate Keys and Manifest
      run: |
        cd secure-distribution
        python generate_keys.py
        
    - name: Extract metadata
      id: meta
      uses: docker/metadata-action@v4
      with:
        images: ghcr.io/${{ github.repository }}/dgla-secure-sdk
        
    - name: Build and push
      uses: docker/build-push-action@v4
      with:
        context: .
        file: ./secure-distribution/Dockerfile
        push: ${{ github.event_name != 'pull_request' }}
        tags: ${{ steps.meta.outputs.tags }}
        labels: ${{ steps.meta.outputs.labels }}
EOL

# Create a README
cat > "$SCRIPT_DIR/README.md" << 'EOL'
# DGLA Secure SDK Distribution

This directory contains tools for creating and distributing the DGLA SDK in a secure, 
immutable format. The SDK is packaged as a Docker container with cryptographic verification
to ensure integrity.

## Key Features

- Immutable core infrastructure that cannot be modified by end users
- Cryptographic verification of all components using RSA signatures
- Blockchain-style integrity validation for all operations
- Secure deployment of agreements and business logic
- User isolation through containerization

## Usage

### Building the Secure Package

```bash
./build-secure-package.sh
```

### Installing the Client

```bash
./dgla-client-installer.sh
```

### Using the SDK

After installation, you can use the `dgla` command from anywhere:

```bash
# Deploy an agreement
dgla consumer deploy-agreement --server example.com:8080 --agreement-path ./my-agreement.yaml --customer-name "My Company"

# Deploy business logic
dgla consumer deploy-logic --server example.com:8080 --logic-path ./my-logic --agreement-id ABC123
```

## Security Model

The DGLA SDK uses a security model inspired by blockchain infrastructure:

1. Core components are immutable and cryptographically verified
2. All deployments are recorded in a tamper-evident lock file
3. Separation between infrastructure and user space
4. Zero-trust verification of all components before execution
EOL

echo "===== Build Complete ====="
echo "Secure package created successfully!"
echo ""
echo "To test locally:"
echo "  docker run --rm -it -v \"$HOME/.dgla:/app/.dgla\" dgla-secure-sdk:$BUILD_VERSION consumer --help"
echo ""
echo "To distribute to users:"
echo "  Provide them with dgla-client-installer.sh"
echo ""
echo "CI/CD pipeline configuration created in ci-pipeline.yml"
