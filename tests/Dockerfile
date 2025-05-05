FROM python:3.12-slim

WORKDIR /usr/src/app

RUN pip install --no-cache-dir requests

COPY simulate_product_service.py .
COPY . .

# Run the simulation script
# CMD ["python", "simulate_product_service.py"]
CMD ["python", "simulate.py"] 