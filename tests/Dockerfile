# Use an official Python runtime as a parent image
FROM python:3.12-slim

# Set the working directory in the container
WORKDIR /usr/src/app

# Install requests
RUN pip install --no-cache-dir requests

# Copy the simulation script into the container
COPY simulate_product_service.py .

# Command to run the script
ENTRYPOINT ["python", "./simulate_product_service.py"] 