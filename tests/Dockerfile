FROM python:3.12-slim

WORKDIR /usr/src/app

RUN pip install --no-cache-dir requests

COPY tests/simulate_product_service.py .

ENTRYPOINT ["python", "./simulate_product_service.py"] 