# ---- Builder Stage ----
# Use a specific Go version matching your development environment
FROM golang:1.24-alpine AS builder

# Set working directory inside the builder stage
WORKDIR /build

# Install build tools if needed (e.g., git for private repos)
# RUN apk add --no-cache git

# --- Dependency Resolution ---
# Copy ONLY the necessary go.mod/go.sum files first to leverage build cache
COPY common/go.mod common/go.sum ./common/
COPY product-service/go.mod product-service/go.sum ./product-service/

# Create the necessary directory structure *required* by the replace directive
# This allows `go mod download` within product-service to find `../common`
# Needs to mirror the relative path in product-service/go.mod's replace directive
RUN mkdir -p ./common

# Change to the product-service directory. This respects the replace directive.
WORKDIR /build/product-service

# Download dependencies. This should fetch external dependencies for both
# product-service and the replaced common module.
# Use "go mod vendor" if you prefer vendoring dependencies
RUN go mod download
# RUN go mod vendor # Alternative: if vendoring

# --- Source Code Copy ---
# Now copy the *source code* needed for the build
# Copy common module source (still needed due to local replace)
WORKDIR /build
COPY common/ ./common/
# Copy product-service source
COPY product-service/src ./product-service/src
# Optionally copy other necessary files like templates, assets etc. for product-service

# --- Build ---
WORKDIR /build/product-service
# Build the application binary within the product-service context
# Use CGO_ENABLED=0 for static linking (good for Alpine)
# Use -ldflags="-w -s" to strip debug symbols for smaller binary
# Output the binary to a known location for the final stage
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/product-service ./src
# If using vendor: go build -mod=vendor ...

# ---- Final Stage ----
# Use distroless nonroot image for minimal attack surface and security
FROM gcr.io/distroless/static:nonroot

# Distroless images contain only the application and minimal OS libraries.
# They don't include package managers or shells.
# ca-certificates and tzdata are typically included in base images or handled differently.
# We assume the base distroless image has necessary certificates.

# Set working directory (good practice, even if minimal)
WORKDIR /app

# Copy ONLY the compiled binary from the builder stage
# Ensure the binary is copied to a location accessible by the nonroot user
# distroless/static:nonroot runs as user 65532 ('nonroot')
COPY --from=builder --chown=nonroot:nonroot /app/product-service /app/product-service

# DO NOT copy data.json here. It will be mounted as a volume via docker-compose.

# Expose the port the application listens on (matches PRODUCT_SERVICE_PORT env var)
# This is informational; the actual port comes from PRODUCT_SERVICE_PORT env var in compose/env
EXPOSE 8082

# Set the entrypoint to run the binary
# Use ENTRYPOINT exec form to handle signals properly
ENTRYPOINT ["/app/product-service"]

# Remove the commented-out USER section as distroless handles it.
# # Optional: Define a non-root user for better security
# # RUN addgroup -S appgroup && adduser -S appuser -G appgroup
# # USER appuser 