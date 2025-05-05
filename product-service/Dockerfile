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
RUN apk add --no-cache curl 
EXPOSE 8082
ENTRYPOINT ["/app/product-service"]
