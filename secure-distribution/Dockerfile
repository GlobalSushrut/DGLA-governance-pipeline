FROM python:3.9-slim

# Set work directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    gnupg \
    apt-transport-https \
    ca-certificates \
    software-properties-common \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install kubectl
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
    && chmod +x kubectl \
    && mv kubectl /usr/local/bin/

# Copy core files (immutable SDK core)
COPY ./cli /app/cli/
COPY ./infrastructure /app/infrastructure/
COPY ./secure-distribution/dgla-container-entrypoint.py /app/
COPY ./secure-distribution/keys /app/keys/

# Install Python dependencies
COPY ./requirements.txt /app/
RUN pip install --no-cache-dir -r requirements.txt
RUN pip install pynacl toml pyOpenSSL cryptography

# Set permissions and create required directories
RUN mkdir -p /app/.dgla /app/user-extensions /app/deployments
RUN chmod 500 /app/cli  # Read+execute only
RUN chmod 500 /app/infrastructure  # Read+execute only
RUN chmod 500 /app/keys  # Read+execute only

# Default command
ENTRYPOINT ["python", "/app/dgla-container-entrypoint.py"]
