FROM python:3.9-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the simulation code
COPY . .

# Make script executable
RUN chmod +x scripts/run_tests.sh

# Default command runs in headless mode with environment variables
CMD ["./scripts/run_tests.sh", "headless", "10", "10", "300", "${PRODUCT_SERVICE_URL}", "stages"] 