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

# Build the application, placing the executable in the target src directory
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /product-service/src/app ./product-service/src

# Final stage
FROM alpine:latest

# Create the directory structure first
RUN mkdir -p /product-service/src

# Set the final working directory
WORKDIR /product-service/src

# Copy the built application from the builder stage into the WORKDIR
COPY --from=builder /product-service/src/app .

# Copy the data file to its location relative to the root
COPY --from=builder /build/product-service/data.json /product-service/data.json

# Install curl for potential health checks or debugging
RUN apk add --no-cache curl

EXPOSE 8082

# Define the entrypoint relative to the WORKDIR (/product-service/src)
ENTRYPOINT ["./app"]
