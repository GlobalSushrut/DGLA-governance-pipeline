FROM python:3.9-slim

WORKDIR /app

# Install required packages
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY dgla_adapter.py run_security_scenarios.py ./

# Create a directory for logs
RUN mkdir -p /app/logs

# Set execution permissions
RUN chmod +x dgla_adapter.py

# Command to run the adapter
CMD ["python", "dgla_adapter.py"]
