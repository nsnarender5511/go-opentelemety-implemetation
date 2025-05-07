# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder
WORKDIR /build

# Copy all Go module files first for better caching
COPY go.work go.work.sum ./
COPY common/go.mod common/go.sum ./common/
COPY master-store/go.mod master-store/go.sum ./master-store/
COPY product-service/go.mod product-service/go.sum ./product-service/

# Download dependencies
RUN go mod download

# Copy source code
COPY common/ ./common/
COPY master-store/ ./master-store/

# Build the application, placing the executable in the target src directory
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /master-store/src/app ./master-store/src

# Final stage
FROM alpine:latest

# Create the directory structure first
RUN mkdir -p /master-store/src

# Set the final working directory
WORKDIR /master-store/src

# Copy the built application from the builder stage into the WORKDIR
COPY --from=builder /master-store/src/app .

# Copy the data file to its location relative to the root
COPY --from=builder /build/master-store/data.json /master-store/data.json

# Install curl for potential health checks or debugging
RUN apk add --no-cache curl

EXPOSE 8083

# Define the entrypoint relative to the WORKDIR (/master-store/src)
ENTRYPOINT ["./app"]
