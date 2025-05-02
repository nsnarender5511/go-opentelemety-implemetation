FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY product-service/go.mod product-service/go.sum ./product-service/

WORKDIR /build/product-service

RUN go mod download

COPY product-service/src ./src/
COPY product-service/data.json ./data.json

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/product-service ./src

FROM golang:1.24-alpine

RUN apk add --no-cache curl ca-certificates

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/product-service /app/product-service
COPY --from=builder /build/product-service/data.json /app/data.json

RUN chown -R appuser:appgroup /app

EXPOSE 8080

USER appuser

ENTRYPOINT ["/app/product-service"] 