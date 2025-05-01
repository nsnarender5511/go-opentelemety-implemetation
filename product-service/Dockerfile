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
FROM alpine:latest


WORKDIR /app

# Copy the static binary from the builder stage
COPY --from=builder /app/product-service-binary .

# Copy necessary static assets like configuration or data files
# Ensure this path matches where the application expects it
COPY --from=builder /app/product-service/data.json ./data.json

# Expose the port the application listens on
# The actual port number should be configured via environment variables at runtime
EXPOSE 8082

# Set the entrypoint to the binary
ENTRYPOINT ["/app/product-service-binary"]

# CMD is often omitted when ENTRYPOINT is used like this,
# but could be used for default arguments if needed. 