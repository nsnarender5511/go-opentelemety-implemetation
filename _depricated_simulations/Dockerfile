FROM python:3.12-slim

WORKDIR /usr/src/app

RUN pip install --no-cache-dir requests

# Remove old copy command if it existed
# COPY simulate_product_service.py .

# Copy the entire src directory into the container
COPY src/ ./src/

# Run the simulation script from its location within src
CMD ["python", "src/simulate.py"] 