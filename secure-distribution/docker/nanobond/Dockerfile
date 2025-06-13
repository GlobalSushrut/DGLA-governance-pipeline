FROM python:3.9-slim

# Security hardening - use non-root user
RUN groupadd -r nanobond && useradd -r -g nanobond -s /bin/false nanobond

# Set working directory
WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt && \
    pip install gunicorn uvicorn && \
    rm -rf /root/.cache

# Copy application code
COPY nanobond.py .
COPY entrypoint.sh .

# Create data directory
RUN mkdir -p /data && chown -R nanobond:nanobond /data

# Set permissions
RUN chmod +x entrypoint.sh

# Security hardening
RUN chmod 700 /app && \
    chmod 400 nanobond.py && \
    chown -R nanobond:nanobond /app

# Switch to non-root user
USER nanobond

# Expose port
EXPOSE 9090

# Set entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]
