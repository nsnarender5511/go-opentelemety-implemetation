# Variables
SERVICE_NAME := product-service
IMAGE_NAME := $(SERVICE_NAME):latest
CONTAINER_NAME := $(SERVICE_NAME)_container
SERVICE_PORT := 8082
SIMULATOR_SCRIPT := tests/simulate_product_service.py
PYTHON := python3 # Or just python if python3 isn't your command
SIGNOZ_DEPLOY_DIR := signoz/deploy

.PHONY: build run simulate run-signoz help

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

# Clone Signoz repo (if needed) and deploy using the official install script
run-signoz:
	@if [ ! -d "$(SIGNOZ_DEPLOY_DIR)" ]; then \
		echo "Cloning Signoz repository (tag v0.81.0)..."; git clone -b v0.81.0 --depth 1 https://github.com/SigNoz/signoz.git; \
	fi
	@echo "Deploying Signoz using official install script..."
	@cd $(SIGNOZ_DEPLOY_DIR) && ./install.sh
	@echo "Signoz deployment script finished."
	@echo "Access frontend at http://localhost:3301 (if deployment succeeded)"

help:
	@echo "Available targets:"
	@echo "  build        : Build the Docker image for the product service"
	@echo "  run          : Build and run the product service container"
	@echo "  simulate     : Run the traffic simulation script against the running service"
	@echo "  run-signoz   : Deploy Signoz using the official install script"
	@echo "  help         : Show this help message" 