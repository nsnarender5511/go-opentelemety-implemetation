# Variables
SERVICE_NAME := product-service
IMAGE_NAME := $(SERVICE_NAME):latest
CONTAINER_NAME := $(SERVICE_NAME)_container
SERVICE_PORT := 8082
SIMULATOR_SCRIPT := tests/simulate_product_service.py
# Simulator Docker settings
SIMULATOR_IMAGE_NAME := simulator:latest
SIMULATOR_DOCKERFILE := tests/Dockerfile
SIMULATOR_CONTEXT := tests
PYTHON := python3 # Or just python if python3 isn't your command
SIGNOZ_IMAGE := signoz/signoz:latest
# Path where the user clones the signoz-install repo
SIGNOZ_INSTALL_DIR ?= .signoz # Default to a sibling directory, user can override
NETWORK_NAME=signoz-net

.PHONY: build run simulate run-signoz stop-signoz help run-local network build-simulator

# Build the Docker image for the product service
build:
	@echo "Building $(IMAGE_NAME) from context root..."
	# Build from workspace root (.) to include 'common', specifying the Dockerfile path with -f
	@docker build -t $(IMAGE_NAME) -f ./$(SERVICE_NAME)/Dockerfile .

# Build the Docker image for the simulator
build-simulator:
	@echo "Building simulator image $(SIMULATOR_IMAGE_NAME)..."
	@docker build -t $(SIMULATOR_IMAGE_NAME) -f $(SIMULATOR_DOCKERFILE) $(SIMULATOR_CONTEXT)

# Build and run the product service container
run: build network
	@echo "Running $(SERVICE_NAME)..."
	@docker run --rm -p $(SERVICE_PORT):$(SERVICE_PORT) \
		--network $(NETWORK_NAME) \
		-e PRODUCT_SERVICE_PORT=$(SERVICE_PORT) \
		-e LOG_LEVEL=info \
		-e LOG_FORMAT=text \
		-e OTEL_SERVICE_NAME=$(SERVICE_NAME) \
		-e SERVICE_VERSION=0.1.0 \
		-e OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317 \
		-e OTEL_EXPORTER_INSECURE=true \
		-e OTEL_SAMPLE_RATIO=1.0 \
		--name $(CONTAINER_NAME) \
		$(IMAGE_NAME)
	@echo "Waiting for service to start..."
	@sleep 5 # Give the container a moment to start up
	@echo "Tailing logs for $(CONTAINER_NAME)... (Press Ctrl+C to stop)"
	@docker logs -f $(CONTAINER_NAME)

# Run the Python simulation script inside multiple Docker containers concurrently
simulate: build-simulator network
	@echo "Running traffic simulation script $(SIMULATOR_SCRIPT) inside 4 Docker containers concurrently..."
	@for i in 1 2 3 4; do \
		echo "Starting simulator container $$i..."; \
		docker run --rm --name simulator_$$i --network $(NETWORK_NAME) \
			-e PRODUCT_SERVICE_URL=http://$(CONTAINER_NAME):$(SERVICE_PORT) \
			$(SIMULATOR_IMAGE_NAME) & \
	done
	@echo "Waiting for simulator containers to finish..."
	@wait # Wait for all background jobs launched by this target to complete
	@echo "All simulator containers finished."

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

# Run the Go application directly (for local development)
run-local:
	@echo "Running $(SERVICE_NAME) locally with go run..."
	@export SERVICE_NAME?=$(SERVICE_NAME); \
	 export SERVICE_VERSION?=0.1.0; \
	 export PRODUCT_SERVICE_PORT?=$(SERVICE_PORT); \
	 export OTEL_EXPORTER_OTLP_ENDPOINT?=host.docker.internal:4317; \
	 export OTEL_EXPORTER_INSECURE?=true; \
	 export LOG_LEVEL?=info; \
	 export LOG_FORMAT?=text; \
	 export OTEL_SAMPLE_RATIO?=1.0; \
	 cd ./$(SERVICE_NAME)/src && go run . # Run from service source directory

help:
	@echo "Available targets:"
	@echo "  build          : Build the Docker image for the product service"
	@echo "  build-simulator: Build the Docker image for the traffic simulator"
	@echo "  run            : Build and run the product service container"
	@echo "  simulate       : Build and run the traffic simulator container"
	@echo "  run-signoz     : Clone (if needed) and run SigNoz using docker-compose"
	@echo "  run-local      : Run the product service locally using go run"
	@echo "  help           : Show this help message"

.PHONY: network
network:
	@docker network inspect $(NETWORK_NAME) >/dev/null 2>&1 || \
		docker network create $(NETWORK_NAME) 