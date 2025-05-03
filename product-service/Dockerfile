# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder
WORKDIR /build

# Copy Go module files first for better caching
COPY go.work go.work.sum ./
COPY common/go.mod common/go.sum ./common/
COPY product-service/go.mod product-service/go.sum ./product-service/

# Download dependencies
RUN go mod download

# Copy source code
COPY common/ ./common/
COPY product-service/ ./product-service/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/product-service ./product-service/src

# Copy data file to expected location
COPY product-service/data.json /app/data.json

WORKDIR /app

# Run directly from builder stage for debugging
EXPOSE 8082
ENTRYPOINT ["/app/product-service"]

# Stage 2: Create the final lightweight image - COMMENTED OUT FOR DEBUGGING
# FROM alpine:latest
# 
# # Install necessary packages (curl for healthcheck, ca-certificates for HTTPS)
# RUN apk add --no-cache curl ca-certificates
# 
# # Create a non-root user and group
# RUN addgroup -S appgroup && adduser -S appuser -G appgroup
# 
# # Set working directory
# WORKDIR /app
# 
# # Copy the built binary from the builder stage
# COPY --from=builder /app/product-service /app/product-service
# 
# # Copy the data file
# COPY --from=builder /app/data.json /app/data.json
# 
# # Set permissions for the non-root user
# RUN chown -R appuser:appgroup /app
# 
# # Switch to the non-root user
# USER appuser
# 
# # Expose the port the application listens on
# EXPOSE 8082
# 
# # Define the entry point for the container
# ENTRYPOINT ["/app/product-service"]
# 
# # No CMD needed as entrypoint is the binary itself 