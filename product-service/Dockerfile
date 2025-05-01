# Stage 1: Build the application
# Use a specific Go version Alpine image for smaller size
FROM golang:1.24-alpine AS builder
LABEL stage=builder

# Set necessary environment variables
ENV CGO_ENABLED=0 GOOS=linux GOPROXY=direct
# Set the working directory **outside** GOPATH to ensure module mode
WORKDIR /app

# Install git for go mod tidy/download
RUN apk add --no-cache git

# --- Go Modules Handling ---
# Copy go.mod and go.sum files first to leverage Docker cache
# IMPORTANT: Assumes the Docker build context is the ROOT directory (Signoz_assignment/)
COPY common/go.mod common/go.sum ./common/
COPY product-service/go.mod product-service/go.sum ./product-service/

# Create a dummy go.work file to satisfy Go tools if needed, or rely on replace
# RUN go work init ./common ./product-service

# Copy the rest of the source code for common module
COPY common/ ./common/
# Copy the rest of the source code for product-service
COPY product-service/ ./product-service/

# --- Add replace directive needed for build ---
# This tells the Go build to use the local common module
# Make sure the path matches the copied structure
RUN echo "replace github.com/narender/common => ../common" >> ./product-service/go.mod
RUN cat ./product-service/go.mod # Verify replace was added (optional)

# Tidy dependencies within the service context
# WORKDIR /app/product-service # <- Keep this commented or remove if not needed before build
# RUN go mod tidy # Tidy should happen relative to the go.mod file
# RUN go mod download

# Change WORKDIR to where the main Go files are
WORKDIR /app/product-service/src

# Build the static binary
# Use -ldflags "-s -w" to strip debug symbols and reduce binary size
# Build from the src directory, outputting to /product-service
RUN go build -ldflags="-s -w" -o /product-service .

# Stage 2: Runtime image
# Use a minimal distroless static image OR alpine
# FROM gcr.io/distroless/static-debian11
FROM alpine:latest
LABEL stage=runtime

WORKDIR /app

# Copy the static binary from the builder stage
COPY --from=builder /product-service /app/product-service

# Copy data file into the image (Alternatively, use a volume)
# Ensure the path matches DATA_FILE_PATH env var used in compose/config
COPY ./product-service/data.json /app/data.json

# Expose the port the application listens on (matches PRODUCT_SERVICE_PORT env var)
EXPOSE 8082

# Set the entrypoint command
ENTRYPOINT ["/app/product-service"] 