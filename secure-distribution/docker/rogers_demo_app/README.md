# Rogers Secure Infrastructure Demo App

This application demonstrates DGLA's military-grade secure infrastructure stack for Rogers. It provides a comprehensive, real-world example of the complete secure Docker registry system integrated with NanoBond™ lightweight immutable ledger, Lockbox image immutability enforcement, and secure mesh networking components.

## Features

- **Secure Infrastructure Setup**: Customer ID, region, and military-grade security parameter initialization.
- **Mesh Network Management**: View mesh servers and manage secure communication channels.
- **Secure Docker Registry**: Browse NanoBond™-verified images with integrity protection.
- **Secure Deployment**: Deploy verified container images to Kubernetes clusters with immutability enforcement.
- **NanoBond™ Ledger Viewer**: Inspect the immutable ledger with real-time cryptographic integrity verification.
- **Kubernetes Dashboard**: Monitor cluster and pod status with security verification.

## Security Features

- Military-grade cryptographic verification using NanoBond™ technology
- Immutable image enforcement through Lockbox
- Secure mesh networking with strict mTLS enforcement
- Full cryptographic audit trail for all operations
- Real-time integrity verification

## System Requirements

- Python 3.9+
- Flask and dependencies (see requirements.txt)
- Docker (for running in containerized mode)
- Access to mesh networking components
- Local or remote access to the secure Docker registry

## Running the Demo

### Local Development

1. Make sure Python 3.9+ is installed
2. Install dependencies:
   ```
   pip install -r requirements.txt
   ```
3. Run the application:
   ```
   python app.py
   ```
4. Open a browser and navigate to: http://localhost:5000

### Docker Container

1. Build the Docker image:
   ```
   docker build -t rogers-demo-app .
   ```
2. Run the container:
   ```
   docker run -p 5000:5000 rogers-demo-app
   ```
3. Open a browser and navigate to: http://localhost:5000

## Integration with Other Components

This demo application is designed to integrate with:

1. **Secure Docker Registry**: Located in `../secure-registry`
2. **NanoBond™ Ledger**: Lightweight immutable ledger service
3. **Lockbox**: Image immutability enforcement service
4. **Secure Mesh Network**: Military-grade secure communication infrastructure

## Demo Flow

1. Start on the Setup page to initialize customer-specific parameters
2. Navigate to Mesh Network to add secure communication endpoints
3. Browse the Secure Registry to view verified images
4. Deploy an image to a Kubernetes cluster
5. Verify deployment status and NanoBond™ ledger integrity
6. Monitor the deployed application through the Kubernetes dashboard

## Notes

- This demo application simulates certain backend interactions when not connected to actual services
- For a full production deployment, ensure all related services are properly configured
- Security parameters are set to military-grade defaults and should not be downgraded

© 2025 DGLA Secure Infrastructure
