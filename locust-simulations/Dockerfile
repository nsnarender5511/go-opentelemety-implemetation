FROM python:3.9-slim

WORKDIR /app

# Install system dependencies for building Python packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libc6-dev \
    libev-dev \
    make \
    && rm -rf /var/lib/apt/lists/*

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the simulation code
COPY . .

# Make script executable
RUN chmod +x scripts/run_tests.sh

# Expose the Locust web interface port
EXPOSE 8089

# Default command will be overridden by docker-compose 