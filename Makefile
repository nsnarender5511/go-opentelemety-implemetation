# Variables
SERVICE_NAME := product-service
IMAGE_NAME := $(SERVICE_NAME):latest
CONTAINER_NAME := $(SERVICE_NAME)_container
SERVICE_PORT := 8082
SIMULATOR_SCRIPT := tests/simulate_product_service.py
PYTHON := python3 # Or just python if python3 isn't your command
SIGNOZ_IMAGE := signoz/signoz:latest

.PHONY: build run simulate run-signoz

# Build the Docker image for the product service
build:
	@echo "Building $(IMAGE_NAME) from context root..."
	# Build from workspace root (.) to include 'common', specifying the Dockerfile path with -f
	@docker build -t $(IMAGE_NAME) -f ./$(SERVICE_NAME)/Dockerfile .

# Build and run the product service container
run: build
	@echo "Ensuring container $(CONTAINER_NAME) is stopped and removed..."
	@docker rm -f $(CONTAINER_NAME) > /dev/null 2>&1 || true
	@echo "Running $(CONTAINER_NAME) on port $(SERVICE_PORT)..."
	@docker run -d --name $(CONTAINER_NAME) -p $(SERVICE_PORT):$(SERVICE_PORT) $(IMAGE_NAME)
	@echo "Waiting for service to start..."
	@sleep 5 # Give the container a moment to start up
	@echo "Tailing logs for $(CONTAINER_NAME)... (Press Ctrl+C to stop)"
	@docker logs -f $(CONTAINER_NAME)

# Run the Python simulation script
simulate:
	@echo "Setting up Python environment (if needed)..."
	@$(PYTHON) -m pip install -q requests
	@echo "Running traffic simulation script $(SIMULATOR_SCRIPT)..."
	@$(PYTHON) $(SIMULATOR_SCRIPT)

# Pull and run the latest Signoz image (Note: This might not start the full platform)
run-signoz:
	@echo "Pulling latest Signoz image: $(SIGNOZ_IMAGE)..."
	@docker pull $(SIGNOZ_IMAGE)
	@echo "Running $(SIGNOZ_IMAGE)... (This might exit if dependencies are missing)"
	@docker run --rm -it $(SIGNOZ_IMAGE)

help:
	@echo "Available targets:"
	@echo "  build        : Build the Docker image for the product service"
	@echo "  run          : Build and run the product service container"
	@echo "  simulate     : Run the traffic simulation script against the running service"
	@echo "  run-signoz   : Pull and run the latest Signoz image"
	@echo "  help         : Show this help message" 