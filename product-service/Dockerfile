# ---- Builder Stage ----
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go.work first to leverage dependency caching
COPY go.work go.work.sum* ./
# Copy module files for common and product-service
COPY common/go.mod common/go.sum ./common/
COPY product-service/go.mod product-service/go.sum ./product-service/

# Download dependencies based on workspace
RUN go mod download

# Copy the rest of the source code
# Ensure .dockerignore excludes unnecessary files
COPY common/ ./common/
COPY product-service/ ./product-service/

# Build the application statically
# Build from within product-service context
WORKDIR /app/product-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o /app/product-service-binary ./src

# ---- Final Stage ----
# Use distroless static image for minimal footprint
# FROM gcr.io/distroless/static-debian11
# Using alpine as distroless might cause issues with OTLP endpoint resolution (host.docker.internal)
FROM alpine:3.18

# Define build arguments for service identity
ARG SERVICE_NAME=product-service
ARG SERVICE_VERSION=0.1.0

WORKDIR /app

# Copy necessary files from the builder stage
COPY --from=builder /app/product-service-binary /app/product-service-binary
# Ensure data.json is copied relative to WORKDIR /app
COPY --from=builder /app/product-service/data.json ./data.json

# Set environment variable for data path (default)
ENV DATA_FILE_PATH=./data.json

# Expose the service port (should match config)
EXPOSE 8082

# Set environment variables from build arguments for OTel resource
ENV SERVICE_NAME=$SERVICE_NAME
# Explicitly set OTel var too
ENV OTEL_SERVICE_NAME=$SERVICE_NAME
ENV SERVICE_VERSION=$SERVICE_VERSION

# Set the entrypoint
ENTRYPOINT ["/app/product-service-binary"]

# CMD is often omitted when ENTRYPOINT is used like this,
# but could be used for default arguments if needed. 