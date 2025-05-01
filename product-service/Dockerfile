# ---- Build Stage ----
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy common module first
COPY common/ ./common/

# Copy the entire product service code
COPY product-service/ ./product-service/

# Change working directory INTO product-service
WORKDIR /app/product-service

# Download dependencies (replace directive '../common' now points to /app/common)
RUN go mod download

# Build the application from within /app/product-service
# Output the binary one level up into /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/main ./src

# ---- Run Stage ----
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Expose the default port the service runs on
EXPOSE 8082

# Set default environment variables
ENV PRODUCT_SERVICE_PORT=8082

# Command to run the executable
CMD ["/app/main"] 