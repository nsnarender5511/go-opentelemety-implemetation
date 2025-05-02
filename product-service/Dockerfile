FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy workspace and module files first for dependency caching
COPY go.work go.work.sum ./
COPY common/go.mod common/go.sum ./common/
COPY product-service/go.mod product-service/go.sum ./product-service/

# Download dependencies using workspace context
# RUN go work sync # Optional: Sync workspace
RUN go mod download

# Copy all source code needed for the build
COPY common/ ./common/
COPY product-service/ ./product-service/

# Build the specific service from the workspace root
# Note: Building from /build, output path is absolute
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/product-service ./product-service/src

# --- Final Stage ---
FROM alpine:latest AS final

# Install certificates needed for potential HTTPS calls
RUN apk add --no-cache curl ca-certificates

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy only the built binary from the builder stage
COPY --from=builder /app/product-service /app/product-service
# Copy data file needed at runtime from its location in the builder stage
COPY --from=builder /build/product-service/data.json /app/data.json

RUN chown -R appuser:appgroup /app

EXPOSE 8080

USER appuser

ENTRYPOINT ["/app/product-service"] 