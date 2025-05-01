# Variables
SERVICE_NAME := product-service
IMAGE_NAME := $(SERVICE_NAME):latest
CONTAINER_NAME := $(SERVICE_NAME)_container
SERVICE_PORT := 8082
SIMULATOR_SCRIPT := tests/simulate_product_service.py
PYTHON := python3 # Or just python if python3 isn't your command
SIGNOZ_IMAGE := signoz/signoz:latest
# Path where the user clones the signoz-install repo
SIGNOZ_INSTALL_DIR ?= .signoz # Default to a sibling directory, user can override

.PHONY: build run simulate run-signoz stop-signoz help

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
	@docker run -d --name $(CONTAINER_NAME) \
		-p $(SERVICE_PORT):$(SERVICE_PORT) \
		-e OTEL_EXPORTER_OTLP_ENDPOINT=host.docker.internal:4317 \
		-e OTEL_EXPORTER_INSECURE=true \
		$(IMAGE_NAME)
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
	@echo "Checking for existing SigNoz installation directory..."
	@if [ ! -d "$(SIGNOZ_INSTALL_DIR)" ]; then \
			echo "Cloning SigNoz installation repository into $(SIGNOZ_INSTALL_DIR)..."; \
			rm -rf "$(SIGNOZ_INSTALL_DIR)"; \
			git clone -b main https://github.com/SigNoz/signoz.git $(SIGNOZ_INSTALL_DIR); \
		else \
			echo "Existing SigNoz installation found."; \
		fi; \
		echo "--- Current directory before cd:"; pwd; \
		echo "--- Attempting to cd into $(SIGNOZ_INSTALL_DIR)/deploy/ and run install script..."; \
		cd $(SIGNOZ_INSTALL_DIR); \
		cd deploy; \
		echo "--- Current directory after cd:"; pwd; \
		echo "--- Listing contents of $(SIGNOZ_INSTALL_DIR)/deploy/:"; ls -la; \
		echo "--- Executing install script..."; bash install.sh; \
		echo "SigNoz installation script finished."; \
		echo "Access UI at http://localhost:3301 (might take a moment to start)"; \
		echo "OTLP endpoint should be available at localhost:4317 (gRPC)"

help:
	@echo "Available targets:"
	@echo "  build        : Build the Docker image for the product service"
	@echo "  run          : Build and run the product service container"
	@echo "  simulate     : Run the traffic simulation script against the running service"
	@echo "  run-signoz   : Clone (if needed) and run SigNoz using docker-compose"
	@echo "  help         : Show this help message" 