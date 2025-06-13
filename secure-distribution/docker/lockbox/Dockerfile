FROM python:3.9-slim

# Security hardening - use non-root user
RUN groupadd -r lockbox && useradd -r -g lockbox -s /bin/false lockbox

# Set working directory
WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt && \
    pip install gunicorn uvicorn && \
    rm -rf /root/.cache

# Copy application code
COPY lockbox.py .
COPY entrypoint.sh .

# Create directories
RUN mkdir -p /app/config /app/logs /app/data && \
    chown -R lockbox:lockbox /app

# Set permissions
RUN chmod +x entrypoint.sh

# Security hardening
RUN chmod 700 /app/config && \
    chmod 400 lockbox.py && \
    chown -R lockbox:lockbox /app

# Switch to non-root user
USER lockbox

# Expose port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]
